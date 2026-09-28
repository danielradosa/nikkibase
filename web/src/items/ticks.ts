export type Stats = { items: number; known: number; unresolved: number }

export type TickDeps = {
  owned: () => number[]
  show: (next: number[]) => void
  send: (ids: number[]) => Promise<{ items: number; known: number }>
  settle: (stats: Stats) => void
  save: (ids: number[]) => Promise<void>
  fail: (confirmed: number[], error: unknown, ticked: boolean) => void
}

export type TickState = { waiting: number; confirmed: number[]; ticked: boolean; stats: Stats | null }

export function tickState(): TickState {
  return { waiting: 0, confirmed: [], ticked: false, stats: null }
}

export function toggled(owned: readonly number[], id: number): number[] {
  return owned.includes(id) ? owned.filter((x) => x !== id) : [...owned, id].sort((a, b) => a - b)
}

export async function tick(id: number, deps: TickDeps, state: TickState): Promise<void> {
  const previous = deps.owned()
  if (state.waiting === 0) {
    state.confirmed = previous
    state.ticked = false
    state.stats = null
  }
  state.waiting += 1
  const next = toggled(previous, id)
  deps.show(next)
  let stats: { items: number; known: number }
  try {
    stats = await deps.send(next)
  } catch (e) {
    state.waiting -= 1
    if (deps.owned() !== next) return
    const { confirmed, ticked, stats: accepted } = state
    deps.fail(confirmed, e, ticked)
    if (!ticked) return
    if (accepted) deps.settle(accepted)
    await deps.save(confirmed)
    return
  }
  state.waiting -= 1
  state.confirmed = next
  state.ticked = true
  state.stats = { items: stats.items, known: stats.known, unresolved: 0 }
  if (deps.owned() !== next) return
  deps.settle(state.stats)
  await deps.save(next)
}
