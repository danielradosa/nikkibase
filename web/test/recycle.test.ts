import { test } from 'node:test'
import assert from 'node:assert/strict'
import { MAX_BACKOFF, MEMORY_LIMIT, crashError, recycle, relay, wardrobeAfter, wasmMemory } from '../src/recycle.ts'

const V = '2026-09-24'
const SMALL = 8 * 1024 * 1024
const STAGE = { weights: [1, 2, 3, 4, 5], attrs: [0, 1, 2, 3, 4], scope: 'wardrobe' }
const TRAP = { ok: false, error: 'RuntimeError: unreachable', fault: true }
const OFFLINE = { ok: false, error: 'TypeError: Failed to fetch' }
const CRASHED = 'The engine crashed and was restarted — pick the stage again.'

function crashed(detail: RegExp) {
  return (e: unknown) => e instanceof Error && e.message === CRASHED && detail.test(String(e.cause))
}

type Message = { id: number; type: string; payload?: any }

class FakeWorker {
  sent: Message[] = []
  answered = new Set<number>()
  terminated = false
  onmessage: ((e: { data: unknown }) => void) | null = null
  onerror: ((e: { message: string }) => void) | null = null

  postMessage(message: Message) {
    if (this.terminated) throw new Error(`${message.type} was posted to a terminated worker`)
    this.sent.push(message)
  }

  terminate() {
    this.terminated = true
  }

  waiting() {
    return this.sent.filter((m) => !this.answered.has(m.id))
  }

  reply(fields: Record<string, unknown> = {}) {
    const [next] = this.waiting()
    this.answered.add(next.id)
    this.onmessage?.({ data: { id: next.id, ok: true, result: null, mem: SMALL, ...fields } })
    return next
  }
}

const tick = () => new Promise((resolve) => setImmediate(resolve))
const shape = (w: FakeWorker) => w.waiting().map((m) => [m.type, m.payload])
const stage = (label: string) => ({ ...STAGE, label })
const labels = (w: FakeWorker) => w.waiting().map((m) => m.payload.label)

function track<T>(promise: Promise<T>) {
  const state: { settled: boolean; value?: T; error?: Error } = { settled: false }
  promise.then(
    (value) => Object.assign(state, { settled: true, value }),
    (error) => Object.assign(state, { settled: true, error }),
  )
  return state
}

async function started() {
  const workers: FakeWorker[] = []
  const send = relay(() => {
    const w = new FakeWorker()
    workers.push(w)
    return w as never
  })
  const init = send('init', { version: V })
  workers[0].reply()
  await init
  return { workers, send }
}

async function owning(ids: number[]) {
  const h = await started()
  const set = h.send('setWardrobe', { ids })
  h.workers[0].reply({ result: { items: ids.length, unresolved: 0, known: ids.length } })
  await set
  return h
}

async function handover(running: string[]) {
  const h = await owning([5, 6])
  const [old] = h.workers
  const big = h.send('best', stage('big'))
  old.reply({ result: 'big', mem: MEMORY_LIMIT })
  await big
  const fresh = h.workers[1]
  const calls = running.map((label) => track(h.send('best', stage(label))))
  fresh.reply()
  await tick()
  fresh.reply()
  await tick()
  assert.deepEqual(labels(old), running)
  assert.deepEqual(fresh.waiting(), [])
  return { ...h, old, fresh, calls }
}

async function full(send: ReturnType<typeof relay>, worker: FakeWorker) {
  const call = send('best', STAGE)
  worker.reply({ result: 'full', mem: MEMORY_LIMIT })
  await call
}

test('the limit is 512 MB', () => {
  assert.equal(MEMORY_LIMIT, 512 * 1024 * 1024)
})

test('a reply below the limit keeps the worker, answered or refused', () => {
  assert.equal(recycle({ id: 1, ok: true, mem: SMALL }), 'keep')
  assert.equal(recycle({ id: 1, ok: true, mem: MEMORY_LIMIT - 1 }), 'keep')
  assert.equal(recycle({ id: 1, ok: true }), 'keep')
  assert.equal(recycle({ id: 1, ok: false, error: 'Error: keystream not loaded', mem: SMALL }), 'keep')
})

