import type { SkillChoice, WorthVersion } from '../engine/engine'

export type Difficulty = 'Maiden' | 'Princess'

export type Rules = { styles?: string[]; require?: number[][] }

type Scoring = {
  weights: number[]
  attrs: number[]
  tags?: Record<string, number>
  rules?: Rules
}

export type Stage = Scoring & {
  name: string
  mode: 'Story' | 'Commission' | 'Dreamweaver' | 'Co-op' | 'Arena'
  variants?: { maiden?: Scoring }
}

export const stageKey = (s: Stage) => `${s.mode}/${s.name}`

const MODE_ORDER = ['Story', 'Commission', 'Dreamweaver', 'Co-op', 'Arena']

const MODE_LABELS: Record<string, string> = { Dreamweaver: 'Dream Weaver' }

export const modeLabel = (mode: string) => MODE_LABELS[mode] ?? mode

export function orderModes(stages: readonly Stage[]): string[] {
  const present = new Set<string>(stages.map((s) => s.mode ?? 'Other'))
  return MODE_ORDER.filter((m) => present.has(m)).concat(
    [...present].filter((m) => !MODE_ORDER.includes(m)).sort(),
  )
}

export function stagesInMode(stages: readonly Stage[], mode: string): Stage[] {
  return stages.filter((s) => (s.mode ?? 'Other') === mode)
}

export function hasVariants(s: Stage): boolean {
  return s.mode === 'Story' && s.variants?.maiden !== undefined
}

export function variantOf(s: Stage, d: Difficulty): 'maiden' | null {
  return d === 'Maiden' && s.mode === 'Story' && s.variants?.maiden ? 'maiden' : null
}

export function placementKey(s: Stage, d: Difficulty): string {
  const variant = variantOf(s, d)
  return variant ? `${stageKey(s)}#${variant}` : stageKey(s)
}

export function resolveStage(s: Stage, d: Difficulty): Stage {
  const maiden = variantOf(s, d) && s.variants?.maiden
  if (!maiden) return s
  return { ...s, weights: maiden.weights, attrs: maiden.attrs, tags: maiden.tags, rules: maiden.rules ?? s.rules }
}

export function rulesUnchecked(rules: Rules | undefined): boolean {
  return rules !== undefined && (Boolean(rules.styles?.length) || !rules.require?.length)
}

const CAVEAT = "Some items or styles score F on this stage. NikkiBase doesn't check this yet, so a suggested item could fail."

export function caveatText(styles: readonly string[]): string {
  return styles.length ? `${CAVEAT} Styles: ${styles.join(', ')}.` : CAVEAT
}

export function weightLabel(attr: string, weight: number): string {
  return `${attr} ×${Math.round(weight * 100) / 100}`
}

export function tagLabel(name: string, award: number): string {
  return `${name} +${award.toLocaleString('en-US')} each`
}

const itemName = (id: number, names: ReadonlyMap<number, string>) => names.get(id) ?? `#${id}`

export function requirementLabel(sets: readonly (readonly number[])[], names: ReadonlyMap<number, string>): string {
  return list(sets.map((set) => list(set.map((id) => itemName(id, names)), 'or')))
}

export function missingMessage(missing: readonly (readonly number[])[], names: ReadonlyMap<number, string>): string {
  return `You don't own ${requirementLabel(missing, names)}, which this stage requires`
}

export type Ideal = { score: number; items: { id: number; pos: number }[]; skills?: SkillChoice }

type StoredOutfit = { score: number; items: [number, number][] }

type StoredIdeal = StoredOutfit & { auto?: StoredOutfit & SkillChoice }

export type IdealTable = Record<string, StoredIdeal & { variants?: Record<string, StoredIdeal> }>

export function lookupIdeal(table: IdealTable | null, s: Stage, d: Difficulty, skills: 'none' | 'max' = 'none'): Ideal | null {
  const entry = table?.[stageKey(s)]
  const variant = variantOf(s, d)
  const chosen = variant ? entry?.variants?.[variant] : entry
  if (skills === 'max') {
    const auto = chosen?.auto
    if (!auto) return null
    return {
      score: auto.score,
      items: auto.items.map(([id, pos]) => ({ id, pos })),
      skills: { charmSmile: auto.charmSmile, smile: auto.smile },
    }
  }
  if (!chosen) return null
  return { score: chosen.score, items: chosen.items.map(([id, pos]) => ({ id, pos })) }
}

