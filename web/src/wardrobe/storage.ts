const DB_NAME = 'nikkibase'
const STORE = 'wardrobe'
const KEY = 'current'

export type WardrobeSource = 'sel' | 'manual' | 'clothes_date'

export type SavedWardrobe = {
  version: string
  ids: number[]
  source: WardrobeSource
  savedAt: number
}

type StoredWardrobe = Omit<SavedWardrobe, 'source'> & { source: string }

function open(): Promise<IDBDatabase | null> {
  return new Promise((resolve) => {
    let request: IDBOpenDBRequest
    try {
      request = indexedDB.open(DB_NAME, 1)
    } catch {
      resolve(null)
      return
    }
    request.onupgradeneeded = () => {
      if (!request.result.objectStoreNames.contains(STORE)) {
        request.result.createObjectStore(STORE)
      }
    }
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => resolve(null)
    request.onblocked = () => resolve(null)
  })
}

async function write(op: (store: IDBObjectStore) => void): Promise<boolean> {
  const db = await open()
  if (!db) return false
  const written = await new Promise<boolean>((resolve) => {
    try {
      const tx = db.transaction(STORE, 'readwrite')
      op(tx.objectStore(STORE))
      tx.oncomplete = () => resolve(true)
      tx.onerror = () => resolve(false)
      tx.onabort = () => resolve(false)
    } catch {
      resolve(false)
    }
  })
  db.close()
  return written
}

export const saveWardrobe = (entry: SavedWardrobe) => write((store) => store.put(entry, KEY))

export type LoadResult =
  | { status: 'none' }
  | { status: 'unreadable' }
  | { status: 'ok'; entry: SavedWardrobe }
  | { status: 'stale'; entry: StoredWardrobe }
  | { status: 'unrecognised'; entry: StoredWardrobe }

export function classify(raw: unknown, version: string): LoadResult {
  const entry = raw as StoredWardrobe | null
  if (!entry || typeof entry !== 'object' || !Array.isArray(entry.ids) || entry.ids.length === 0) {
    return { status: 'none' }
  }
  if (entry.version !== version) return { status: 'stale', entry }
  if (entry.source === 'sel' || entry.source === 'manual' || entry.source === 'clothes_date') {
    return { status: 'ok', entry: entry as SavedWardrobe }
  }
  return { status: 'unrecognised', entry }
}

export async function loadWardrobe(version: string): Promise<LoadResult> {
  const db = await open()
  if (!db) return { status: 'unreadable' }
  const read = await new Promise<{ ok: true; entry: unknown } | { ok: false }>((resolve) => {
    try {
      const tx = db.transaction(STORE, 'readonly')
      const req = tx.objectStore(STORE).get(KEY)
      req.onsuccess = () => resolve({ ok: true, entry: req.result ?? null })
      req.onerror = () => resolve({ ok: false })
      tx.onabort = () => resolve({ ok: false })
    } catch {
      resolve({ ok: false })
    }
  })
  db.close()
  return read.ok ? classify(read.entry, version) : { status: 'unreadable' }
}

export const clearWardrobe = () => write((store) => store.delete(KEY))
