/// <reference lib="webworker" />
import './wasm_exec.js'
import wasmUrl from './generated/nikkibase.wasm?url'
import { wasmMemory } from './recycle'

declare const Go: any
declare const nikkibase: {
  loadCatalogue(b: Uint8Array): Reply
  loadKeystream(b: Uint8Array): Reply
  decodeWardrobe(text: string): Reply
  importSelections(text: string): Reply
  setWardrobe(ids: number[]): Reply
  best(weights: number[], attrs: number[], skills: unknown, tags: unknown, scope: string, require: unknown): Reply
  worthStart(versions: unknown, settings: unknown, suits: unknown): Reply
  worthRun(session: unknown, count: unknown): Reply
  worthRank(session: unknown, filter: unknown, limit: unknown): Reply
  places(): Reply
}
type Reply = { ok: true; json: string } | { ok: false; error: string }

let ready: Promise<void> | null = null
let memory: WebAssembly.Memory | undefined

async function boot(version: string) {
  const go = new Go()
  const [mod, catalogue] = await Promise.all([
    WebAssembly.instantiateStreaming(fetch(wasmUrl), go.importObject),
    fetch(`/data/${version}/items.bin`).then((r) => r.arrayBuffer()),
  ])
  memory = wasmMemory(mod.instance.exports)
  go.run(mod.instance)
  call(() => nikkibase.loadCatalogue(new Uint8Array(catalogue)))
}

let keystream: Promise<void> | null = null

async function fetchKeystream() {
  const r = await fetch(__KEYSTREAM_URL__)
  if (!r.ok) throw new Error(`keystream.bin: ${r.status}`)
  const bytes = new Uint8Array(await r.arrayBuffer())
  call(() => nikkibase.loadKeystream(bytes))
}

function loadKeystream() {
  keystream ??= fetchKeystream().catch((err) => {
    keystream = null
    throw err
  })
  return keystream
}

const faults = new WeakSet<object>()

function fault(err: unknown) {
  const thrown = err instanceof Object ? err : new Error(String(err))
  faults.add(thrown)
  return thrown
}

function call(run: () => Reply | undefined) {
  let reply: Reply | undefined
  try {
    reply = run()
  } catch (err) {
    throw fault(err)
  }
  if (!reply) throw fault(new Error('the engine gave no reply'))
  return unwrap(reply)
}

function unwrap(reply: Reply) {
  if (!reply.ok) throw new Error(reply.error)
  return JSON.parse(reply.json)
}

const size = () => memory?.buffer.byteLength ?? 0

function dispatch(type: string, payload: any) {
  switch (type) {
    case 'decode':
      return call(() => nikkibase.decodeWardrobe(payload.text))
    case 'selections':
      return call(() => nikkibase.importSelections(payload.text))
    case 'setWardrobe':
      return call(() => nikkibase.setWardrobe(payload.ids))
    case 'worthStart':
      return call(() => nikkibase.worthStart(payload.versions, payload.settings ?? null, payload.suits ?? null))
    case 'worthRun':
      return call(() => nikkibase.worthRun(payload.session, payload.count))
    case 'worthRank':
      return call(() => nikkibase.worthRank(payload.session, payload.filter, payload.limit))
    case 'places':
      return call(() => nikkibase.places())
    default:
      return call(() =>
        nikkibase.best(
          payload.weights,
          payload.attrs,
          payload.skills ?? null,
          payload.tags ?? null,
          payload.scope ?? 'wardrobe',
          payload.require ?? null,
        ),
      )
  }
}

self.onmessage = async (e: MessageEvent) => {
  const { id, type, payload } = e.data
  try {
    if (type === 'init') {
      ready ??= boot(payload.version)
      await ready
      self.postMessage({ id, ok: true, result: null, mem: size() })
      return
    }
    await ready
    if (type === 'keystream') {
      await loadKeystream()
      self.postMessage({ id, ok: true, result: null, mem: size() })
      return
    }
    if (type === 'decode') await loadKeystream()
    const result = dispatch(type, payload)
    self.postMessage({ id, ok: true, result, mem: size() })
  } catch (err) {
    self.postMessage({ id, ok: false, error: String(err), fault: err instanceof Object && faults.has(err), mem: size() })
  }
}