export function variantStages(stages: readonly Stage[]): Set<string> {
  return new Set(stages.filter(hasVariants).map(stageKey))
}

export function worthVersions(stages: readonly Stage[], table: IdealTable, skills: 'none' | 'max'): WorthVersion[] {
  const out: WorthVersion[] = []
  for (const mode of orderModes(stages)) {
    for (const s of stagesInMode(stages, mode)) {
      const difficulties: Difficulty[] = hasVariants(s) ? ['Princess', 'Maiden'] : ['Princess']
      for (const d of difficulties) {
        const ideal = lookupIdeal(table, s, d, skills)?.score
        if (!ideal || ideal < 1 || !Number.isInteger(ideal)) continue
        const r = resolveStage(s, d)
        const version: WorthVersion = { key: placementKey(s, d), mode: s.mode, weights: r.weights, attrs: r.attrs, ideal }
        if (r.tags && Object.keys(r.tags).length) version.tags = r.tags
        if (r.rules?.require?.length) version.require = r.rules.require
        out.push(version)
      }
    }
  }
  return out
}

export function worthSkip(stages: readonly Stage[], d: Difficulty): string[] {
  return stages.filter(hasVariants).map((s) => placementKey(s, d === 'Maiden' ? 'Princess' : 'Maiden'))
}

const ROMAN = ['', 'I', 'II', 'III']

type StoryPlace = { volume: number; chapter: number; side: boolean; stage: number; round: number }

const STORY_NAME = /^(?:(II|III)-)?(\d+)-(?:side\s*(\d+)|(\d+))(?:-(\d+))?$/i

function storyPlace(name: string): StoryPlace | null {
  const m = STORY_NAME.exec(name.trim())
  if (!m) return null
  return {
    volume: m[1] ? ROMAN.indexOf(m[1].toUpperCase()) : 1,
    chapter: Number(m[2]),
    side: m[3] !== undefined,
    stage: Number(m[3] ?? m[4]),
    round: Number(m[5] ?? 0),
  }
}

function commissionPlace(name: string): { act: number; stage: number } | null {
  const m = /^(\d+)-(\d+)$/.exec(name.trim())
  return m ? { act: Number(m[1]), stage: Number(m[2]) } : null
}

function sortKey(s: Stage): number[] | null {
  if (s.mode === 'Story') {
    const p = storyPlace(s.name)
    return p && [p.volume, p.chapter, p.side ? 1 : 0, p.stage, p.round]
  }
  if (s.mode === 'Commission') {
    const p = commissionPlace(s.name)
    return p && [p.act, p.stage]
  }
  return []
}

export function compareStages(a: Stage, b: Stage): number {
  if (a.mode !== b.mode) return a.mode < b.mode ? -1 : 1
  const ka = sortKey(a)
  const kb = sortKey(b)
  if (ka && kb) {
    for (let i = 0; i < Math.min(ka.length, kb.length); i++) if (ka[i] !== kb[i]) return ka[i] - kb[i]
    return 0
  }
  if (ka || kb) return ka ? -1 : 1
  return a.name.localeCompare(b.name, 'en', { numeric: true })
}

export type StageLeaf = { value: string; label: string }
export type StageGroup = { label: string; options: StageLeaf[] }

function groupLabel(s: Stage): string | null {
  if (s.mode === 'Story') {
    const p = storyPlace(s.name)
    return p ? `Volume ${ROMAN[p.volume]} · Chapter ${p.chapter}` : 'Other'
  }
  if (s.mode === 'Commission') {
    const p = commissionPlace(s.name)
    return p ? `Act ${p.act}` : 'Other'
  }
  return null
}

