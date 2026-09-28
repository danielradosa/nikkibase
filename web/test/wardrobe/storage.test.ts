import { test } from 'node:test'
import assert from 'node:assert/strict'
import { classify, clearWardrobe, loadWardrobe, saveWardrobe } from '../../src/wardrobe/storage.ts'

const V = '2026-09-22'
const record = (source: string, version = V) => ({ version, ids: [10001, 20001], source, savedAt: 0 })

test('current sources load as they are', () => {
  for (const source of ['sel', 'manual', 'clothes_date']) {
    const r = classify(record(source), V)
    assert.equal(r.status, 'ok')
    assert.equal(r.status === 'ok' && r.entry.source, source)
  }
})

test('an unknown source is reported, not guessed at', () => {
  assert.equal(classify(record('something-else'), V).status, 'unrecognised')
})

test('another catalogue version is stale whatever its source', () => {
  assert.equal(classify(record('clothes_date', '2026-07-29'), V).status, 'stale')
  assert.equal(classify(record('sel', '2026-07-29'), V).status, 'stale')
})

test('empty or malformed records are nothing', () => {
  for (const raw of [null, undefined, 'x', {}, { version: V, ids: [], source: 'sel' }]) {
    assert.equal(classify(raw, V).status, 'none')
  }
})

type Opening = 'success' | 'error' | 'blocked' | 'throw'
type Ending = 'complete' | 'error' | 'abort' | 'throw'
type Handlers = { result?: unknown; onsuccess?: () => void; onerror?: () => void; onblocked?: () => void; onupgradeneeded?: () => void }
type TxHandlers = { oncomplete?: () => void; onerror?: () => void; onabort?: () => void }

function fakeIndexedDB({ open = 'success', tx = 'complete', stored = null }: { open?: Opening; tx?: Ending; stored?: unknown } = {}) {
  const log: string[] = []
  const later = (run: () => void) => setTimeout(run, 0)
  const db = {
    objectStoreNames: { contains: () => true },
    createObjectStore: () => {},
    close: () => log.push('close'),
    transaction: () => {
      if (tx === 'throw') throw new Error('InvalidStateError')
      const t: TxHandlers & { objectStore: () => unknown } = {
        objectStore: () => ({
          put: (_value: unknown, key: string) => {
            log.push(`put ${key}`)
            end()
          },
          delete: (key: string) => {
            log.push(`delete ${key}`)
            end()
          },
          get: (key: string) => {
            log.push(`get ${key}`)
            const req: Handlers = {}
            later(() => {
              if (tx === 'complete') {
                req.result = stored
                req.onsuccess?.()
              } else if (tx === 'error') req.onerror?.()
              else t.onabort?.()
            })
            return req
          },
        }),
      }
      const end = () => later(() => (tx === 'complete' ? t.oncomplete?.() : tx === 'error' ? t.onerror?.() : t.onabort?.()))
      return t
    },
  }
  return {
    log,
    open: () => {
      if (open === 'throw') throw new Error('SecurityError')
      const req: Handlers = {}
      later(() => {
        if (open === 'success') {
          req.result = db
          req.onupgradeneeded?.()
          req.onsuccess?.()
        } else if (open === 'error') req.onerror?.()
        else req.onblocked?.()
      })
      return req
    },
  }
}

async function withStorage<T>(fake: ReturnType<typeof fakeIndexedDB> | undefined, run: () => Promise<T>): Promise<T> {
  const g = globalThis as { indexedDB?: unknown }
  const before = g.indexedDB
  if (fake) g.indexedDB = fake
  else delete g.indexedDB
  try {
    return await run()
  } finally {
    if (before === undefined) delete g.indexedDB
    else g.indexedDB = before
  }
}

const entry = { version: V, ids: [10001, 20001], source: 'sel' as const, savedAt: 0 }

test('a save or a clear says it worked once the browser has stored it', async () => {
  const fake = fakeIndexedDB()
  assert.equal(await withStorage(fake, () => saveWardrobe(entry)), true)
  assert.equal(await withStorage(fake, () => clearWardrobe()), true)
  assert.deepEqual(fake.log, ['put current', 'close', 'delete current', 'close'])
})

test('a save or a clear the browser refuses says it did not work', async () => {
  for (const tx of ['error', 'abort', 'throw'] as const) {
    const fake = fakeIndexedDB({ tx })
    assert.equal(await withStorage(fake, () => saveWardrobe(entry)), false, `save when the transaction ends in ${tx}`)
    assert.equal(await withStorage(fake, () => clearWardrobe()), false, `clear when the transaction ends in ${tx}`)
    assert.equal(fake.log.filter((line) => line === 'close').length, 2, `the database is closed after ${tx}`)
  }
})

test('a save or a clear that cannot open the storage says it did not work', async () => {
  for (const open of ['error', 'blocked', 'throw'] as const) {
    const fake = fakeIndexedDB({ open })
    assert.equal(await withStorage(fake, () => saveWardrobe(entry)), false, `save when opening ends in ${open}`)
    assert.equal(await withStorage(fake, () => clearWardrobe()), false, `clear when opening ends in ${open}`)
  }
  assert.equal(await withStorage(undefined, () => saveWardrobe(entry)), false, 'save with no storage at all')
  assert.equal(await withStorage(undefined, () => clearWardrobe()), false, 'clear with no storage at all')
})

test('loading tells nothing saved apart from a storage it could not read', async () => {
  const ok = await withStorage(fakeIndexedDB({ stored: entry }), () => loadWardrobe(V))
  assert.equal(ok.status, 'ok')
  assert.equal((await withStorage(fakeIndexedDB({ stored: undefined }), () => loadWardrobe(V))).status, 'none')
  for (const tx of ['error', 'abort', 'throw'] as const) {
    assert.equal((await withStorage(fakeIndexedDB({ tx }), () => loadWardrobe(V))).status, 'unreadable', `read ending in ${tx}`)
  }
  for (const open of ['error', 'blocked', 'throw'] as const) {
    assert.equal((await withStorage(fakeIndexedDB({ open }), () => loadWardrobe(V))).status, 'unreadable', `opening ending in ${open}`)
  }
  assert.equal((await withStorage(undefined, () => loadWardrobe(V))).status, 'unreadable', 'no storage at all')
})
