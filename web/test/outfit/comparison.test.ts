import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  FINDING,
  NAMES_WAIT,
  alternativesLabel,
  bestNote,
  bookReads,
  bookScore,
  closeCall,
  closeCallText,
  comparisonRows,
  expandLabel,
  outfitText,
  resultView,
  rowOpens,
  unwornLabel,
} from '../../src/outfit/comparison.ts'
import { SITE_HOST, skillsLine } from '../../src/outfit/skills.ts'
import type { Outfit } from '../../src/engine/engine.ts'

const TOP = 3
const ACCESSORY = 9
const HELD_LEFT = 20

test('an empty place the player owns something for was left empty by choice', () => {
  const owned = new Set([TOP, ACCESSORY, HELD_LEFT])
  for (const pos of [TOP, ACCESSORY, HELD_LEFT]) assert.equal(unwornLabel(pos, owned), 'not worn')
})

test('an empty place the player owns nothing for says so', () => {
  assert.equal(unwornLabel(ACCESSORY, new Set([TOP])), 'nothing owned')
  assert.equal(unwornLabel(ACCESSORY, new Set()), 'nothing owned')
})

test('with nothing worn there is nothing to swap, whatever else is owned', () => {
  assert.equal(alternativesLabel(false, 0), '—')
  assert.equal(alternativesLabel(false, 3), '—')
})

test('a worn item counts the owned items that could replace it', () => {
  assert.equal(alternativesLabel(true, 0), 'none owned')
  assert.equal(alternativesLabel(true, 1), '1 other')
  assert.equal(alternativesLabel(true, 5), '5 others')
})

test('a list cut short at five says there are more, not that there are five', () => {
  assert.equal(alternativesLabel(true, 5, true), '5+ others')
  assert.equal(alternativesLabel(false, 5, true), '—')
})

const SLOTS = ['hair', 'dress', 'coat', 'top']
const ATTRS = ['Gorgeous', 'Simple', 'Elegant', 'Lively', 'Mature', 'Cute']
const PLACES = [{ name: 'Hair', slot: 0 }, { name: 'Dress', slot: 1 }, { name: 'Coat', slot: 2 }, { name: 'Top', slot: 3 }]
const NO_SUITS = new Map<number, string>()
const NAMES = new Map([[10, 'Rose Bun'], [20, 'Silk Gown'], [30, 'Wool Coat'], [40, 'Lace Top'], [50, 'Star Crown']])

function outfit(score: number, items: [id: number, pos: number][], extra: Partial<Outfit> = {}): Outfit {
  return {
    score,
    items: items.map(([id, pos]) => ({ id, pos, slot: pos, alts: [], moreAlts: false })),
    dress: 0,
    separates: 0,
    ownedPlaces: items.map(([, pos]) => pos),
    ...extra,
  }
}

test('a dress and separates within 5% of each other are a close call, and the gap is returned', () => {
  assert.equal(closeCall({ dress: 1000, separates: 960 }), 0.04)
  assert.equal(closeCall({ dress: 1000, separates: 950 }), null)
})

test('a close call says how close, never below 1%, and what to do', () => {
  assert.equal(
    closeCallText(0.04),
    'Close call: your best dress and best top + bottom are within 4%. Either could win in game, so try both.',
  )
  assert.equal(
    closeCallText(0.001),
    'Close call: your best dress and best top + bottom are within 1%. Either could win in game, so try both.',
  )
})

test('there is no close call without both a dress and separates', () => {
  assert.equal(closeCall(null), null)
  assert.equal(closeCall({ dress: 0, separates: 900 }), null)
  assert.equal(closeCall({ dress: 900, separates: 0 }), null)
})

test('every place either outfit fills gets one row, in place order', () => {
  const mine = outfit(100, [[40, 3], [10, 0]])
  const best = outfit(200, [[50, 0], [20, 1]])
  const rows = comparisonRows(mine, best, NAMES, PLACES, SLOTS)
  assert.deepEqual(rows.map((r) => r.slot), ['Hair', 'Dress', 'Top'])
  assert.deepEqual(rows.map((r) => [r.mine, r.best]), [['Rose Bun', 'Star Crown'], [null, 'Silk Gown'], ['Lace Top', null]])
})

