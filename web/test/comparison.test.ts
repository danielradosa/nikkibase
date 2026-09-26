import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  alternativesLabel,
  bestNote,
  closeCall,
  closeCallText,
  comparisonRows,
  copyLines,
  expandLabel,
  outfitText,
  rowOpens,
  unwornLabel,
} from '../src/comparison.ts'
import { skillsLine } from '../src/skills.ts'
import type { Outfit } from '../src/engine.ts'

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
  const text = outfitText(outfit(1234, [[40, 3], [10, 0]]), outfit(5678, []), { mode: 'Commission', name: '1-1' }, 'Maiden', NAMES, PLACES, SLOTS, skillsLine(undefined, ATTRS))
  assert.deepEqual(text.split('\n'), [
    'Commission 1-1 — 1,234 (2 items)',
    'Hair: Rose Bun',
    'Top: Lace Top',
    'best possible — 5,678',
    'Scores assume no skills. nikkibase.up.railway.app',
  ])
})

test('the copied outfit names the skills it was scored with', () => {
  const skilled = outfit(1234, [[10, 0]], { skills: { charmSmile: 3, smile: 5 } })
  const text = outfitText(skilled, outfit(5678, []), { mode: 'Commission', name: '1-1' }, 'Maiden', NAMES, PLACES, SLOTS, skillsLine(skilled.skills, ATTRS))
  assert.deepEqual(text.split('\n').slice(-2), [
    'best possible — 5,678',
    'Skills: Charming + Smile on Lively, Smile on Cute (max level). nikkibase.up.railway.app',
  ])
  const plain = outfitText(outfit(1234, [[10, 0]]), null, { mode: 'Commission', name: '1-1' }, 'Maiden', NAMES, PLACES, SLOTS, skillsLine(undefined, ATTRS))
  assert.equal(plain.split('\n').at(-1), 'Scores assume no skills. nikkibase.up.railway.app')
})

test('the copied outfit names the difficulty on Story stages only', () => {
  const story = outfitText(outfit(1, []), null, { mode: 'Story', name: '6-9' }, 'Maiden', NAMES, PLACES, SLOTS, skillsLine(undefined, ATTRS))
  assert.equal(story.split('\n')[0], 'Story 6-9 (Maiden) — 1 (0 items)')
})

test('there is nothing to copy without an outfit and a stage', () => {
  assert.equal(outfitText(null, null, { mode: 'Story', name: '1-1' }, 'Maiden', NAMES, PLACES, SLOTS, skillsLine(undefined, ATTRS)), '')
  assert.equal(outfitText(outfit(1, []), null, null, 'Maiden', NAMES, PLACES, SLOTS, skillsLine(undefined, ATTRS)), '')
})

test('on phones each row names the best possible item under yours, or says yours is it', () => {
  assert.equal(bestNote({ best: 'Star Crown', same: false }), 'Best: Star Crown')
  assert.equal(bestNote({ best: 'Rose Bun', same: true }), 'best possible')
  assert.equal(bestNote({ best: null, same: false }), 'Best: —')
})

test('a row opens when it has alternatives, and on phones also when it differs from the best possible', () => {
  const alts = [{ id: 11, delta: -40 }]
  assert.equal(rowOpens({ same: true, alts: [] }, false), false)
  assert.equal(rowOpens({ same: false, alts: [] }, false), false)
  assert.equal(rowOpens({ same: true, alts }, false), true)
  assert.equal(rowOpens({ same: true, alts: [] }, true), false)
  assert.equal(rowOpens({ same: false, alts: [] }, true), true)
  assert.equal(rowOpens({ same: true, alts }, true), true)
})

test('an opened row on a phone offers each name in it to copy', () => {
  assert.deepEqual(copyLines({ mine: 'Rose Bun', best: 'Star Crown', same: false }), [
    { label: 'Your best', name: 'Rose Bun' },
    { label: 'Best possible', name: 'Star Crown' },
  ])
  assert.deepEqual(copyLines({ mine: 'Rose Bun', best: 'Rose Bun', same: true }), [{ label: 'Your best', name: 'Rose Bun' }])
  assert.deepEqual(copyLines({ mine: null, best: 'Silk Gown', same: false }), [{ label: 'Best possible', name: 'Silk Gown' }])
  assert.deepEqual(copyLines({ mine: 'Lace Top', best: null, same: false }), [{ label: 'Your best', name: 'Lace Top' }])
})

test('the button that opens a row says which slot it opens', () => {
  assert.equal(expandLabel('Hair'), 'Show alternatives for Hair')
})
