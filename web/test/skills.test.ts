import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  CHARMING_PERCENT, MAX_LEVELS, NO_SKILLS, SMILE_PERCENT, bestElsewhere, idealSkills, isMax, levelsLabel, levelsShort,
  parseSkills, pickerDefault, placementPhrases, samePlacement, scoredOn, searchPlan, skillRequest, skillsLine, skillsOff,
  type SkillSettings,
} from '../src/skills.ts'

const ATTRS = ['Gorgeous', 'Simple', 'Elegant', 'Lively', 'Mature', 'Cute']
const on = (charming: number, smile: number, manual = false): SkillSettings => ({ on: true, manual, levels: { charming, smile } })

test('the level tables are the wiki values', () => {
  assert.deepEqual([...SMILE_PERCENT], [0, 15, 17, 19, 20, 22, 24, 25, 26, 27])
  assert.deepEqual([...CHARMING_PERCENT], [0, 24, 26, 28, 30, 32, 34, 36, 38, 40])
})

test('the old saved setting carries over, and the new one wins over it', () => {
  assert.deepEqual(parseSkills(null, 'max'), { on: true, manual: false, levels: MAX_LEVELS })
  assert.deepEqual(parseSkills(null, 'none'), NO_SKILLS)
  assert.deepEqual(parseSkills(null, null), NO_SKILLS)
  const saved = JSON.stringify({ on: true, manual: true, levels: { charming: 0, smile: 6 } })
  assert.deepEqual(parseSkills(saved, 'none'), on(0, 6, true))
  assert.deepEqual(parseSkills(JSON.stringify({ on: false, levels: { charming: 3, smile: 4 } }), 'max'), {
    on: false, manual: false, levels: { charming: 3, smile: 4 },
  })
})

test('anything unreadable falls back to max levels, and a broken save to the old setting', () => {
  for (const bad of [10, -1, 2.5, '3', null, true]) {
    assert.deepEqual(parseSkills(JSON.stringify({ on: true, levels: { charming: bad, smile: bad } }), null).levels, MAX_LEVELS)
  }
  assert.deepEqual(parseSkills(JSON.stringify({ on: true }), null).levels, MAX_LEVELS)
  for (const broken of ['{', '"max"', '[1]', '7', 'null']) assert.deepEqual(parseSkills(broken, 'max'), parseSkills(null, 'max'))
})

test('skills are off when switched off or when neither skill is taken', () => {
  assert.equal(skillsOff(NO_SKILLS), true)
  assert.equal(skillsOff(on(0, 0)), true)
  assert.equal(skillsOff(on(0, 1)), false)
  assert.equal(skillsOff(on(1, 0)), false)
  assert.equal(isMax(MAX_LEVELS), true)
  assert.equal(isMax({ charming: 9, smile: 8 }), false)
})

test('max-level requests keep today\'s exact shapes, and other levels say so', () => {
  assert.equal(skillRequest(NO_SKILLS), undefined)
  assert.equal(skillRequest(on(0, 0)), undefined)
  assert.deepEqual(skillRequest(on(9, 9)), { auto: true })
  assert.deepEqual(skillRequest(on(9, 9, true), { charmSmile: 3, smile: 5 }), { charmSmile: 3, smile: 5 })
  assert.deepEqual(skillRequest(on(4, 6)), { auto: true, levels: { charming: 4, smile: 6 } })
  assert.deepEqual(skillRequest(on(4, 6, true), { charmSmile: 3, smile: 5, levels: { charming: 9, smile: 9 } }), {
    charmSmile: 3, smile: 5, levels: { charming: 4, smile: 6 },
  })
  assert.deepEqual(skillRequest(on(9, 9, true), null), { auto: true })
  assert.deepEqual(skillRequest(on(9, 9, false), { charmSmile: 3, smile: 5 }), { auto: true })
})

test('only max levels and no skills are read from the precomputed table', () => {
  assert.equal(idealSkills(NO_SKILLS), 'none')
  assert.equal(idealSkills(on(0, 0)), 'none')
  assert.equal(idealSkills(on(9, 9)), 'max')
  assert.equal(idealSkills(on(9, 8)), null)
  assert.equal(idealSkills(on(0, 9)), null)
})