test('a reply at or over the limit asks for a fresh worker, answered or refused', () => {
  assert.equal(recycle({ id: 1, ok: true, mem: MEMORY_LIMIT }), 'full')
  assert.equal(recycle({ id: 1, ok: true, mem: 2 * MEMORY_LIMIT }), 'full')
  assert.equal(recycle({ id: 1, ok: false, error: 'Error: not a selections file', mem: MEMORY_LIMIT }), 'full')
})

test('a trapped or exited engine is dead, whatever its memory', () => {
  for (const error of ['RuntimeError: unreachable', 'Error: Go program has already exited', 'Error: the engine gave no reply']) {
    assert.equal(recycle({ id: 1, ok: false, error, fault: true, mem: SMALL }), 'dead')
    assert.equal(recycle({ id: 1, ok: false, error, fault: true, mem: MEMORY_LIMIT }), 'dead')
  }
})

test('a crash tells the player what to do again and keeps the engine\'s own words as the cause', () => {
  const pick = crashError('best', 'RuntimeError: unreachable')
  assert.equal(pick.message, 'The engine crashed and was restarted — pick the stage again.')
  assert.equal(pick.cause, 'RuntimeError: unreachable')
  for (const type of ['decode', 'selections']) {
    assert.equal(crashError(type, 'x').message, 'The engine crashed and was restarted — import your wardrobe again.')
  }
  assert.equal(crashError('setWardrobe', 'x').message, 'The engine crashed and was restarted — try again.')
})

test('the wardrobe to replay is the one set, or the one an import produced', () => {
  assert.deepEqual(wardrobeAfter({ id: 1, type: 'setWardrobe', payload: { ids: [1, 2] } }, { items: 2 }), [1, 2])
  assert.deepEqual(wardrobeAfter({ id: 1, type: 'decode', payload: { text: 'x' } }, { ids: [3] }), [3])
  assert.deepEqual(wardrobeAfter({ id: 1, type: 'selections', payload: { text: '@SEL' } }, { ids: [4] }), [4])
  assert.equal(wardrobeAfter({ id: 1, type: 'best', payload: STAGE }, { score: 1 }), undefined)
  assert.equal(wardrobeAfter({ id: 1, type: 'init', payload: { version: V } }, null), undefined)
})

test('the memory is found under TinyGo\'s export name and under Go\'s', () => {
  const memory = new WebAssembly.Memory({ initial: 2 })
  assert.equal(wasmMemory({ memory }), memory)
  assert.equal(wasmMemory({ mem: memory }), memory)
  assert.equal(wasmMemory({}), undefined)
  assert.equal(wasmMemory({ memory: () => 0 }), undefined)
  assert.equal(memory.buffer.byteLength, 131072)
})

test('calls go to one worker and each is answered with its own result', async () => {
  const { workers, send } = await started()
  const a = send('best', STAGE)
  const b = send('best', { ...STAGE, scope: 'all' })
  workers[0].reply({ result: 'a' })
  workers[0].reply({ result: 'b' })
  assert.equal(await a, 'a')
  assert.equal(await b, 'b')
  assert.equal(workers.length, 1)
})

test('at the limit the reply is delivered, then a fresh worker boots with the same version and wardrobe', async () => {
  const { workers, send } = await owning([5, 6])
  const [old] = workers
  const refused = send('selections', { text: 'nope' })
  old.reply({ ok: false, error: 'Error: not a selections file' })
  await assert.rejects(refused, /not a selections file/)
  const big = send('best', STAGE)
  old.reply({ result: 'big', mem: MEMORY_LIMIT })
  assert.equal(await big, 'big')
  assert.equal(workers.length, 2)
  const fresh = workers[1]
  assert.deepEqual(shape(fresh), [['init', { version: V }]])

  const during = send('best', STAGE)
  assert.deepEqual(old.waiting().map((m) => m.type), ['best'])
  fresh.reply()
  await tick()
  assert.deepEqual(shape(fresh), [['setWardrobe', { ids: [5, 6] }]])
  fresh.reply({ result: { items: 2 } })
  await tick()

  const after = send('best', STAGE)
  assert.equal(fresh.waiting().length, 0, 'nothing reaches the fresh worker before the old one has settled')
  assert.equal(old.waiting().length, 1, 'nothing new reaches the old worker once the fresh one is ready')
  assert.equal(old.terminated, false)

  old.reply({ result: 'during' })
  assert.equal(await during, 'during')
  assert.equal(old.terminated, true)
  assert.deepEqual(fresh.waiting().map((m) => m.type), ['best'])
  fresh.reply({ result: 'after' })
  assert.equal(await after, 'after')

  const later = send('best', STAGE)
  fresh.reply({ result: 'later' })
  assert.equal(await later, 'later')
  assert.equal(workers.length, 2)
  const ids = [...old.sent, ...fresh.sent].map((m) => m.id)
  assert.equal(new Set(ids).size, ids.length, 'no request is sent twice')
})

