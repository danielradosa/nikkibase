import type {
  WorthExample, WorthFilter, WorthNeeded, WorthProgress, WorthRanking, WorthRow, WorthSession, WorthSettings, WorthSuit,
  WorthVersion,
} from './engine'
import type { Difficulty } from './stages'

export type AcquireEntry = {
  k: string
  t: string
  from?: [number, number][]
  cost?: [number, string][]
  recipe?: string
  stage?: string
  level?: string
  past?: number
}
export type AcquireTable = Record<string, AcquireEntry[]>
export type AcquireFile = { version: string; items: AcquireTable }

export const ALL_MODES = 'All'
export const FIRST_ROWS = 50
export const MORE_ROWS = 200
export const MORE_SUITS = 100
export const ROW_STEP = 50

export function hashIds(ids: readonly number[]): string {
  const sorted = [...ids].sort((a, b) => a - b)
  let h = 0x811c9dc5
  for (const id of sorted) {
    for (let shift = 0; shift < 32; shift += 8) {
      h ^= (id >>> shift) & 0xff
      h = Math.imul(h, 0x01000193)
    }
  }
  return `${sorted.length}.${(h >>> 0).toString(36)}`
}

export const worthKey = (version: string, settings: WorthSettings, owned: readonly number[]) =>
  `${version}|${JSON.stringify(settings)}|${hashIds(owned)}`

export function worthFilter(
  mode: string,
  where: { slots?: number[]; places?: number[] },
  skip: readonly string[],
  bySuit = false,
): WorthFilter {
  const filter: WorthFilter = {}
  if (mode !== ALL_MODES) filter.modes = [mode]
  if (where.slots) filter.slots = [...where.slots]
  if (where.places) filter.places = [...where.places]
  if ((mode === ALL_MODES || mode === 'Story') && skip.length) filter.skip = [...skip]
  if (bySuit) filter.suits = true
  return filter
}

export const filterMode = (filter: WorthFilter) => filter.modes?.[0] ?? ALL_MODES

export const filterSuits = (filter: WorthFilter) => filter.suits === true

export const rankingStatus = (bySuit: boolean) => (bySuit ? 'Ranking suits…' : 'Ranking items…')

export const waitPercent = (done: number, total: number) => (total ? Math.floor((done / total) * 100) : 0)

export const checkingText = (done: number, total: number) =>
  `Step 1 of 2 · Checking stages ${done.toLocaleString('en-US')} of ${total.toLocaleString('en-US')}`

export function rankingText({
  suits,
  mode,
  where,
  got,
  of,
  first,
}: {
  suits: boolean
  mode: string
  where?: string | null
  got?: number
  of?: number
  first: boolean
}): string {
  const what = suits ? 'suits' : 'items'
  const count = got && of ? ` · ${got.toLocaleString('en-US')} of ${of.toLocaleString('en-US')}` : ''
  if (first) return `Step 2 of 2 · Ranking ${what}${count}`
  return `Ranking ${mode === ALL_MODES ? what : `${mode} ${what}`}${where ? ` for ${where}` : ''}${count}`
}

export const MORE_WAIT = 'Ranking 50 more…'

export function rankSteps(suits: boolean, from: number, to: number): number[] {
  if (!suits) return from === 0 && to > 10 ? [10, to] : [to]
  const steps: number[] = []
  for (let n = from + 10; n < to; n += 10) steps.push(n)
  return [...steps, to]
}

export function worthSuits(items: readonly { id: number; suit: string; scoreable: boolean }[]): WorthSuit[] {
  const bySuit = new Map<string, number[]>()
  for (const it of items) {
    if (!it.suit || !it.scoreable) continue
    const ids = bySuit.get(it.suit)
    if (ids) ids.push(it.id)
    else bySuit.set(it.suit, [it.id])
  }
  return [...bySuit.keys()].sort((a, b) => (a < b ? -1 : a > b ? 1 : 0)).map((key) => ({ key, items: bySuit.get(key)! }))
}