test('a row says when the player wears the best possible item there', () => {
  const rows = comparisonRows(outfit(1, [[10, 0]]), outfit(1, [[10, 0]]), NAMES, PLACES, SLOTS)
  assert.equal(rows[0].same, true)
})

test('an empty row says why it is empty', () => {
  const mine = outfit(1, [[10, 0]], { ownedPlaces: [0, 1] })
  const best = outfit(1, [[20, 1], [40, 3]])
  const rows = comparisonRows(mine, best, NAMES, PLACES, SLOTS)
  assert.deepEqual(rows.filter((r) => r.mine === null).map((r) => r.unworn), ['not worn', 'nothing owned'])
})

test('unknown places and unnamed items fall back to their numbers', () => {
  const rows = comparisonRows(outfit(1, [[99, 7]]), null, NAMES, PLACES, SLOTS)
  assert.equal(rows[0].slot, '#7')
  assert.equal(rows[0].mine, '#99')
})

test('the copied outfit lists the stage, score and each item by place', () => {
  const text = outfitText(outfit(1234, [[40, 3], [10, 0]]), outfit(5678, []), { mode: 'Commission', name: '1-1' }, 'Maiden', NAMES, NO_SUITS, PLACES, SLOTS, skillsLine(undefined, ATTRS))
  assert.deepEqual(text.split('\n'), [
    'Commission 1-1 — 1,234 (2 items)',
    'Hair: Rose Bun',
    'Top: Lace Top',
    'best possible — 5,678',
    `Scores assume no skills. ${SITE_HOST}`,
  ])
})

test('the copied outfit names the suit of each piece that has one', () => {
  const suits = new Map([[10, 'Metallic Crisis']])
  const text = outfitText(outfit(1234, [[40, 3], [10, 0]]), outfit(5678, []), { mode: 'Commission', name: '1-1' }, 'Maiden', NAMES, suits, PLACES, SLOTS, skillsLine(undefined, ATTRS))
  assert.deepEqual(text.split('\n'), [
    'Commission 1-1 — 1,234 (2 items)',
    'Hair: Rose Bun (Metallic Crisis)',
    'Top: Lace Top',
    'best possible — 5,678',
    `Scores assume no skills. ${SITE_HOST}`,
  ])
})

test('the copied outfit does not repeat a name when the suit is named after the piece', () => {
  const suits = new Map([[10, 'Rose Bun'], [20, 'Silk gown'], [40, 'Metallic Crisis']])
  const text = outfitText(outfit(1234, [[10, 0], [20, 1], [40, 3]]), null, { mode: 'Commission', name: '1-1' }, 'Maiden', NAMES, suits, PLACES, SLOTS, skillsLine(undefined, ATTRS))
  assert.deepEqual(text.split('\n').slice(1, 4), ['Hair: Rose Bun', 'Dress: Silk Gown', 'Top: Lace Top (Metallic Crisis)'])
})

test('the copied outfit names the skills it was scored with', () => {
  const skilled = outfit(1234, [[10, 0]], { skills: { charmSmile: 3, smile: 5 } })
  const text = outfitText(skilled, outfit(5678, []), { mode: 'Commission', name: '1-1' }, 'Maiden', NAMES, NO_SUITS, PLACES, SLOTS, skillsLine(skilled.skills, ATTRS))
  assert.deepEqual(text.split('\n').slice(-2), [
    'best possible — 5,678',
    `Skills: Charming + Smile on Lively, Smile on Cute (max level). ${SITE_HOST}`,
  ])
  const plain = outfitText(outfit(1234, [[10, 0]]), null, { mode: 'Commission', name: '1-1' }, 'Maiden', NAMES, NO_SUITS, PLACES, SLOTS, skillsLine(undefined, ATTRS))
  assert.equal(plain.split('\n').at(-1), `Scores assume no skills. ${SITE_HOST}`)
})

test('each Cloud Adventure read adds 1% to the whole score, rounded down', () => {
  assert.equal(bookScore(77438, 5), 81309)
  assert.equal(bookScore(50541, 5), 53068)
  assert.equal(bookScore(1000, 3), 1030)
  assert.equal(bookScore(77438, 0), 77438)
  assert.equal(bookScore(0, 5), 0)
})

