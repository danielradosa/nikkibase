import { useEffect, useState, useSyncExternalStore } from 'react'
import { ideals, loadAcquire } from './boot'
import { engine, type WorthFilter, type WorthRanking } from './engine'
import type { IdealTable } from './stages'
import {
  FIRST_ROWS, askSteps, filterSuits, rankReady, rankSteps, worthRunner, type AcquireTable, type WorthRequest, type WorthRun,
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
