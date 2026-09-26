import type { SkillChoice, WorthVersion } from './engine'

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
  mode: 'Story' | 'Commission' | 'Co-op' | 'Arena'
  variants?: { maiden?: Scoring }
}

export const stageKey = (s: Stage) => `${s.mode}/${s.name}`

const MODE_ORDER = ['Story', 'Commission', 'Co-op', 'Arena', 'Dreamweaver']

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
  parts.push(...[...modes].filter((m) => !named.includes(m)).sort())

  return parts.length ? `Has ${list(parts)}. Newer stages aren't in yet.` : ''
}

const EXAMPLES: Record<string, string> = { Story: '5-11', Commission: '3-7' }

export function stagePlaceholder(mode: string): string {
  const example = EXAMPLES[mode]
  return `Pick ${/^[AEIOU]/.test(mode) ? 'an' : 'a'} ${mode} stage${example ? `, e.g. ${example}` : ''}`
}
