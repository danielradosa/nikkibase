import { create } from 'zustand'
import type { Outfit } from './engine'
import type { SkillChoice } from './engine'
import { parseSkills, type SkillSettings } from './skills'
import type { WardrobeSource } from './storage'
import type { OpenTarget } from './worth'
import type { Difficulty, Ideal } from './stages'

type State = {
  ready: boolean
  owned: number[]
  source: WardrobeSource | null
  decoded: { items: number; unresolved: number; known: number } | null
  stage: string | null
  outfit: Outfit | null
  ideal: Ideal | null
  busy: boolean
  importing: boolean
  engine: 'loading' | 'ready' | 'failed'
  notice: string | null
  error: string | null
  difficulty: Difficulty
  setDifficulty: (d: Difficulty) => void
  skills: SkillSettings
  setSkills: (s: SkillSettings) => void
  placements: Record<string, SkillChoice>
  setPlacement: (key: string, p: SkillChoice | null) => void
  tab: string
  mode: string
  jump: boolean
  worthY: number | null
  openStage: (target: OpenTarget, worthY: number) => void
  set: (patch: Partial<Omit<State, 'set' | 'setDifficulty' | 'setSkills' | 'setPlacement' | 'openStage'>>) => void
}

const DIFFICULTY_KEY = 'nikkibase.difficulty'
const SKILLS_KEY = 'nikkibase.skillSettings'
const LEGACY_SKILLS_KEY = 'nikkibase.skills'

function saveDifficulty(difficulty: Difficulty) {
  try {
    localStorage.setItem(DIFFICULTY_KEY, difficulty)
  } catch {
  }
}

function savedDifficulty(): Difficulty {
  try {
    const d = localStorage.getItem(DIFFICULTY_KEY)
    if (d === 'Maiden' || d === 'Princess') return d
  } catch {
  }
  return 'Maiden'
}

function savedSkills(): SkillSettings {
  try {
    return parseSkills(localStorage.getItem(SKILLS_KEY), localStorage.getItem(LEGACY_SKILLS_KEY))
  } catch {
  }
  return parseSkills(null, null)
}

export const useStore = create<State>((set) => ({
  ready: false,
  owned: [],
  source: null,
  decoded: null,
  stage: null,
  outfit: null,
  ideal: null,
  busy: false,
  importing: false,
  engine: 'loading',
  notice: null,
  error: null,
  difficulty: savedDifficulty(),
  setDifficulty: (difficulty) => {
    saveDifficulty(difficulty)
    set({ difficulty })
  },
  skills: savedSkills(),
  setSkills: (skills) => {
    try {
      localStorage.setItem(SKILLS_KEY, JSON.stringify(skills))
    } catch {
    }
    set({ skills })
  },
  placements: {},
  setPlacement: (key, p) =>
    set((state) => {
      const placements = { ...state.placements }
      if (p) placements[key] = p
      else delete placements[key]
      return { placements }
    }),
  tab: 'outfit',
  mode: 'Story',
  jump: false,
  worthY: null,
  openStage: ({ mode, stage, difficulty }, worthY) =>
    set((state) => {
      if (difficulty && difficulty !== state.difficulty) saveDifficulty(difficulty)
      return { tab: 'outfit', mode, stage, difficulty: difficulty ?? state.difficulty, jump: true, worthY }
    }),
  set: (patch) => set(patch),
}))