test('the placement reads as what each attribute gets at those levels', () => {
  const p = { charmSmile: 3, smile: 5 }
  assert.deepEqual(placementPhrases(p, MAX_LEVELS, ATTRS), ['Charming + Smile on Lively', 'Smile on Cute'])
  assert.deepEqual(placementPhrases(p, { charming: 0, smile: 4 }, ATTRS), ['Smile on Lively', 'Smile on Cute'])
  assert.deepEqual(placementPhrases({ charmSmile: 3, smile: -1 }, { charming: 5, smile: 0 }, ATTRS), ['Charming on Lively'])
  assert.deepEqual(placementPhrases({ charmSmile: 3, smile: -1 }, MAX_LEVELS, ATTRS), ['Charming + Smile on Lively'])
  assert.deepEqual(placementPhrases({ charmSmile: -1, smile: -1 }, MAX_LEVELS, ATTRS), [])
})

test('the skills note names the levels in brackets', () => {
  assert.equal(levelsShort(MAX_LEVELS), 'max level')
  assert.equal(levelsShort({ charming: 6, smile: 6 }), 'Smile 6, Charming 6')
  assert.equal(levelsShort({ charming: 0, smile: 6 }), 'Smile 6, Charming off')
  assert.equal(levelsShort({ charming: 2, smile: 0 }), 'Smile off, Charming 2')
})

test('the copied line keeps its max-level wording and names other levels', () => {
  assert.equal(skillsLine(undefined, ATTRS), 'Scores assume no skills. nikkibase.up.railway.app')
  assert.equal(
    skillsLine({ charmSmile: 3, smile: 5 }, ATTRS),
    'Skills: Charming + Smile on Lively, Smile on Cute (max level). nikkibase.up.railway.app',
  )
  assert.equal(
    skillsLine({ charmSmile: 3, smile: 5, levels: { charming: 0, smile: 6 } }, ATTRS),
    'Skills: Smile on Lively, Smile on Cute (Smile level 6, no Charming). nikkibase.up.railway.app',
  )
  assert.equal(
    skillsLine({ charmSmile: 3, smile: -1, levels: { charming: 7, smile: 0 } }, ATTRS),
    'Skills: Charming on Lively (no Smile, Charming level 7). nikkibase.up.railway.app',
  )
})

test('the levels button always names both levels', () => {
  assert.equal(levelsLabel(MAX_LEVELS), 'Smile 9 · Charming 9')
  assert.equal(levelsLabel({ charming: 0, smile: 6 }), 'Smile 6 · Charming off')
  assert.equal(levelsLabel({ charming: 4, smile: 0 }), 'Smile off · Charming 4')
})

test('only the attributes a skill actually lands on count', () => {
  assert.deepEqual(scoredOn({ charmSmile: 3, smile: 5 }, MAX_LEVELS), [3, 5])
  assert.deepEqual(scoredOn({ charmSmile: 3, smile: 5 }, { charming: 9, smile: 0 }), [3])
  assert.deepEqual(scoredOn({ charmSmile: 3, smile: -1 }, MAX_LEVELS), [3])
  assert.deepEqual(scoredOn({ charmSmile: 3, smile: 3 }, MAX_LEVELS), [3])
  assert.deepEqual(scoredOn({ charmSmile: -1, smile: -1 }, MAX_LEVELS), [])
})

test('placements match on what they score: order matters only when Charming is taken', () => {
  assert.equal(samePlacement({ charmSmile: 1, smile: 2 }, { charmSmile: 1, smile: 2 }, MAX_LEVELS), true)
  assert.equal(samePlacement({ charmSmile: 1, smile: 2 }, { charmSmile: 2, smile: 1 }, MAX_LEVELS), false)
  assert.equal(samePlacement({ charmSmile: 1, smile: 2 }, { charmSmile: 2, smile: 1 }, { charming: 0, smile: 9 }), true)
  assert.equal(samePlacement({ charmSmile: 1, smile: -1 }, { charmSmile: 1, smile: 4 }, { charming: 7, smile: 0 }), true)
  assert.equal(samePlacement({ charmSmile: 1, smile: -1 }, { charmSmile: 2, smile: 4 }, { charming: 7, smile: 0 }), false)
})

