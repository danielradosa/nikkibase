import { useEffect, useRef } from 'react'
import { ideals } from './boot'
import { engine, type Outfit } from './engine'
import { searchPlan } from './skills'
import { useStore } from './store'
import { lookupIdeal, placementKey, resolveStage, stageKey, type Stage } from './stages'

const liveIdeals = new Map<string, Promise<Outfit>>()

function liveIdeal(key: string, run: () => Promise<Outfit>): Promise<Outfit> {
  let pending = liveIdeals.get(key)
  if (!pending) {
    pending = run()
    pending.catch(() => liveIdeals.delete(key))
    liveIdeals.set(key, pending)
  }
  return pending
}

type Shown = { owned: number[]; sig: string; outfit: Outfit }

export function useBestOutfit(stages: readonly Stage[]) {
  const { owned, stage, difficulty, skills, placements, tab, set } = useStore()
  const found = stages.find((s) => stageKey(s) === stage)
  const key = found ? placementKey(found, difficulty) : null
  const placement = key && skills.manual ? placements[key] ?? null : null
  const { mine, ceiling, fromTable } = searchPlan(skills, placement)
  const mineKey = JSON.stringify(mine ?? null)
  const ceilingKey = JSON.stringify(ceiling ?? null)
  const onTab = tab === 'outfit'
  const shown = useRef<Shown | null>(null)

  useEffect(() => {
    if (!onTab || !owned.length || !found || !key) return
    const sig = `${key}|${difficulty}|${mineKey}|${ceilingKey}|${fromTable}`
    const last = shown.current
    if (last && last.owned === owned && last.sig === sig && useStore.getState().outfit === last.outfit) return
    const chosen = resolveStage(found, difficulty)
    const require = chosen.rules?.require
    const liveKey = `${key}|${ceilingKey}`
    let live = true
    let settled = false
    set({ busy: true, error: null })
    Promise.all([
      engine.best(chosen.weights, chosen.attrs, chosen.tags, 'wardrobe', mine, require),
      ideals.then(
        (table) =>
          (fromTable && lookupIdeal(table, found, difficulty, fromTable)) ??
          liveIdeal(liveKey, () => engine.best(chosen.weights, chosen.attrs, chosen.tags, 'all', ceiling, require)),
      ),
    ])
      .then(([outfit, best]) => {
        if (!live) return
        shown.current = { owned, sig, outfit }
        set({ outfit, ideal: best, busy: false })
      })
      .catch((e) => live && set({ error: String(e), busy: false, outfit: null, ideal: null }))
      .finally(() => {
        settled = true
      })
    return () => {
      live = false
      if (!settled) set({ busy: false })
    }
  }, [onTab, owned, found, key, difficulty, mineKey, ceilingKey, fromTable, set])
}