test('a wardrobe changed while the fresh worker boots is replayed again before the switch', async () => {
  const { workers, send } = await owning([5, 6])
  const [old] = workers
  const big = send('best', STAGE)
  old.reply({ result: 'big', mem: MEMORY_LIMIT })
  await big
  const fresh = workers[1]
  fresh.reply()
  await tick()

  const decode = send('decode', { text: 'clothes' })
  old.reply({ result: { items: 1, unresolved: 0, known: 1, ids: [9] } })
  await decode
  fresh.reply()
  await tick()
  assert.deepEqual(shape(fresh), [['setWardrobe', { ids: [9] }]])

  const after = send('best', STAGE)
  assert.equal(fresh.waiting().length, 1)
  fresh.reply()
  await tick()
  assert.deepEqual(fresh.waiting().map((m) => m.type), ['best'])
  fresh.reply({ result: 'after' })
  assert.equal(await after, 'after')
  assert.deepEqual(
    fresh.sent.map((m) => m.type),
    ['init', 'setWardrobe', 'setWardrobe', 'best'],
  )
})

test('a trapped call reports its own error, and the calls behind it go to a fresh worker in order', async () => {
  const { workers, send } = await owning([5, 6])
  const [old] = workers
  const bad = send('best', { ...STAGE, weights: [2 ** 31] })
  const queued = send('best', STAGE)
  old.reply({ ok: false, error: 'RuntimeError: unreachable', fault: true })
  await assert.rejects(bad, crashed(/RuntimeError: unreachable/))
  assert.equal(old.terminated, true)
  assert.equal(workers.length, 2)

  const next = send('best', { ...STAGE, scope: 'all' })
  const fresh = workers[1]
  fresh.reply()
  await tick()
  fresh.reply()
  await tick()
  assert.deepEqual(shape(fresh), [
    ['best', STAGE],
    ['best', { ...STAGE, scope: 'all' }],
  ])
  fresh.reply({ result: 'queued' })
  fresh.reply({ result: 'next' })
  assert.equal(await queued, 'queued')
  assert.equal(await next, 'next')
})

test('a reply the trapped worker sends after its fault does not answer the call it handed on', async () => {
  const { workers, send } = await owning([5])
  const [old] = workers
  const bad = send('best', stage('bad'))
  const handed = track(send('best', stage('handed')))
  old.reply(TRAP)
  await assert.rejects(bad, crashed(/unreachable/))
  const late = old.reply({ result: 'stale' })
  await tick()
  assert.equal(handed.settled, false)

  const fresh = workers[1]
  fresh.reply()
  await tick()
  fresh.reply()
  await tick()
  assert.deepEqual(fresh.waiting().map((m) => m.id), [late.id])
  old.onmessage?.({ data: { id: late.id, ok: true, result: 'stale', mem: SMALL } })
  await tick()
  assert.equal(handed.settled, false)
  fresh.reply({ result: 'fresh' })
  await tick()
  assert.equal(handed.value, 'fresh')
})

test('a trap while the fresh worker waits hands it the trapped worker\'s calls first, then the queued ones', async () => {
  const { workers, send, old, fresh, calls: [x, y] } = await handover(['x', 'y'])
  const z = track(send('best', stage('z')))
  const w = track(send('best', stage('w')))
  assert.deepEqual(fresh.waiting(), [])
  old.reply(TRAP)
  await tick()
  assert.ok(crashed(/unreachable/)(x.error))
  assert.equal(old.terminated, true)
  assert.deepEqual(labels(fresh), ['y', 'z', 'w'])
  for (const label of ['y', 'z', 'w']) fresh.reply({ result: label })
  await tick()
  assert.deepEqual([y.value, z.value, w.value], ['y', 'z', 'w'])
  assert.equal(workers.length, 2)
})