export const rowKey = (row: Pick<WorthRow, 'suit' | 'items'>) => (row.suit ? `suit:${row.suit}` : row.items.join('+'))

export function rowName(row: Pick<WorthRow, 'suit' | 'items'>, names: ReadonlyMap<number, string>): string {
  return row.suit ?? row.items.map((id) => names.get(id) ?? `#${id}`).join(' + ')
}

export const rankedName = (rank: number, name: string) => `${rank}. ${name}`

export const detailsLabel = (name: string) => `Show details for ${name}`

export const hideLabel = (name: string) => `Hide details for ${name}`

export function openRows(keys: readonly string[], key: string, open: boolean, single: boolean): string[] {
  if (!open) return keys.filter((k) => k !== key)
  if (single) return [key]
  return keys.includes(key) ? [...keys] : [...keys, key]
}

export function piecesText(count: number): string {
  return count === 1 ? "1 piece you don't own" : `${count.toLocaleString('en-US')} pieces you don't own`
}

export type Piece = { id: number; name: string; place: string; meta: string }

export function suitPieces(
  row: Pick<WorthRow, 'items' | 'places'>,
  items: ReadonlyMap<number, { name: string; rarity: number }>,
  places: readonly { name: string }[],
): Piece[] {
  return row.items.map((id, i) => {
    const it = items.get(id)
    const where = places[row.places[i]]?.name ?? ''
    return { id, name: it?.name ?? `#${id}`, place: where, meta: [where, it?.rarity ? '★'.repeat(it.rarity) : ''].filter(Boolean).join(' · ') }
  })
}

export type PieceGroup = { key: string; text: string; past: boolean; pieces: Piece[] }

export function groupPieces(pieces: readonly Piece[], table: AcquireTable): (Piece | PieceGroup)[] {
  const blocks: (Piece | PieceGroup)[] = []
  const groups = new Map<string, PieceGroup>()
  for (const piece of pieces) {
    const ways = table[String(piece.id)] ?? []
    const plain = ways.length === 1 && !ways[0].from?.length && !ways[0].recipe && !ways[0].stage
    if (ways.length && !plain) {
      blocks.push(piece)
      continue
    }
    const text = ways[0]?.t ?? NO_SOURCE
    const past = ways[0]?.past === 1
    const key = `${past ? 1 : 0}|${text}`
    const group = groups.get(key)
    if (group) {
      group.pieces.push(piece)
    } else {
      const made = { key, text, past, pieces: [piece] }
      groups.set(key, made)
      blocks.push(made)
    }
  }
  return blocks
}

const NAME_PLACES = new Map<string, readonly string[]>([
  ['handheld', ['held, right', 'held, left', 'held, both hands']],
  ['head ornament', ['hair ornament']],
  ['bracelet', ['right hand', 'left hand']],
  ['leglet', ['leglets']],
])

const namesPlace = (name: string, place: string) => {
  const end = /\(([^()]+)\)$/.exec(name)?.[1].toLowerCase()
  const at = place.toLowerCase()
  return end !== undefined && (end === at || (NAME_PLACES.get(end)?.includes(at) ?? false))
}

export function pieceList(pieces: readonly Pick<Piece, 'name' | 'place'>[]): string {
  return pieces.map((p) => (p.place && !namesPlace(p.name, p.place) ? `${p.name} (${p.place})` : p.name)).join(', ')
}

export function groupTail(group: Pick<PieceGroup, 'past' | 'pieces'>): string {
  return `${group.past ? `· ${PAST_NOTE} ` : ''}(${group.pieces.length.toLocaleString('en-US')})`
}

export function rankingNote(bySuit: boolean, narrowed = false): string {
  if (!bySuit) {
    return "Ranked by total gain: the % an item adds on each stage, added up (the bar). A small gain on many stages can rank high. % is of each stage's best possible score. Each row assumes you got the ones above it."
  }
  const note =
    "Ranked by total gain if you get every missing piece of a suit (the bar). % is of each stage's best possible score. Each row assumes you got the suits above it. Items not in a suit show with group by suit off."
  return narrowed ? `${note} A suit shows if one of its pieces fits the slot. Its gain counts every piece.` : note
}

