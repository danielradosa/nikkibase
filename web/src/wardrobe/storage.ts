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

export async function saveWardrobe(entry: SavedWardrobe): Promise<boolean> {
  const db = await open()
  if (!db) return false
  const saved = await new Promise<boolean>((resolve) => {
    try {
      const tx = db.transaction(STORE, 'readwrite')
      tx.objectStore(STORE).put(entry, KEY)
      tx.oncomplete = () => resolve(true)
      tx.onerror = () => resolve(false)
      tx.onabort = () => resolve(false)
    } catch {
      resolve(false)
    }
  })
  db.close()
  return saved
}

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

export async function clearWardrobe(): Promise<boolean> {
  const db = await open()
  if (!db) return false
  const cleared = await new Promise<boolean>((resolve) => {
    try {
      const tx = db.transaction(STORE, 'readwrite')
      tx.objectStore(STORE).delete(KEY)
      tx.oncomplete = () => resolve(true)
      tx.onerror = () => resolve(false)
      tx.onabort = () => resolve(false)
    } catch {
      resolve(false)
    }
  })
  db.close()
  return cleared
}