test('if the old worker crashes while the fresh one waits, its own calls fail and the queued ones go to the fresh worker', async () => {
  const { workers, send, old, fresh, calls: [x] } = await handover(['x'])
  const y = track(send('best', stage('y')))
  const z = track(send('best', stage('z')))
  old.onerror?.({ message: 'Uncaught RuntimeError: memory access out of bounds' })
  await tick()
  assert.ok(crashed(/out of bounds/)(x.error))
  assert.equal(old.terminated, true)
  assert.deepEqual(labels(fresh), ['y', 'z'])
  fresh.reply({ result: 'y' })
  fresh.reply({ result: 'z' })
  await tick()
  assert.deepEqual([y.value, z.value], ['y', 'z'])
  assert.equal(workers.length, 2)
})

test('if the fresh worker crashes while it waits, the queued calls go back to the old worker in order', async () => {
  const { workers, send, old, fresh, calls: [x] } = await handover(['x'])
  const y = track(send('best', stage('y')))
  const z = track(send('best', stage('z')))
  assert.deepEqual(labels(old), ['x'])
  fresh.onerror?.({ message: 'Uncaught RangeError: WebAssembly.Memory(): could not allocate memory' })
  await tick()
  assert.equal(fresh.terminated, true)
  assert.deepEqual(labels(old), ['x', 'y', 'z'])

  const after = track(send('best', stage('after')))
  assert.deepEqual(labels(old), ['x', 'y', 'z', 'after'])
  for (const label of ['x', 'y', 'z', 'after']) old.reply({ result: label })
  await tick()
  assert.deepEqual([x.value, y.value, z.value, after.value], ['x', 'y', 'z', 'after'])
  assert.equal(old.terminated, false)
  assert.equal(workers.length, 2)
})

test('if the catch-up replay fails, the queued calls go back to the old worker in order', async () => {
  const { workers, send } = await owning([5, 6])
  const [old] = workers
  const big = send('best', stage('big'))
  old.reply({ result: 'big', mem: MEMORY_LIMIT })
  await big
  const fresh = workers[1]
  fresh.reply()
  await tick()
  const change = send('setWardrobe', { ids: [7] })
  fresh.reply()
  await tick()
  const y = track(send('best', stage('y')))
  const z = track(send('best', stage('z')))
  old.reply({ result: { items: 1, unresolved: 0, known: 1 } })
  await change
  await tick()
  assert.deepEqual(shape(fresh), [['setWardrobe', { ids: [7] }]])
  assert.deepEqual(old.waiting(), [])

  fresh.reply(TRAP)
  await tick()
  assert.equal(fresh.terminated, true)
  assert.deepEqual(labels(old), ['y', 'z'])
  old.reply({ result: 'y' })
  old.reply({ result: 'z' })
  await tick()
  assert.deepEqual([y.value, z.value], ['y', 'z'])
  assert.equal(old.terminated, false)
})

test('an engine that exited is replaced the same way', async () => {
  const { workers, send } = await started()
  const gone = send('best', STAGE)
  workers[0].reply({ ok: false, error: 'Error: Go program has already exited', fault: true })
  await assert.rejects(gone, crashed(/already exited/))
  const next = send('best', STAGE)
  workers[1].reply()
  await tick()
  workers[1].reply({ result: 'works' })
  assert.equal(await next, 'works')
})

test('if the fresh worker cannot start, the old one keeps working and a later reply at the limit tries again', async () => {
  const { workers, send } = await owning([5])
  const [old] = workers
  const big = send('best', STAGE)
  old.reply({ result: 'big', mem: MEMORY_LIMIT })
  await big
  workers[1].reply(OFFLINE)
  await tick()
  assert.equal(workers[1].terminated, true)

  const next = send('best', STAGE)
  old.reply({ result: 'old', mem: MEMORY_LIMIT })
  assert.equal(await next, 'old')
  assert.equal(workers.length, 2)
  const later = send('best', STAGE)
  old.reply({ result: 'later', mem: MEMORY_LIMIT })
  assert.equal(await later, 'later')
  assert.equal(workers.length, 3)
  assert.deepEqual(shape(workers[2]), [['init', { version: V }]])
})

test('while no fresh worker can start, a start is tried again after 1, 2, 4 and more replies at the limit, up to MAX_BACKOFF', async () => {
  assert.equal(MAX_BACKOFF, 64)
  const { workers, send } = await owning([5])
  const [old] = workers
  const tries: number[] = []
  for (let n = 1; n <= 200; n++) {
    const before = workers.length
    const call = send('best', STAGE)
    old.reply({ result: n, mem: MEMORY_LIMIT })
    assert.equal(await call, n)
    if (workers.length === before) continue
    tries.push(n)
    workers[workers.length - 1].reply(OFFLINE)
    await tick()
  }
  assert.deepEqual(tries, [1, 3, 6, 11, 20, 37, 70, 135, 200])
  assert.ok(workers.slice(1).every((w) => w.terminated))
  assert.equal(old.terminated, false)
})

