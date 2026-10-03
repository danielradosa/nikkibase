import { useEffect, useRef } from 'react'
import { ideals } from '../boot'
import { engine, type Outfit, type ThemeSplit } from '../engine/engine'
import { searchPlan, type SkillRequest } from './skills'
import { useStore } from '../store'
import { lookupIdeal, placementKey, resolveStage, stageKey, themeGroup, type Stage } from './stages'
import { bestOutfitRun, type Shown } from './bestOutfitRun'

const search = bestOutfitRun({ ideals, get: () => useStore.getState(), set: (patch) => useStore.getState().set(patch) })

const splits = { owned: null as readonly number[] | null, runs: new Map<string, Promise<ThemeSplit>>() }

function splitOnce(owned: readonly number[], key: string, run: () => Promise<ThemeSplit>): Promise<ThemeSplit> {
  if (splits.owned !== owned) {
    splits.owned = owned
    splits.runs = new Map()
  }
  let found = splits.runs.get(key)
  if (!found) {
    found = run()
    found.catch(() => splits.runs.delete(key))
    splits.runs.set(key, found)
  }
  return found
}

export function useBestOutfit(stages: readonly Stage[]) {
  const { owned, stage, difficulty, skills, placements, tab, engine: engineState } = useStore()
  const found = stages.find((s) => stageKey(s) === stage)
  const key = found ? placementKey(found, difficulty) : null
  const placement = key && skills.manual ? placements[key] ?? null : null
  const { mine, ceiling, fromTable } = searchPlan(skills, placement)
  const group = found ? themeGroup(stages, found) : null
  const plans = group?.map((t) => {
    const k = placementKey(t, difficulty)
    return { stage: resolveStage(t, difficulty), ...searchPlan(skills, skills.manual ? placements[k] ?? null : null) }
  })
  const groupKey = plans ? JSON.stringify(plans.map((p) => [stageKey(p.stage), p.mine ?? null, p.ceiling ?? null])) : ''
  const mineKey = JSON.stringify(mine ?? null)
  const ceilingKey = JSON.stringify(ceiling ?? null)
  const onTab = tab === 'outfit'
  const shown = useRef<Shown | null>(null)

  useEffect(() => {
    if (!onTab || engineState === 'failed' || !owned.length || !found || !key) return
    const chosen = resolveStage(found, difficulty)
    const require = chosen.rules?.require
    const at = group ? group.indexOf(found) : -1
    const themed = (scope: 'wardrobe' | 'all', request: SkillRequest | undefined, which: 'mine' | 'ceiling') => async (): Promise<Outfit> => {
      if (!group || !plans) return engine.best(chosen.weights, chosen.attrs, chosen.tags, scope, request, require)
      const split = await splitOnce(owned, `${scope}|${groupKey}`, () =>
        engine.split(
          plans.map((p) => ({ weights: p.stage.weights, attrs: p.stage.attrs, tags: p.stage.tags, require: p.stage.rules?.require, skills: p[which] })),
          scope,
        ),
      )
      const outfit = await engine.best(chosen.weights, chosen.attrs, chosen.tags, scope, request, require, split.exclude[at])
      return { ...outfit, taken: split.taken[at] }
    }
    return search(
      {
        owned,
        sig: `${key}|${difficulty}|${mineKey}|${ceilingKey}|${fromTable}|${groupKey}`,
        liveKey: `${key}|${ceilingKey}|${groupKey}`,
        yours: themed('wardrobe', mine, 'mine'),
        possible: themed('all', ceiling, 'ceiling'),
        lookup: (table) => (fromTable ? lookupIdeal(table, found, difficulty, fromTable) : null),
      },
      shown,
    )
  }, [onTab, engineState, owned, found, key, difficulty, mineKey, ceilingKey, fromTable, groupKey])
}