export function groupOptions(stages: readonly Stage[]): (StageGroup | StageLeaf)[] {
  const out: (StageGroup | StageLeaf)[] = []
  const groups = new Map<string, StageGroup>()
  for (const s of [...stages].sort(compareStages)) {
    const leaf = { value: stageKey(s), label: s.name }
    const label = groupLabel(s)
    if (label === null) {
      out.push(leaf)
      continue
    }
    let group = groups.get(label)
    if (!group) {
      group = { label, options: [] }
      groups.set(label, group)
      out.push(group)
    }
    group.options.push(leaf)
  }
  return out
}

const fold = (text: string) => text.trim().toLowerCase().replace(/[\s-]+/g, '-')

const VOLUME_QUERY = /^v(?:ol(?:ume)?)?\.?[\s-]*([1-3]|i{1,3})(?![a-z\d])[\s-]*(.*)$/i

export function matches(input: string, option?: { value?: unknown; label?: unknown; options?: unknown }): boolean {
  if (!option || option.options !== undefined) return false
  const name = String(option.label ?? '')
  const query = fold(input)
  if (fold(name).includes(query)) return true

  const q = VOLUME_QUERY.exec(input.trim())
  if (!q || !String(option.value ?? '').startsWith('Story/')) return false
  const place = storyPlace(name)
  const volume = /\d/.test(q[1]) ? Number(q[1]) : q[1].length
  if (!place || place.volume !== volume) return false
  return `-${fold(name.replace(/^(II|III)-/i, ''))}`.includes(`-${fold(q[2])}`)
}

const span = (from: string, to: string) => (from === to ? from : `${from}\u2060–\u2060${to}`)

function list(parts: string[], last = 'and'): string {
  return parts.length < 2 ? parts.join('') : `${parts.slice(0, -1).join(', ')} ${last} ${parts[parts.length - 1]}`
}

export function coverageLabel(stages: readonly Stage[]): string {
  const parts: string[] = []
  const modes = new Set(stages.map((s) => s.mode as string))

  if (modes.has('Story')) {
    const places = stages
      .filter((s) => s.mode === 'Story')
      .map((s) => storyPlace(s.name))
      .filter((p): p is StoryPlace => p !== null)
    if (places.length) {
      const last = places.reduce((a, b) =>
        b.volume > a.volume || (b.volume === a.volume && b.chapter > a.chapter) ? b : a,
      )
      const volume = last.volume === 1 ? '' : `Volume ${ROMAN[last.volume]} `
      parts.push(`Story up to ${volume}Chapter ${last.chapter}`)
    } else {
      parts.push('Story')
    }
  }

  if (modes.has('Commission')) {
    const acts = stages
      .filter((s) => s.mode === 'Commission')
      .map((s) => commissionPlace(s.name)?.act)
      .filter((a): a is number => a !== undefined)
    if (acts.length) {
      const [lo, hi] = [Math.min(...acts), Math.max(...acts)]
      parts.push(`Commission ${lo === hi ? 'Act' : 'Acts'}\u00a0${span(String(lo), String(hi))}`)
    } else {
      parts.push('Commission')
    }
  }

  const named = ['Story', 'Commission', 'Arena', 'Co-op']
  parts.push(...named.slice(2).filter((m) => modes.has(m)))
  parts.push(...[...modes].filter((m) => !named.includes(m)).sort().map(modeLabel))

  return parts.length ? `Has ${list(parts)}. Newer stages aren't in yet.` : ''
}

const EXAMPLES: Record<string, string> = { Story: '5-11', Commission: '3-7' }

export function stagePlaceholder(mode: string): string {
  const example = EXAMPLES[mode]
  const label = modeLabel(mode)
  return `Pick ${/^[AEIOU]/.test(label) ? 'an' : 'a'} ${label} stage${example ? `, e.g. ${example}` : ''}`
}

export type PickerLeaf = { key: string; label: string; name: string }
export type PickerGroup = { key: string; label: string; leaves: PickerLeaf[] }
export type PickerBranch = { key: string; label: string; groups: PickerGroup[] }

const CHIP_MODES = new Set(['Story', 'Commission', 'Dreamweaver'])

