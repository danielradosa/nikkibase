import { test } from 'node:test'
import assert from 'node:assert/strict'
import { bestOutfitRun, type BestPatch, type BestQuery, type Shown } from '../../src/outfit/bestOutfitRun.ts'
import type { Outfit } from '../../src/engine/engine.ts'
import type { Ideal, IdealTable } from '../../src/outfit/stages.ts'

const flush = () => new Promise((resolve) => setTimeout(resolve, 0))

const outfit = (score: number): Outfit => ({ score, items: [], dress: 0, separates: 0, ownedPlaces: [] })

function held<T>() {
  let resolve!: (value: T) => void
  let reject!: (e: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

function fakeStore() {
  const patches: BestPatch[] = []
  let state: { outfit: Outfit | null } = { outfit: null }
  return {
    patches,
    get: () => state,
    set: (patch: BestPatch) => {
      patches.push(patch)
      if ('outfit' in patch) state = { outfit: patch.outfit ?? null }
    },
    clear: () => {
      state = { outfit: null }
    },
  }
}

function fakeEngine() {
  const calls: string[] = []
  return {
    calls,
    count: (what: string) => calls.filter((c) => c === what).length,
    answer: (what: string, result: () => Promise<Outfit>) => () => {
      calls.push(what)
      return result()
    },
  }
}

const owned = [1, 2, 3]

function query(engine: ReturnType<typeof fakeEngine>, over: Partial<BestQuery> = {}): BestQuery {
  return {
    owned,
    sig: 'Story/1-1|Princess|null|null|none',
    liveKey: 'Story/1-1|null',
    yours: engine.answer('yours', async () => outfit(100)),
    possible: engine.answer('possible', async () => outfit(900)),
    lookup: () => null,
    ...over,
  }
}

const tableIdeal: Ideal = { score: 800, items: [{ id: 7, pos: 0 }] }

test('a search says it is busy, then shows the outfit and the best possible one', async () => {
  const store = fakeStore()
  const engine = fakeEngine()
  const search = bestOutfitRun({ ideals: Promise.resolve(null), get: store.get, set: store.set })
  search(query(engine), { current: null })
  assert.deepEqual(store.patches, [{ busy: true, error: null }])
  await flush()
  assert.deepEqual(store.patches, [
    { busy: true, error: null },
    { outfit: outfit(100), ideal: outfit(900), busy: false },
  ])
})

test('the same wardrobe and settings with the shown outfit still there ask for nothing new', async () => {
  const store = fakeStore()
  const engine = fakeEngine()
  const search = bestOutfitRun({ ideals: Promise.resolve(null), get: store.get, set: store.set })
  const shown: { current: Shown | null } = { current: null }
  search(query(engine), shown)
  await flush()
  assert.equal(search(query(engine), shown), undefined)
  assert.equal(engine.count('yours'), 1)
  assert.equal(store.patches.length, 2)
  search(query(engine, { sig: 'Story/1-1|Maiden|null|null|none' }), shown)
  await flush()
  assert.equal(engine.count('yours'), 2, 'other settings search again')
  store.clear()
  search(query(engine, { sig: 'Story/1-1|Maiden|null|null|none' }), shown)
  await flush()
  assert.equal(engine.count('yours'), 3, 'an outfit cleared from the page is searched again')
  search(query(engine, { owned: [...owned], sig: 'Story/1-1|Maiden|null|null|none' }), shown)
  await flush()
  assert.equal(engine.count('yours'), 4, 'a new wardrobe is searched again')
})

test('the best possible outfit comes from the table when it has the stage, and from one live run when not', async () => {
  const table: IdealTable = {}
  const store = fakeStore()
  const engine = fakeEngine()
  const search = bestOutfitRun({ ideals: Promise.resolve(table), get: store.get, set: store.set })
  const seen: (IdealTable | null)[] = []
  search(query(engine, { lookup: (t) => (seen.push(t), tableIdeal) }), { current: null })
  await flush()
  assert.equal(seen[0], table)
  assert.equal(engine.count('possible'), 0)
  assert.deepEqual(store.patches.at(-1), { outfit: outfit(100), ideal: tableIdeal, busy: false })
  search(query(engine, { sig: 'other', lookup: () => null }), { current: null })
  await flush()
  assert.equal(engine.count('possible'), 1)
  assert.deepEqual(store.patches.at(-1), { outfit: outfit(100), ideal: outfit(900), busy: false })
})

test('searches with the same live key share one live run, and a failed live run is tried again next time', async () => {
  const store = fakeStore()
  const engine = fakeEngine()
  const search = bestOutfitRun({ ideals: Promise.resolve(null), get: store.get, set: store.set })
  const slow = held<Outfit>()
  const shared = engine.answer('possible', () => slow.promise)
  search(query(engine, { sig: 'a', possible: shared }), { current: null })
  search(query(engine, { sig: 'b', possible: shared }), { current: null })
  await flush()
  slow.resolve(outfit(900))
  await flush()
  search(query(engine, { sig: 'c', possible: shared }), { current: null })
  await flush()
  assert.equal(engine.count('possible'), 1)
  assert.equal(engine.count('yours'), 3)

  let fail = true
  const flaky = engine.answer('flaky', async () => {
    if (fail) throw new Error('worker died')
    return outfit(950)
  })
  search(query(engine, { sig: 'd', liveKey: 'Story/2-2|null', possible: flaky }), { current: null })
  await flush()
  assert.deepEqual(store.patches.at(-1), { error: 'Error: worker died', busy: false, outfit: null, ideal: null })
  fail = false
  search(query(engine, { sig: 'e', liveKey: 'Story/2-2|null', possible: flaky }), { current: null })
  await flush()
  assert.equal(engine.count('flaky'), 2)
  assert.deepEqual(store.patches.at(-1), { outfit: outfit(100), ideal: outfit(950), busy: false })
})

test('a failed search says why and clears the outfit and the best possible one', async () => {
  const store = fakeStore()
  const engine = fakeEngine()
  const search = bestOutfitRun({ ideals: Promise.resolve(null), get: store.get, set: store.set })
  search(query(engine, { yours: engine.answer('yours', () => Promise.reject(new Error('no wardrobe'))) }), { current: null })
  await flush()
  assert.deepEqual(store.patches, [
    { busy: true, error: null },
    { error: 'Error: no wardrobe', busy: false, outfit: null, ideal: null },
  ])
})

test('a search stopped before it ends is no longer busy, and its late answer is dropped', async () => {
  const store = fakeStore()
  const engine = fakeEngine()
  const search = bestOutfitRun({ ideals: Promise.resolve(null), get: store.get, set: store.set })
  const late = held<Outfit>()
  const shown: { current: Shown | null } = { current: null }
  const stop = search(query(engine, { yours: engine.answer('yours', () => late.promise) }), shown)
  stop?.()
  assert.deepEqual(store.patches, [{ busy: true, error: null }, { busy: false }])
  late.resolve(outfit(100))
  await flush()
  assert.deepEqual(store.patches, [{ busy: true, error: null }, { busy: false }])
  assert.equal(shown.current, null)
  assert.equal(store.get().outfit, null)

  const failing = held<Outfit>()
  const stopFailing = search(query(engine, { sig: 'b', yours: engine.answer('yours', () => failing.promise) }), shown)
  stopFailing?.()
  failing.reject(new Error('too late'))
  await flush()
  assert.equal(store.patches.some((p) => 'error' in p && p.error !== null), false)
})

test('stopping a search that has already ended leaves the page alone', async () => {
  const store = fakeStore()
  const engine = fakeEngine()
  const search = bestOutfitRun({ ideals: Promise.resolve(null), get: store.get, set: store.set })
  const stop = search(query(engine), { current: null })
  await flush()
  const before = store.patches.length
  stop?.()
  assert.equal(store.patches.length, before)
})
