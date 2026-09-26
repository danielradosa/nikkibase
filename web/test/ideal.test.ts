import { test } from 'node:test'
import assert from 'node:assert/strict'
import { lookupIdeal, type IdealTable, type Stage } from '../src/stages.ts'

const W = [1, 2, 3, 4, 5]
const A = [0, 1, 2, 3, 4]

function stage(mode: Stage['mode'], name: string, extra: Partial<Stage> = {}): Stage {
  return { name, mode, weights: W, attrs: A, ...extra }
}

const maidenNumbers = { weights: [5, 4, 3, 2, 1], attrs: A }
const withVariant = stage('Story', '3-10', { variants: { maiden: maidenNumbers } })
const princessOnly = stage('Story', '6-9')

const table: IdealTable = {
  'Story/3-10': {
    score: 237681,
    items: [
      [10001, 0],
      [20001, 1],
    ],
    auto: { charmSmile: 3, smile: 5, score: 301442, items: [[10003, 0]] },
    variants: {
      maiden: { score: 198311, items: [[10002, 0]], auto: { charmSmile: 8, smile: 1, score: 250067, items: [[10004, 0]] } },
    },
  },
  'Story/6-9': { score: 219849, items: [[30001, 2]], auto: { charmSmile: 7, smile: 2, score: 270113, items: [[30002, 2]] } },
  'Commission/6-9': { score: 7000, items: [[40001, 3]] },
}

test('the shipped outfit comes back in the shape the comparison table reads', () => {
  assert.deepEqual(lookupIdeal(table, withVariant, 'Princess'), {
    score: 237681,
    items: [
      { id: 10001, pos: 0 },
      { id: 20001, pos: 1 },
    ],
  })
})

test('Maiden reads the Maiden variant only where the stage has one', () => {
  assert.equal(lookupIdeal(table, withVariant, 'Maiden')?.score, 198311)
  assert.equal(lookupIdeal(table, princessOnly, 'Maiden')?.score, 219849)
  assert.equal(lookupIdeal(table, princessOnly, 'Princess')?.score, 219849)
})

test('a stage is found by mode and name, so Story 6-9 and Commission 6-9 stay apart', () => {
  assert.equal(lookupIdeal(table, stage('Commission', '6-9'), 'Maiden')?.score, 7000)
})

test('difficulty is ignored outside Story, as in resolveStage', () => {
  const arena = stage('Arena', 'Beach Party', { variants: { maiden: maidenNumbers } })
  const withArena: IdealTable = {
    'Arena/Beach Party': { score: 68325, items: [], variants: { maiden: { score: 1, items: [] } } },
  }
  assert.equal(lookupIdeal(withArena, arena, 'Maiden')?.score, 68325)
})

test('with skills the auto outfit comes back with the placement it was scored with', () => {
  assert.deepEqual(lookupIdeal(table, withVariant, 'Princess', 'max'), {
    score: 301442,
    items: [{ id: 10003, pos: 0 }],
    skills: { charmSmile: 3, smile: 5 },
  })
  assert.equal(lookupIdeal(table, withVariant, 'Princess', 'none')?.skills, undefined)
  assert.equal(lookupIdeal(table, withVariant, 'Princess')?.score, 237681)
})

test('with skills Maiden reads the Maiden variant auto outfit', () => {
  assert.deepEqual(lookupIdeal(table, withVariant, 'Maiden', 'max'), {
    score: 250067,
    items: [{ id: 10004, pos: 0 }],
    skills: { charmSmile: 8, smile: 1 },
  })
  assert.equal(lookupIdeal(table, princessOnly, 'Maiden', 'max')?.score, 270113)
})

test('with skills a table without the auto outfit returns null, so the engine computes it', () => {
  assert.equal(lookupIdeal(table, stage('Commission', '6-9'), 'Maiden', 'max'), null)
  assert.equal(lookupIdeal(table, stage('Commission', '6-9'), 'Maiden', 'none')?.score, 7000)
  assert.equal(lookupIdeal(null, withVariant, 'Princess', 'max'), null)
  const noVariantAuto: IdealTable = {
    'Story/3-10': {
      score: 237681,
      items: [],
      auto: { charmSmile: 3, smile: 5, score: 301442, items: [] },
      variants: { maiden: { score: 198311, items: [] } },
    },
  }
  assert.equal(lookupIdeal(noVariantAuto, withVariant, 'Maiden', 'max'), null)
  assert.equal(lookupIdeal(noVariantAuto, withVariant, 'Princess', 'max')?.score, 301442)
})

test('a stage the table does not cover returns null, so the engine computes it', () => {
  assert.equal(lookupIdeal(null, withVariant, 'Princess'), null)
  assert.equal(lookupIdeal(table, stage('Story', '1-1'), 'Princess'), null)
  const noVariant: IdealTable = { 'Story/3-10': { score: 237681, items: [] } }
  assert.equal(lookupIdeal(noVariant, withVariant, 'Maiden'), null)
})