function chipPlace(s: Stage): { branch: string; group: string; label: string } {
  if (s.mode === 'Story') {
    const p = storyPlace(s.name)
    if (!p) return { branch: 'Other', group: 'Other', label: s.name }
    return {
      branch: ROMAN[p.volume],
      group: String(p.chapter),
      label: p.side ? `Side ${p.stage}` : s.name.replace(/^(II|III)-/i, ''),
    }
  }
  if (s.mode === 'Commission') {
    const p = commissionPlace(s.name)
    return { branch: '', group: p ? String(p.act) : 'Other', label: s.name }
  }
  const cut = s.name.indexOf(' - ')
  return cut < 0 ? { branch: '', group: s.name, label: s.name } : { branch: '', group: s.name.slice(0, cut), label: s.name.slice(cut + 3) }
}

export function pickerTree(stages: readonly Stage[], mode: string): PickerBranch[] {
  if (!CHIP_MODES.has(mode)) return []
  const out: PickerBranch[] = []
  for (const s of [...stagesInMode(stages, mode)].sort(compareStages)) {
    const place = chipPlace(s)
    let branch = out.find((b) => b.label === place.branch)
    if (!branch) {
      branch = { key: `${mode}/${place.branch}`, label: place.branch, groups: [] }
      out.push(branch)
    }
    let group = branch.groups.find((g) => g.label === place.group)
    if (!group) {
      group = { key: `${branch.key}/${place.group}`, label: place.group, leaves: [] }
      branch.groups.push(group)
    }
    group.leaves.push({ key: stageKey(s), label: place.label, name: s.name })
  }
  return out
}

export function placeOf(tree: readonly PickerBranch[], key: string): { branch: string; group: string } | null {
  for (const b of tree) for (const g of b.groups) if (g.leaves.some((l) => l.key === key)) return { branch: b.key, group: g.key }
  return null
}

export function stepStage(stages: readonly Stage[], mode: string, key: string, dir: 1 | -1): string | null {
  const keys = [...stagesInMode(stages, mode)].sort(compareStages).map(stageKey)
  const at = keys.indexOf(key)
  if (at < 0) return null
  return keys[at + dir] ?? null
}

export function jumpOptions(stages: readonly Stage[], mode: string, query: string, limit = 8): StageLeaf[] {
  const q = fold(query)
  if (!q) return []
  const ranked: [number, StageLeaf][] = []
  for (const s of [...stagesInMode(stages, mode)].sort(compareStages)) {
    const leaf = { value: stageKey(s), label: s.name }
    const name = fold(s.name)
    const rank =
      name === q ? 0
      : name.startsWith(q) ? 1
      : fold(s.name.replace(/^(II|III)-/i, '')).startsWith(q) ? 2
      : matches(query, leaf) ? 3
      : -1
    if (rank >= 0) ranked.push([rank, leaf])
  }
  return ranked
    .sort((a, b) => a[0] - b[0])
    .slice(0, limit)
    .map(([, leaf]) => leaf)
}

export type StageScore = { pct: number; failing: boolean }
export type ScoreBand = 'high' | 'mid' | 'low'

export function stageScores(
  bases: readonly { key: string; score: number; failing: boolean }[],
  stages: readonly Stage[],
  table: IdealTable | null,
  d: Difficulty,
  skills: 'none' | 'max',
): Map<string, StageScore> {
  const byKey = new Map(bases.map((b) => [b.key, b]))
  const out = new Map<string, StageScore>()
  for (const s of stages) {
    const base = byKey.get(placementKey(s, d))
    const ideal = lookupIdeal(table, s, d, skills)?.score
    if (!base || !ideal) continue
    out.set(stageKey(s), base.failing ? { pct: 0, failing: true } : { pct: Math.min(100, Math.round((base.score / ideal) * 100)), failing: false })
  }
  return out
}

export const scoreBand = (pct: number): ScoreBand => (pct >= 90 ? 'high' : pct >= 80 ? 'mid' : 'low')

export function groupTint(keys: readonly string[], scores: ReadonlyMap<string, StageScore>): { band: ScoreBand | null; failing: boolean } | null {
  let weakest: number | null = null
  let failing = false
  let any = false
  for (const key of keys) {
    const s = scores.get(key)
    if (!s) continue
    any = true
    if (s.failing) failing = true
    else weakest = weakest === null ? s.pct : Math.min(weakest, s.pct)
  }
  return any ? { band: weakest === null ? null : scoreBand(weakest), failing } : null
}