test('Cloud Adventure counts on Commission stages only', () => {
  assert.equal(bookReads('Commission', 4), 4)
  for (const mode of ['Story', 'Arena', 'Co-op', undefined]) assert.equal(bookReads(mode, 4), 0)
})

test('with Cloud Adventure an alternative loses what your score would lose, and a tie stays a tie', () => {
  const mine = outfit(1000, [[10, 0]])
  mine.items[0].alts = [{ id: 20, delta: 0 }, { id: 30, delta: -10 }, { id: 40, delta: -1 }]
  assert.deepEqual(comparisonRows(mine, null, NAMES, PLACES, SLOTS, 5)[0].alts, [
    { id: 20, delta: 0 },
    { id: 30, delta: -11 },
    { id: 40, delta: -2 },
  ])
  assert.deepEqual(comparisonRows(mine, null, NAMES, PLACES, SLOTS)[0].alts, mine.items[0].alts)
})

test('the copied outfit counts Cloud Adventure in both scores and names it', () => {
  const reads = bookReads('Commission', 5)
  const text = outfitText(outfit(77438, [[40, 3], [10, 0]]), outfit(80000, []), { mode: 'Commission', name: '2-3' }, 'Maiden', NAMES, NO_SUITS, PLACES, SLOTS, skillsLine(undefined, ATTRS, reads), reads)
  assert.deepEqual(text.split('\n'), [
    'Commission 2-3 — 81,309 (2 items)',
    'Hair: Rose Bun',
    'Top: Lace Top',
    'best possible — 84,000',
    `Scores assume no skills. Cloud Adventure +5%. ${SITE_HOST}`,
  ])
  const story = bookReads('Story', 5)
  const plain = outfitText(outfit(77438, []), outfit(80000, []), { mode: 'Story', name: '2-3' }, 'Maiden', NAMES, NO_SUITS, PLACES, SLOTS, skillsLine(undefined, ATTRS, story), story)
  assert.deepEqual(plain.split('\n'), ['Story 2-3 (Maiden) — 77,438 (0 items)', 'best possible — 80,000', `Scores assume no skills. ${SITE_HOST}`])
})

test('the copied outfit names the difficulty on Story stages only', () => {
  const story = outfitText(outfit(1, []), null, { mode: 'Story', name: '6-9' }, 'Maiden', NAMES, NO_SUITS, PLACES, SLOTS, skillsLine(undefined, ATTRS))
  assert.equal(story.split('\n')[0], 'Story 6-9 (Maiden) — 1 (0 items)')
})

test('there is nothing to copy without an outfit and a stage', () => {
  assert.equal(outfitText(null, null, { mode: 'Story', name: '1-1' }, 'Maiden', NAMES, NO_SUITS, PLACES, SLOTS, skillsLine(undefined, ATTRS)), '')
  assert.equal(outfitText(outfit(1, []), null, null, 'Maiden', NAMES, NO_SUITS, PLACES, SLOTS, skillsLine(undefined, ATTRS)), '')
})

test('on phones each row names the best possible item under yours, or says yours is it', () => {
  assert.equal(bestNote({ best: 'Star Crown', same: false }), 'Best: Star Crown')
  assert.equal(bestNote({ best: 'Rose Bun', same: true }), 'best possible')
  assert.equal(bestNote({ best: null, same: false }), 'Best: —')
})

test('a row opens only when it has alternatives', () => {
  const alts = [{ id: 11, delta: -40 }]
  assert.equal(rowOpens({ alts: [] }), false)
  assert.equal(rowOpens({ alts }), true)
})

test('the button that opens a row says which slot it opens', () => {
  assert.equal(expandLabel('Hair'), 'Show alternatives for Hair')
})

test('a first search shows a placeholder, a later one dims the outfit it replaces', () => {
  const view = (owned: number, chosen: boolean, outfit: boolean, busy: boolean) => resultView({ owned, chosen, outfit, busy })
  assert.equal(view(10, true, false, true), 'first')
  assert.equal(view(10, true, true, true), 'stale')
  assert.equal(view(10, true, true, false), 'ready')
  assert.equal(view(10, true, false, false), 'none')
  assert.equal(view(0, true, false, true), 'none')
  assert.equal(view(10, false, false, true), 'none')
  assert.equal(FINDING, 'Finding your best outfit…')
  assert.equal(NAMES_WAIT, 'Loading item names…')
})
