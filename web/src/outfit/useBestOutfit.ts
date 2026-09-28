import { useEffect, useRef } from 'react'
import { ideals } from '../boot'
import { engine } from '../engine/engine'
import { searchPlan } from './skills'
import { useStore } from '../store'
import { lookupIdeal, placementKey, resolveStage, stageKey, type Stage } from './stages'
import { bestOutfitRun, type Shown } from './bestOutfitRun'

const search = bestOutfitRun({ ideals, get: () => useStore.getState(), set: (patch) => useStore.getState().set(patch) })

export function useBestOutfit(stages: readonly Stage[]) {
  const { owned, stage, difficulty, skills, placements, tab } = useStore()
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
    const chosen = resolveStage(found, difficulty)
    const require = chosen.rules?.require
    return search(
      {
        owned,
        sig: `${key}|${difficulty}|${mineKey}|${ceilingKey}|${fromTable}`,
        liveKey: `${key}|${ceilingKey}`,
        yours: () => engine.best(chosen.weights, chosen.attrs, chosen.tags, 'wardrobe', mine, require),
        possible: () => engine.best(chosen.weights, chosen.attrs, chosen.tags, 'all', ceiling, require),
        lookup: (table) => (fromTable ? lookupIdeal(table, found, difficulty, fromTable) : null),
      },
      shown,
    )
  }, [onTab, owned, found, key, difficulty, mineKey, ceilingKey, fromTable])
}