export const UNLOCK_NOTE =
  'Ranked by stages unlocked per item. Items you can still get come first. Each row assumes you got the ones above it.'

export function nothingText(bySuit: boolean): string {
  return bySuit
    ? 'No suit you could complete would raise your best on these stages.'
    : 'Nothing you could get would raise your best on these stages.'
}

function splitKey(key: string) {
  const hash = key.indexOf('#')
  const base = hash < 0 ? key : key.slice(0, hash)
  const slash = base.indexOf('/')
  return {
    base,
    variant: hash < 0 ? '' : key.slice(hash + 1),
    mode: slash < 0 ? base : base.slice(0, slash),
    name: slash < 0 ? '' : base.slice(slash + 1),
  }
}

export function stageLabel(key: string, variants: ReadonlySet<string> = new Set()): string {
  const { base, variant, mode, name } = splitKey(key)
  const text = name ? `${mode} ${name}` : mode
  if (variant === 'maiden') return `${text} (Maiden)`
  return variants.has(base) ? `${text} (Princess)` : text
}

export type OpenTarget = { mode: string; stage: string; difficulty: Difficulty | null }

export function openTarget(key: string, variants: ReadonlySet<string> = new Set()): OpenTarget {
  const { base, variant, mode } = splitKey(key)
  const difficulty = variant === 'maiden' ? 'Maiden' : variants.has(base) ? 'Princess' : null
  return { mode, stage: base, difficulty }
}

function levelOf(level: string | undefined): Difficulty | null {
  return level === 'Maiden' || level === 'Princess' ? level : null
}

export function pctText(v: number): string {
  if (!(v > 0)) return '0'
  if (v >= 0.1) return v.toFixed(1)
  return String(Number(v.toPrecision(1)))
}

export function improvesParts(row: WorthRow, mode: string, variants?: ReadonlySet<string>): [string, string, string] {
  const best = stageLabel(row.best.key, variants)
  if (row.stages <= 1) return [`+${pctText(row.best.pct)}% on `, best, '']
  const kind = mode === ALL_MODES ? 'stages' : `${mode} stages`
  const average = pctText(row.worth / row.stages)
  return [`+${average}% on ${row.stages.toLocaleString('en-US')} ${kind} · best +${pctText(row.best.pct)}% on `, best, '']
}

export function improvesText(row: WorthRow, mode: string, variants?: ReadonlySet<string>): string {
  return improvesParts(row, mode, variants).join('')
}

export const SCORE_F = 'may score F'

export function gainText(ex: Pick<WorthExample, 'points' | 'pct'>, flagged = false): string {
  const gain = `+${ex.points.toLocaleString('en-US')} (+${pctText(ex.pct)}%)`
  return flagged ? `${gain} · ${SCORE_F}` : gain
}

export function scoreFNote(gain: boolean): string {
  const note = "may score F: some items score F on this stage and NikkiBase doesn't check it"
  return gain ? `${note}, so the real gain may be smaller.` : `${note}.`
}

export function neededLine(count: number): string {
  return count === 1
    ? "1 stage you can't pass yet is left out."
    : `${count.toLocaleString('en-US')} stages you can't pass yet are left out.`
}

export function itemMeta(row: Pick<WorthRow, 'items' | 'places'>, places: readonly { name: string }[], rarity: number): string {
  const where = row.places.map((pos) => places[pos]?.name ?? '')
  if (row.items.length > 1) return `${where.join(' + ')} · worth more together`
  return [where[0], rarity ? '★'.repeat(rarity) : ''].filter(Boolean).join(' · ')
}

export type UnlockRow = { items: number[]; stages: string[] }

