export const MEMORY_LIMIT = 536870912
export const MAX_BACKOFF = 64

export type Request = { id: number; type: string; payload?: unknown }
export type Reply = { id: number; ok: boolean; result?: unknown; error?: string; fault?: boolean; mem?: number }
export type Verdict = 'keep' | 'dead' | 'full'
export type Port = Pick<Worker, 'postMessage' | 'terminate' | 'onmessage' | 'onerror'>

export function recycle(reply: Reply, limit = MEMORY_LIMIT): Verdict {
  if (reply.fault) return 'dead'
  return (reply.mem ?? 0) >= limit ? 'full' : 'keep'
}

export function wardrobeAfter(request: Request, result: unknown): number[] | undefined {
  if (request.type === 'setWardrobe') return (request.payload as { ids: number[] }).ids
  if (request.type === 'decode' || request.type === 'selections') return (result as { ids: number[] }).ids
  return undefined
}

const AGAIN: Record<string, string> = {
  best: 'pick the stage again',
  decode: 'import your wardrobe again',
  selections: 'import your wardrobe again',
}

export function crashError(type: string, detail: unknown): Error {
  return new Error(`The engine crashed and was restarted — ${AGAIN[type] ?? 'try again'}.`, { cause: detail })
}

export function wasmMemory(exports: WebAssembly.Exports): WebAssembly.Memory | undefined {
  const memory = exports.memory ?? exports.mem
  return memory instanceof WebAssembly.Memory ? memory : undefined
}

type Call = { request: Request; resolve: (value: any) => void; reject: (error: Error) => void }
type Slot = { port: Port; calls: Map<number, Call>; dead: boolean; ready: boolean; replayed: number[] | null }

export function relay(spawn: () => Port, limit = MEMORY_LIMIT) {
  let nextId = 0
  let version: string | null = null
  let wardrobe: number[] | null = null
  let held: Call[] = []
  let holding = false
  let spare: Slot | null = null
  let skip = 0
  let backoff = 1
  let current = open()

  function open(): Slot {
    const slot: Slot = { port: spawn(), calls: new Map(), dead: false, ready: false, replayed: null }
    slot.port.onmessage = (e) => receive(slot, e.data)
    slot.port.onerror = (e) => crash(slot, new Error(e.message || 'the engine stopped'))
    return slot
  }

  function post(slot: Slot, call: Call) {
    slot.calls.set(call.request.id, call)
    slot.port.postMessage(call.request)
  }

  function ask<T>(slot: Slot, type: string, payload?: unknown) {
    return new Promise<T>((resolve, reject) => post(slot, { request: { id: nextId++, type, payload }, resolve, reject }))
  }

  function close(slot: Slot) {
    if (slot.dead) return
    slot.dead = true
    slot.port.terminate()
  }

  function take() {
    const calls = held
    held = []
    return calls
  }

  function receive(slot: Slot, reply: Reply) {
    const call = reply ? slot.calls.get(reply.id) : undefined
    if (!call) return
    slot.calls.delete(reply.id)
    if (reply.ok) {
      const ids = wardrobeAfter(call.request, reply.result)
      if (ids && slot === current) wardrobe = ids
      call.resolve(reply.result)
    } else call.reject(reply.fault ? crashError(call.request.type, reply.error) : new Error(reply.error))
    if (slot !== current) return
    const verdict = recycle(reply, limit)
    if (verdict === 'dead') retire(slot)
    else if (verdict === 'full' && skip) skip--
    else if (verdict === 'full') prepare()
    settle()
  }

  function retire(slot: Slot) {
    close(slot)
    held = [...slot.calls.values(), ...held]
    slot.calls.clear()
    holding = true
    prepare()
  }

  function crash(slot: Slot, error: Error) {
    if (slot.dead) return
    close(slot)
    const calls = [...slot.calls.values()]
    slot.calls.clear()
    for (const call of calls) call.reject(crashError(call.request.type, error.message))
    if (slot === spare) drop(slot, error)
    if (slot !== current) return
    holding = true
    prepare()
    settle()
  }

  function prepare() {
    if (spare) return
    if (version === null) {
      for (const call of take()) call.reject(new Error('the engine was never started'))
      return
    }
    const slot = open()
    spare = slot
    ask(slot, 'init', { version })
      .then(() => replay(slot))
      .catch((err) => drop(slot, err))
  }

  function replay(slot: Slot) {
    const ids = wardrobe
    return (ids ? ask(slot, 'setWardrobe', { ids }) : Promise.resolve()).then(() => {
      if (spare !== slot) return
      slot.replayed = ids
      slot.ready = true
      holding = true
      settle()
    })
  }

  function drop(slot: Slot, error: unknown) {
    if (spare !== slot) return
    spare = null
    close(slot)
    if (current.dead) {
      const reason = error instanceof Error ? error : new Error(String(error))
      for (const call of take()) call.reject(reason)
      return
    }
    skip = backoff
    backoff = Math.min(backoff * 2, MAX_BACKOFF)
    holding = false
    for (const call of take()) post(current, call)
  }

  function settle() {
    const slot = spare
    if (!slot || !slot.ready || current.calls.size) return
    if (slot.replayed !== wardrobe) {
      slot.ready = false
      replay(slot).catch((err) => drop(slot, err))
      return
    }
    const old = current
    spare = null
    current = slot
    holding = false
    skip = 0
    backoff = 1
    close(old)
    for (const call of take()) post(current, call)
  }

  return function send<T>(type: string, payload?: unknown): Promise<T> {
    if (type === 'init') version = (payload as { version: string }).version
    return new Promise<T>((resolve, reject) => {
      const call: Call = { request: { id: nextId++, type, payload }, resolve, reject }
      if (!holding) return post(current, call)
      held.push(call)
      prepare()
    })
  }
}
