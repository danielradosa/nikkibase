import type { Outfit } from '../engine/engine'
import type { Ideal, IdealTable } from './stages'

export type Shown = { owned: readonly number[]; sig: string; outfit: Outfit }

export type BestPatch = { busy?: boolean; error?: string | null; outfit?: Outfit | null; ideal?: Ideal | null }

export type BestDeps = {
  ideals: Promise<IdealTable | null>
  get: () => { outfit: Outfit | null }
  set: (patch: BestPatch) => void
}

export type BestQuery = {
  owned: readonly number[]
  sig: string
  liveKey: string
  yours: () => Promise<Outfit>
  possible: () => Promise<Outfit>
  lookup: (table: IdealTable | null) => Ideal | null
}

export function liveRuns() {
  const pending = new Map<string, Promise<Outfit>>()
  return (key: string, run: () => Promise<Outfit>): Promise<Outfit> => {
    let found = pending.get(key)
    if (!found) {
      found = run()
      found.catch(() => pending.delete(key))
      pending.set(key, found)
    }
    return found
  }
}

export function bestOutfitRun(deps: BestDeps) {
  const live = liveRuns()
  return (query: BestQuery, shown: { current: Shown | null }): (() => void) | undefined => {
    const last = shown.current
    if (last && last.owned === query.owned && last.sig === query.sig && deps.get().outfit === last.outfit) return undefined
    let current = true
    let settled = false
    deps.set({ busy: true, error: null })
    Promise.all([query.yours(), deps.ideals.then((table) => query.lookup(table) ?? live(query.liveKey, query.possible))])
      .then(([outfit, ideal]) => {
        if (!current) return
        shown.current = { owned: query.owned, sig: query.sig, outfit }
        deps.set({ outfit, ideal, busy: false })
      })
      .catch((e) => current && deps.set({ error: String(e), busy: false, outfit: null, ideal: null }))
      .finally(() => {
        settled = true
      })
    return () => {
      current = false
      if (!settled) deps.set({ busy: false })
    }
  }
}