export function unlockRanking(needed: readonly WorthNeeded[], hard: (id: number) => boolean = () => false): UnlockRow[] {
  const got = new Set<number>()
  let locked = needed.map((n, order) => ({ key: n.key, order, sets: n.missing }))
  const rows: UnlockRow[] = []
  const open = (sets: readonly (readonly number[])[], extra: ReadonlySet<number>) =>
    sets.filter((set) => !set.some((id) => got.has(id) || extra.has(id)))
  while (locked.length) {
    const seen = new Map<number, number>()
    for (const stage of locked) for (const set of open(stage.sets, new Set())) for (const id of set) seen.set(id, (seen.get(id) ?? 0) + 1)
    const pick = (set: readonly number[]) =>
      set.reduce((best, id) => {
        const a = seen.get(id) ?? 0
        const b = seen.get(best) ?? 0
        return a > b || (a === b && id < best) ? id : best
      })
    const candidates: { items: number[]; stages: typeof locked }[] = []
    const tried = new Set<string>()
    for (const stage of locked) {
      const items = [...new Set(open(stage.sets, new Set()).map(pick))].sort((a, b) => a - b)
      const id = items.join('+')
      if (tried.has(id)) continue
      tried.add(id)
      const adds = new Set(items)
      candidates.push({ items, stages: locked.filter((other) => open(other.sets, adds).length === 0) })
    }
    const rate = (a: (typeof candidates)[number], b: (typeof candidates)[number]) =>
      a.stages.length * b.items.length - b.stages.length * a.items.length
    const inside = (a: readonly number[], b: readonly number[]) => a.length < b.length && a.every((id) => b.includes(id))
    const kept = candidates.filter((c) => !candidates.some((d) => inside(d.items, c.items) && rate(d, c) >= 0))
    const easy = (c: (typeof candidates)[number]) => !c.items.some(hard)
    const better = (c: (typeof candidates)[number], b: (typeof candidates)[number]) => {
      if (easy(c) !== easy(b)) return easy(c)
      const r = rate(c, b)
      if (r !== 0) return r > 0
      if (c.stages.length !== b.stages.length) return c.stages.length > b.stages.length
      if (c.items.length !== b.items.length) return c.items.length < b.items.length
      return c.stages[0].order < b.stages[0].order
    }
    let best: (typeof candidates)[number] | null = null
    for (const c of kept) if (!best || better(c, best)) best = c
    if (!best) break
    rows.push({ items: best.items, stages: best.stages.map((s) => s.key) })
    for (const id of best.items) got.add(id)
    const done = new Set(best.stages)
    locked = locked.filter((s) => !done.has(s))
  }
  return rows
}

export function hardToGet(entries: readonly AcquireEntry[] | undefined): boolean {
  return !entries?.length || entries.every((e) => e.past === 1)
}

export function unlockText(row: UnlockRow, variants?: ReadonlySet<string>): string {
  if (row.stages.length === 1) return `Unlocks ${stageLabel(row.stages[0], variants)}`
  return `Unlocks ${row.stages.length.toLocaleString('en-US')} stages`
}

export type Ingredient = { id: number; name: string; qty: number; owned: boolean }
export type HowLine = { text: string; recipe: string | null; from: Ingredient[]; past: boolean; stage: OpenTarget | null }

export const PAST_NOTE = 'may have ended'

export const NO_SOURCE = 'Source unknown'

export const OWNED_KEY = '✓ = in your wardrobe (you may need more copies)'

export function recipeText(recipe: string): string {
  return recipe === 'Available by default' ? 'Recipe: unlocked from the start' : `Recipe from ${recipe}`
}

export function chipText(part: Pick<Ingredient, 'qty' | 'name' | 'owned'>): string {
  return `${part.qty.toLocaleString('en-US')}× ${part.name}${part.owned ? ' ✓' : ''}`
}

function verbLine(e: AcquireEntry): string {
  const colon = e.t.indexOf(':')
  const verb = colon < 0 ? e.t : e.t.slice(0, colon)
  const cost = (e.cost ?? []).map(([qty, what]) => `${qty.toLocaleString('en-US')} ${what}`).join(' + ')
  return cost ? `${verb} · ${cost}` : verb
}