test('a fresh worker that does start clears the wait for the one after it', async () => {
  const { workers, send } = await owning([5])
  const [old] = workers
  await full(send, old)
  workers[1].reply(OFFLINE)
  await tick()
  await full(send, old)
  await full(send, old)
  assert.equal(workers.length, 3)
  const fresh = workers[2]
  fresh.reply()
  await tick()
  fresh.reply()
  await tick()
  assert.equal(old.terminated, true)

  await full(send, fresh)
  assert.equal(workers.length, 4)
  workers[3].reply(OFFLINE)
  await tick()
  await full(send, fresh)
  assert.equal(workers.length, 4)
  await full(send, fresh)
  assert.equal(workers.length, 5)
})

test('a trap during the wait starts a fresh worker at once, and that worker tries its own start at once', async () => {
  const { workers, send } = await owning([5])
  const [old] = workers
  await full(send, old)
  workers[1].reply(OFFLINE)
  await tick()

  const bad = send('best', stage('bad'))
  const next = track(send('best', stage('next')))
  old.reply({ ...TRAP, mem: MEMORY_LIMIT })
  await assert.rejects(bad, crashed(/unreachable/))
  assert.equal(workers.length, 3)
  const fresh = workers[2]
  fresh.reply()
  await tick()
  fresh.reply()
  await tick()
  assert.deepEqual(labels(fresh), ['next'])
  fresh.reply({ result: 'next', mem: MEMORY_LIMIT })
  await tick()
  assert.equal(next.value, 'next')
  assert.equal(workers.length, 4)
})

test('if the worker is dead and no fresh one can start, its calls and the queued ones fail with the reason, and every later call tries again', async () => {
  const { workers, send } = await started()
  const bad = track(send('best', stage('bad')))
  const handed = track(send('best', stage('handed')))
  workers[0].reply(TRAP)
  const queued = track(send('best', stage('queued')))
  workers[1].reply(OFFLINE)
  await tick()
  assert.ok(crashed(/unreachable/)(bad.error))
  assert.match(handed.error?.message ?? '', /Failed to fetch/)
  assert.match(queued.error?.message ?? '', /Failed to fetch/)
  assert.equal(workers.length, 2)

  const first = track(send('best', STAGE))
  assert.equal(workers.length, 3)
  workers[2].reply(OFFLINE)
  await tick()
  assert.match(first.error?.message ?? '', /Failed to fetch/)

  const retry = send('best', STAGE)
  assert.equal(workers.length, 4)
  workers[3].reply()
  await tick()
  workers[3].reply({ result: 'works' })
  assert.equal(await retry, 'works')
})

test('a late error event from a worker already let go starts nothing', async () => {
  const { workers, send } = await started()
  const bad = send('best', STAGE)
  workers[0].reply(TRAP)
  await assert.rejects(bad, crashed(/unreachable/))
  workers[1].reply(OFFLINE)
  await tick()
  workers[0].onerror?.({ message: 'Uncaught RuntimeError: unreachable' })
  workers[1].onerror?.({ message: 'Uncaught TypeError: Failed to fetch' })
  await tick()
  assert.equal(workers.length, 2)

  const next = send('best', STAGE)
  assert.equal(workers.length, 3)
  workers[2].reply()
  await tick()
  workers[2].reply({ result: 'works' })
  assert.equal(await next, 'works')
})

test('a worker that crashes outright fails its calls and is replaced', async () => {
  const { workers, send } = await owning([5])
  const lost = send('best', STAGE)
  workers[0].onerror?.({ message: 'Uncaught RuntimeError: unreachable' })
  await assert.rejects(lost, crashed(/Uncaught RuntimeError: unreachable/))
  assert.equal(workers[0].terminated, true)
  const next = send('best', STAGE)
  workers[1].reply()
  await tick()
  assert.deepEqual(shape(workers[1]), [['setWardrobe', { ids: [5] }]])
  workers[1].reply()
  await tick()
  workers[1].reply({ result: 'works' })
  assert.equal(await next, 'works')
})
