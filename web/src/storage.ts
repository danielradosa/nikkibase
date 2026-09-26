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

export async function saveWardrobe(entry: SavedWardrobe): Promise<void> {
  const db = await open()
  if (!db) return
  await new Promise<void>((resolve) => {
    try {
      const tx = db.transaction(STORE, 'readwrite')
      tx.objectStore(STORE).put(entry, KEY)
      tx.oncomplete = () => resolve()
      tx.onerror = () => resolve()
      tx.onabort = () => resolve()
    } catch {
      resolve()
    }
  })
  db.close()
}

export type LoadResult =
  | { status: 'none' }
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
  if (!db) return { status: 'none' }
  const entry = await new Promise<unknown>((resolve) => {
    try {
      const tx = db.transaction(STORE, 'readonly')
      const req = tx.objectStore(STORE).get(KEY)
      req.onsuccess = () => resolve(req.result ?? null)
      req.onerror = () => resolve(null)
    } catch {
      resolve(null)
    }
  })
  db.close()
  return classify(entry, version)
}

export async function clearWardrobe(): Promise<void> {
  const db = await open()
  if (!db) return
  await new Promise<void>((resolve) => {
    try {
      const tx = db.transaction(STORE, 'readwrite')
      tx.objectStore(STORE).delete(KEY)
      tx.oncomplete = () => resolve()
      tx.onerror = () => resolve()
    } catch {
      resolve()
    }
  })
  db.close()
}