function ingredients(
  from: readonly [number, number][],
  owned: ReadonlySet<number>,
  names: ReadonlyMap<number, string>,
): Ingredient[] {
  const byId = new Map<number, Ingredient>()
  for (const [id, qty] of from) {
    const seen = byId.get(id)
    if (seen) seen.qty += qty
    else byId.set(id, { id, name: names.get(id) ?? `#${id}`, qty, owned: owned.has(id) })
  }
  return [...byId.values()]
}

export function howToGet(
  entries: readonly AcquireEntry[] | undefined,
  owned: ReadonlySet<number>,
  names: ReadonlyMap<number, string>,
): HowLine[] {
  return (entries ?? []).map((e) => {
    const stage = e.stage ? splitKey(e.stage) : null
    return {
      text: e.from?.length ? verbLine(e) : e.t,
      recipe: e.recipe ?? null,
      from: ingredients(e.from ?? [], owned, names),
      past: e.past === 1,
      stage: stage ? { mode: stage.mode, stage: stage.base, difficulty: levelOf(e.level) } : null,
    }
  })
}

export function ownsAnyPart(lines: readonly HowLine[]): boolean {
  return lines.some((line) => line.from.some((part) => part.owned))
}

export type WorthApi = {
  start: (versions: WorthVersion[], settings: WorthSettings, suits: WorthSuit[]) => Promise<WorthSession>
  run: (session: number, count: number) => Promise<WorthProgress>
  rank: (session: number, filter: WorthFilter, limit: number) => Promise<WorthRanking>
}

export type WorthPhase = 'idle' | 'running' | 'stopping' | 'stopped' | 'done' | 'failed'
export type WorthRun = { key: string | null; run: number; phase: WorthPhase; done: number; total: number; error: string | null }
export type WorthRequest = { key: string; settings: WorthSettings; versions: () => WorthVersion[]; suits?: () => WorthSuit[] }

const message = (e: unknown) => (e instanceof Error ? e.message : String(e))

export const noSession = (e: unknown) => message(e).includes('no session')

export const RESTARTED = 'Ranking stopped unexpectedly. Try again.'

