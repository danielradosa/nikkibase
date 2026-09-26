import type { SkillChoice, WorthSettings } from './engine'

export type SkillLevels = { charming: number; smile: number }
export type SkillSettings = { on: boolean; manual: boolean; levels: SkillLevels }
export type SkillRequest = SkillChoice | { auto: true; levels?: SkillLevels }

export const SMILE_PERCENT = [0, 15, 17, 19, 20, 22, 24, 25, 26, 27] as const
export const CHARMING_PERCENT = [0, 24, 26, 28, 30, 32, 34, 36, 38, 40] as const
export const MAX_LEVELS: SkillLevels = { charming: 9, smile: 9 }
export const NO_SKILLS: SkillSettings = { on: false, manual: false, levels: MAX_LEVELS }

const SITE = 'nikkibase.up.railway.app'

export const isMax = (l: SkillLevels) => l.charming === 9 && l.smile === 9

export const skillsOff = (s: SkillSettings) => !s.on || (s.levels.charming === 0 && s.levels.smile === 0)

const level = (v: unknown) => (typeof v === 'number' && Number.isInteger(v) && v >= 0 && v <= 9 ? v : 9)

export function parseSkills(saved: string | null, legacy: string | null): SkillSettings {
  if (saved) {
    try {
      const v = JSON.parse(saved)
      if (v && typeof v === 'object' && !Array.isArray(v)) {
        return {
          on: v.on === true,
          manual: v.manual === true,
          levels: { charming: level(v.levels?.charming), smile: level(v.levels?.smile) },
        }
      }
    } catch {
    }
  }
  return { ...NO_SKILLS, on: legacy === 'max' }
}

export function idealSkills(s: SkillSettings): 'none' | 'max' | null {
  if (skillsOff(s)) return 'none'
  return isMax(s.levels) ? 'max' : null
}

export function skillRequest(s: SkillSettings, placement?: SkillChoice | null): SkillRequest | undefined {
  if (skillsOff(s)) return undefined
  const levels = isMax(s.levels) ? {} : { levels: { charming: s.levels.charming, smile: s.levels.smile } }
  if (s.manual && placement) return { charmSmile: placement.charmSmile, smile: placement.smile, ...levels }
  return { auto: true, ...levels }
}

export function levelsLabel(l: SkillLevels): string {
  return `Smile ${l.smile || 'off'} · Charming ${l.charming || 'off'}`
}

export function levelsShort(l: SkillLevels): string {
  if (isMax(l)) return 'max level'
  return `Smile ${l.smile || 'off'}, Charming ${l.charming || 'off'}`
}

export function placementPhrases(p: SkillChoice, l: SkillLevels, attrs: readonly string[]): string[] {
  const first = attrs[p.charmSmile]
  const second = p.smile >= 0 ? attrs[p.smile] : undefined
  if (!first) return []
  if (l.smile === 0) return [`Charming on ${first}`]
  const lead = l.charming === 0 ? `Smile on ${first}` : `Charming + Smile on ${first}`
  return second ? [lead, `Smile on ${second}`] : [lead]
}

function levelsText(l: SkillLevels): string {
  if (isMax(l)) return 'max level'
  return `${l.smile ? `Smile level ${l.smile}` : 'no Smile'}, ${l.charming ? `Charming level ${l.charming}` : 'no Charming'}`
}

export function skillsLine(p: SkillChoice | undefined, attrs: readonly string[]): string {
  if (!p) return `Scores assume no skills. ${SITE}`
  const l = p.levels ?? MAX_LEVELS
  return `Skills: ${placementPhrases(p, l, attrs).join(', ')} (${levelsText(l)}). ${SITE}`
}

export function scoredOn(p: SkillChoice, l: SkillLevels): number[] {
  if (p.charmSmile < 0) return []
  if (l.smile === 0) return [p.charmSmile]
  return p.smile >= 0 && p.smile !== p.charmSmile ? [p.charmSmile, p.smile] : [p.charmSmile]
}

export function samePlacement(a: SkillChoice, b: SkillChoice, l: SkillLevels): boolean {
  const x = scoredOn(a, l)
  const y = scoredOn(b, l)
  if (l.charming === 0) {
    x.sort((m, n) => m - n)
    y.sort((m, n) => m - n)
  }
  return x.length === y.length && x.every((code, i) => code === y[i])
}

type Scored = { score: number; skills?: SkillChoice }

export function bestElsewhere(mine: Scored, best: Scored | null, attrs: readonly string[]): string {
  if (!mine.skills || !best?.skills || mine.score >= best.score) return ''
  const l = best.skills.levels ?? MAX_LEVELS
  if (samePlacement(mine.skills, best.skills, l)) return ''
  const on = scoredOn(best.skills, l).map((code) => attrs[code])
  if (!on.length) return ''
  return ` Best possible puts ${on.length > 1 ? 'them' : 'it'} on ${on.join(' and ')}.`
}

export function searchPlan(s: SkillSettings, placement: SkillChoice | null) {
  return {
    mine: skillRequest(s, placement),
    ceiling: skillRequest({ ...s, manual: false }),
    fromTable: idealSkills(s),
  }
}

export function pickerDefault(
  weighted: readonly number[],
  l: SkillLevels,
  ...candidates: (SkillChoice | null | undefined)[]
): SkillChoice {
  const valid = (code: number) => weighted.includes(code)
  const found = candidates.find((p) => p && valid(p.charmSmile) && (p.smile < 0 || valid(p.smile)))
  const first = found?.charmSmile ?? weighted[0] ?? -1
  let second = found && found.smile >= 0 && found.smile !== first ? found.smile : -1
  if (second < 0 && l.smile > 0) second = weighted.find((code) => code !== first) ?? -1
  return { charmSmile: first, smile: second }
}

export function worthSettings(s: SkillSettings): WorthSettings {
  if (skillsOff(s)) return null
  if (isMax(s.levels)) return { auto: true }
  return { auto: true, levels: { charming: s.levels.charming, smile: s.levels.smile } }
}

export function worthSkillsText(s: SkillSettings): string {
  if (!s.on) return 'Scored without skills. Turn Skills on in Best outfit to include them.'
  if (skillsOff(s)) return 'Scored without skills: Smile and Charming are both off.'
  const taken = [s.levels.smile ? `Smile ${s.levels.smile}` : '', s.levels.charming ? `Charming ${s.levels.charming}` : '']
  return `Scored with ${taken.filter(Boolean).join(' and ')}.`
}
