import { useEffect, useState, useSyncExternalStore } from 'react'
import { ideals, loadAcquire } from './boot'
import { engine, type WorthFilter, type WorthRanking } from './engine'
import type { IdealTable } from './stages'
import { askRanking, rankReady, worthRunner, type AcquireTable, type WorthRequest, type WorthRun } from './worth'

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
}

export function useWorthRanking(run: WorthRun, active: boolean, filter: WorthFilter, limit: number) {
  const [result, setResult] = useState<Ranked | null>(null)
  const ready = rankReady(run)
  const view = `${run.run}|${run.done}|${JSON.stringify(filter)}`
  const key = `${view}|${limit}`

  useEffect(
    () =>
      askRanking(runner, run, active, filter, limit, (ranking, error) =>
        setResult({ key, view, run: run.run, filter, ranking, error }),
      ),
    [ready, key, active],
  )

  const usable = ready && result !== null && result.run === run.run
  return {
    ready,
    ranking: usable ? result.ranking : null,
    rankedFilter: usable ? result.filter : filter,
    error: usable && result.key === key ? result.error : null,
    loading: ready && result?.key !== key,
    current: usable && result.view === view,
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