test('the best possible placement is named only when it scores more and lands elsewhere', () => {
  const at = (charmSmile: number, smile: number, levels?: { charming: number; smile: number }) => ({ charmSmile, smile, levels })
  assert.equal(bestElsewhere({ score: 10, skills: at(3, 5) }, { score: 20, skills: at(3, 4) }, ATTRS), ' Best possible puts them on Lively and Mature.')
  assert.equal(bestElsewhere({ score: 20, skills: at(3, 5) }, { score: 20, skills: at(3, 4) }, ATTRS), '')
  assert.equal(bestElsewhere({ score: 30, skills: at(3, 5) }, { score: 20, skills: at(3, 4) }, ATTRS), '')
  assert.equal(bestElsewhere({ score: 10, skills: at(3, 5) }, { score: 20, skills: at(3, 5) }, ATTRS), '')
  assert.equal(bestElsewhere({ score: 10, skills: at(3, -1, { charming: 7, smile: 0 }) }, { score: 20, skills: at(3, -1, { charming: 7, smile: 0 }) }, ATTRS), '')
  assert.equal(bestElsewhere({ score: 10, skills: at(2, -1, { charming: 7, smile: 0 }) }, { score: 20, skills: at(3, -1, { charming: 7, smile: 0 }) }, ATTRS), ' Best possible puts it on Lively.')
  assert.equal(bestElsewhere({ score: 10, skills: at(5, 3, { charming: 0, smile: 6 }) }, { score: 20, skills: at(3, 5, { charming: 0, smile: 6 }) }, ATTRS), '')
  assert.equal(bestElsewhere({ score: 10 }, { score: 20, skills: at(3, 4) }, ATTRS), '')
  assert.equal(bestElsewhere({ score: 10, skills: at(3, 4) }, null, ATTRS), '')
})

test('each search gets the right request: yours may be placed by hand, the ceiling never is', () => {
  assert.deepEqual(searchPlan(NO_SKILLS, { charmSmile: 1, smile: 2 }), { mine: undefined, ceiling: undefined, fromTable: 'none' })
  assert.deepEqual(searchPlan(on(9, 9), null), { mine: { auto: true }, ceiling: { auto: true }, fromTable: 'max' })
  assert.deepEqual(searchPlan(on(9, 9, true), { charmSmile: 1, smile: 2 }), {
    mine: { charmSmile: 1, smile: 2 }, ceiling: { auto: true }, fromTable: 'max',
  })
  assert.deepEqual(searchPlan(on(3, 5, true), { charmSmile: 1, smile: 2 }), {
    mine: { charmSmile: 1, smile: 2, levels: { charming: 3, smile: 5 } },
    ceiling: { auto: true, levels: { charming: 3, smile: 5 } },
    fromTable: null,
  })
  assert.deepEqual(searchPlan(on(0, 0, true), { charmSmile: 1, smile: 2 }), { mine: undefined, ceiling: undefined, fromTable: 'none' })
})

test('the placement pickers start from your pick, then the engine\'s, and only ever show this stage\'s attributes', () => {
  const weighted = [1, 3, 5, 7]
  assert.deepEqual(pickerDefault(weighted, MAX_LEVELS, { charmSmile: 5, smile: 1 }, { charmSmile: 3, smile: 7 }), { charmSmile: 5, smile: 1 })
  assert.deepEqual(pickerDefault(weighted, MAX_LEVELS, undefined, { charmSmile: 3, smile: 7 }), { charmSmile: 3, smile: 7 })
  assert.deepEqual(pickerDefault(weighted, MAX_LEVELS, { charmSmile: 2, smile: 1 }, { charmSmile: 8, smile: 3 }), { charmSmile: 1, smile: 3 })
  assert.deepEqual(pickerDefault(weighted, MAX_LEVELS, undefined, { charmSmile: 3, smile: -1 }), { charmSmile: 3, smile: 1 })
  assert.deepEqual(pickerDefault(weighted, { charming: 9, smile: 0 }, undefined, { charmSmile: 3, smile: -1 }), { charmSmile: 3, smile: -1 })
  assert.deepEqual(pickerDefault(weighted, MAX_LEVELS, { charmSmile: 5, smile: 5 }), { charmSmile: 5, smile: 1 })
  assert.deepEqual(pickerDefault([4], MAX_LEVELS), { charmSmile: 4, smile: -1 })
  assert.deepEqual(pickerDefault([], MAX_LEVELS), { charmSmile: -1, smile: -1 })
})

