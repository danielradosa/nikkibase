import { useEffect, useMemo, useState, useSyncExternalStore } from 'react'
import { ideals, loadAcquire, version } from '../boot'
import { engine, type WorthBase, type WorthFilter, type WorthRanking } from '../engine/engine'
import { worthSettings } from '../outfit/skills'
import { stageScores, worthVersions, type IdealTable, type Stage, type StageScore } from '../outfit/stages'
import { useStore } from '../store'
import {
  FIRST_ROWS, askSteps, filterSuits, rankReady, rankSteps, worthKey, worthRunner, worthSuits, type AcquireTable, type WorthRequest,
  type WorthRun,
} from './worth'

export const runner = worthRunner(engine.worth)

export function useWorthRun(active: boolean, request: WorthRequest | null): WorthRun {
  const state = useSyncExternalStore(runner.subscribe, runner.get)

  useEffect(() => {
    if (!request) runner.reset()
    else if (active) runner.ensure(request)
    else if (runner.get().key !== request.key) runner.reset()
  }, [active, request])

  return state
}

type Ranked = {
  key: string
  view: string
  run: number
  filter: WorthFilter
  ranking: WorthRanking | null
  error: string | null
  got: number
  of: number
  streaming: boolean
  first: boolean
}

export function useWorthRanking(run: WorthRun, active: boolean, filter: WorthFilter, limit: number) {
  const [result, setResult] = useState<Ranked | null>(null)
  const ready = rankReady(run)
  const view = `${run.run}|${run.done}|${JSON.stringify(filter)}`
  const key = `${view}|${limit}`

  useEffect(
    () =>
      askSteps(runner, run, active, filter, rankSteps(filterSuits(filter), limit > FIRST_ROWS ? FIRST_ROWS : 0, limit), (step) =>
        setResult((prev) => ({
          key,
          view,
          run: run.run,
          filter,
          ...step,
          first: prev === null || prev.run !== run.run || (prev.first && prev.key === key),
        })),
      ),
    [ready, key, active],
  )

  const usable = ready && result !== null && result.run === run.run
  const settled = usable && result.key === key
  return {
    ready,
    ranking: usable ? result.ranking : null,
    rankedFilter: usable ? result.filter : filter,
    error: settled ? result.error : null,
    loading: ready && (!settled || result.streaming),
    current: usable && result.view === view,
    streaming: settled && result.streaming,
    got: settled ? result.got : 0,
    of: settled ? result.of : 0,
    first: settled ? result.first : !usable,
  }
}

export function useIdeals(): IdealTable | null | undefined {
  const [table, setTable] = useState<IdealTable | null | undefined>(undefined)
  useEffect(() => {
    let live = true
    ideals.then((t) => live && setTable(t))
    return () => {
      live = false
    }
  }, [])
  return table
}

export type Acquire = { table: AcquireTable | null; loading: boolean }

export function useAcquire(active: boolean): Acquire {
  const [state, setState] = useState<Acquire>({ table: null, loading: true })
  useEffect(() => {
    if (!active || state.table) return
    let live = true
    setState((s) => (s.loading ? s : { ...s, loading: true }))
    loadAcquire().then((table) => live && setState({ table, loading: false }))
    return () => {
      live = false
    }
  }, [active, state.table])
  return state
}

type Scoreable = { id: number; suit: string; scoreable: boolean }

export function useWorthRequest(stages: readonly Stage[], items: readonly Scoreable[] | null): WorthRequest | null {
  const ownedIds = useStore((s) => s.owned)
  const skills = useStore((s) => s.skills)
  const table = useIdeals()
  const settings = worthSettings(skills)
  const settingsKey = JSON.stringify(settings)
  return useMemo<WorthRequest | null>(
    () =>
      ownedIds.length && table && items
        ? {
            key: worthKey(version, settings, ownedIds),
            settings,
            versions: () => worthVersions(stages, table, settings ? 'max' : 'none'),
            suits: () => worthSuits(items),
          }
        : null,
    [ownedIds, table, items, stages, settingsKey],
  )
}

export type StageScores = { scores: ReadonlyMap<string, StageScore> | null; working: boolean }

export function useStageScores(stages: readonly Stage[], request: WorthRequest | null): StageScores {
  const run = useSyncExternalStore(runner.subscribe, runner.get)
  const busy = useStore((s) => s.busy)
  const difficulty = useStore((s) => s.difficulty)
  const skills = useStore((s) => s.skills)
  const table = useIdeals()
  const [bases, setBases] = useState<{ run: number; list: WorthBase[] } | null>(null)

  useEffect(() => {
    if (request && !busy) runner.ensure(request)
  }, [request, busy])

  const current = request !== null && run.key === request.key
  useEffect(() => {
    if (!current || run.done === 0) return
    let live = true
    runner
      .bases()
      .then((list) => live && setBases({ run: run.run, list }))
      .catch(() => undefined)
    return () => {
      live = false
    }
  }, [current, run.run, run.done])

  const scored = current && bases !== null && bases.run === run.run && table ? bases.list : null
  const mode = worthSettings(skills) ? 'max' : 'none'
  const scores = useMemo(
    () => (scored && table ? stageScores(scored, stages, table, difficulty, mode) : null),
    [scored, stages, table, difficulty, mode],
  )
  return { scores, working: request !== null && (!current || run.phase === 'running' || run.phase === 'stopping') }
}