export function worthRunner(api: WorthApi, chunk = 32, retries = 3) {
  let state: WorthRun = { key: null, run: 0, phase: 'idle', done: 0, total: 0, error: null }
  let request: WorthRequest | null = null
  let versions: WorthVersion[] = []
  let suits: WorthSuit[] = []
  let session: number | null = null
  let looping = -1
  let restarts = 0
  let ranks = new Map<string, Promise<WorthRanking>>()
  const listeners = new Set<() => void>()

  const phase = () => state.phase

  function update(patch: Partial<WorthRun>) {
    state = { ...state, ...patch }
    for (const listener of listeners) listener()
  }

  function lost(e: unknown) {
    if (noSession(e) && request && restarts < retries) {
      restarts++
      begin()
      return
    }
    update({ phase: 'failed', error: noSession(e) ? RESTARTED : message(e) })
  }

  async function loop(run: number) {
    looping = run
    try {
      if (session === null) {
        const started = await api.start(versions, request?.settings ?? null, suits)
        if (run !== state.run) return
        session = started.session
        update({ total: started.total, ...(state.phase === 'stopping' ? { phase: 'stopped' as const } : {}) })
      }
      while (run === state.run && state.phase === 'running') {
        const progress = await api.run(session, chunk)
        if (run !== state.run) return
        const finished = progress.done >= progress.total
        if (finished) restarts = 0
        const now = phase()
        update({ done: progress.done, total: progress.total, phase: finished ? 'done' : now === 'stopping' ? 'stopped' : now })
      }
    } catch (e) {
      if (run === state.run) lost(e)
    } finally {
      if (looping === run) looping = -1
    }
  }

  function begin() {
    if (!request) return
    session = null
    ranks = new Map()
    versions = request.versions()
    suits = request.suits?.() ?? []
    const run = state.run + 1
    update({ key: request.key, run, phase: 'running', done: 0, total: versions.length, error: null })
    void loop(run)
  }

  function ensure(next: WorthRequest) {
    request = next
    const live = state.phase === 'running' || state.phase === 'stopping' || state.phase === 'stopped' || state.phase === 'done'
    if (state.key === next.key && live) return
    restarts = 0
    begin()
  }

  function stop() {
    if (state.phase === 'running') update({ phase: looping === state.run ? 'stopping' : 'stopped' })
  }

  function resume() {
    if ((state.phase !== 'stopped' && state.phase !== 'stopping') || !request) return
    update({ phase: 'running' })
    if (looping !== state.run) void loop(state.run)
  }

  function retry() {
    restarts = 0
    begin()
  }

  function reset() {
    request = null
    session = null
    versions = []
    suits = []
    ranks = new Map()
    if (state.key === null && state.phase === 'idle') return
    update({ key: null, run: state.run + 1, phase: 'idle', done: 0, total: 0, error: null })
  }

  function rank(filter: WorthFilter, limit: number): Promise<WorthRanking> {
    const run = state.run
    if (session === null || (state.phase !== 'done' && state.phase !== 'stopped')) {
      return Promise.reject(new Error('the ranking is not ready'))
    }
    const key = `${run}|${state.done}|${limit}|${JSON.stringify(filter)}`
    let pending = ranks.get(key)
    if (!pending) {
      const asked = api.rank(session, filter, limit)
      pending = asked
      ranks.set(key, asked)
      asked.catch((e) => {
        if (ranks.get(key) === asked) ranks.delete(key)
        if (run === state.run) lost(e)
      })
    }
    return pending
  }

  return {
    get: () => state,
    subscribe: (listener: () => void) => {
      listeners.add(listener)
      return () => {
        listeners.delete(listener)
      }
    },
    ensure,
    stop,
    resume,
    retry,
    reset,
    rank,
  }
}

export type WorthRunner = ReturnType<typeof worthRunner>

export const rankReady = (run: WorthRun) =>
  (run.phase === 'done' || run.phase === 'stopped') && (run.done > 0 || run.total === 0)

export function askRanking(
  runner: Pick<WorthRunner, 'rank'>,
  run: WorthRun,
  active: boolean,
  filter: WorthFilter,
  limit: number,
  settle: (ranking: WorthRanking | null, error: string | null) => void,
): (() => void) | undefined {
  if (!active || !rankReady(run)) return undefined
  let live = true
  runner.rank(filter, limit).then(
    (ranking) => live && settle(ranking, null),
    (e) => {
      if (live && !noSession(e)) settle(null, message(e))
    },
  )
  return () => {
    live = false
  }
}

export type Stepped = { ranking: WorthRanking | null; error: string | null; got: number; of: number; streaming: boolean }

export function askSteps(
  runner: Pick<WorthRunner, 'rank'>,
  run: WorthRun,
  active: boolean,
  filter: WorthFilter,
  steps: readonly number[],
  settle: (step: Stepped) => void,
): (() => void) | undefined {
  if (!active || !rankReady(run) || !steps.length) return undefined
  let live = true
  const of = steps[steps.length - 1]
  const ask = async () => {
    for (let i = 0; i < steps.length && live; i++) {
      let ranking: WorthRanking
      try {
        ranking = await runner.rank(filter, steps[i])
      } catch (e) {
        if (live && !noSession(e)) settle({ ranking: null, error: message(e), got: 0, of, streaming: false })
        return
      }
      if (!live) return
      const got = ranking.rows.length
      const streaming = i < steps.length - 1 && got >= steps[i]
      settle({ ranking, error: null, got, of, streaming })
      if (!streaming) return
    }
  }
  void ask()
  return () => {
    live = false
  }
}
