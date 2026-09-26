import { relay } from './recycle'
import type { ItemPlaces } from './items'
import type { SkillLevels, SkillRequest } from './skills'

export type Alternative = { id: number; delta: number }
export type OutfitItem = { id: number; slot: number; pos: number; alts: Alternative[]; moreAlts: boolean }
export type SkillChoice = { charmSmile: number; smile: number; levels?: SkillLevels }
export type Outfit = {
  score: number
  items: OutfitItem[]
  dress: number
  separates: number
  ownedPlaces: number[]
  skills?: SkillChoice
  missing?: number[][]
}
export type StageTags = Record<string, number>
export type Decoded = { items: number; unresolved: number; known: number; ids: number[] }

export type WorthVersion = {
  key: string
  mode: string
  weights: number[]
  attrs: number[]
  tags?: StageTags
  require?: number[][]
  ideal: number
}
export type WorthSettings = { auto: true; levels?: SkillLevels } | null
export type WorthSuit = { key: string; items: number[] }
export type WorthFilter = { modes?: string[]; slots?: number[]; places?: number[]; skip?: string[]; suits?: boolean }
export type WorthSession = { session: number; total: number }
export type WorthProgress = { done: number; total: number }
export type WorthExample = { key: string; points: number; pct: number }
export type WorthRow = {
  suit?: string
  items: number[]
  places: number[]
  worth: number
  stages: number
  best: WorthExample
  examples: WorthExample[]
}
export type WorthNeeded = { key: string; missing: number[][] }
export type WorthRanking = { rows: WorthRow[]; needed: WorthNeeded[] }

const send = relay(() => new Worker(new URL('./worker.ts', import.meta.url), { type: 'module' }))

export const engine = {
  init: (version: string) => send<null>('init', { version }),
  loadKeystream: () => send<null>('keystream'),
  decode: (text: string) => send<Decoded>('decode', { text }),
  selections: (text: string) => send<Decoded>('selections', { text }),
  setWardrobe: (ids: number[]) => send<Omit<Decoded, 'ids'>>('setWardrobe', { ids }),
  places: () => send<ItemPlaces>('places'),
  best: (
    weights: number[],
    attrs: number[],
    tags?: StageTags,
    scope: 'wardrobe' | 'all' = 'wardrobe',
    skills?: SkillRequest,
    require?: number[][],
  ) => send<Outfit>('best', { weights, attrs, skills, tags, scope, require }),
  worth: {
    start: (versions: WorthVersion[], settings: WorthSettings = null, suits: WorthSuit[] = []) =>
      send<WorthSession>('worthStart', { versions, settings, suits }),
    run: (session: number, count: number) => send<WorthProgress>('worthRun', { session, count }),
    rank: (session: number, filter: WorthFilter, limit: number) =>
      send<WorthRanking>('worthRank', { session, filter, limit }),
  },
}
