import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  ANY_SLOT, ITEM_STEP, choiceLabel, gradesLine, inChoice, itemPage, itemsTabLabel, tickHint, moreItemsText, ownLabel, parseItems, placeName, slotChoice, slotOptions,
  type Row,
} from '../src/items.ts'

const places = [
  { name: 'Hair', slot: 0 },
  { name: 'Dress', slot: 1 },
  { name: 'Coat', slot: 2 },
  { name: 'Top', slot: 3 },
  { name: 'Bottom', slot: 4 },
  { name: 'Leglets', slot: 5 },
  { name: 'Hosiery', slot: 5 },
  { name: 'Shoes', slot: 6 },
  { name: 'Makeup', slot: 7 },
  { name: 'Earrings', slot: 8 },
  { name: 'Held, right', slot: 8 },
  { name: 'Spirit', slot: 9 },
]

const row = (id: number, name: string, slot: number): Row => [id, name, slot, 0, 2, 4, 6, 8, 'S', 'A', 'B', 'C', 'SS', 5, ''] as unknown as Row

test('each item takes its place from the engine, and has none when the engine gave none', () => {
  const [earrings, leglets, lost] = parseItems(
    [row(170004, 'Moon Drop', 8), row(60001, 'Lace Leglet', 5), row(99, 'Unknown', 8)],
    { ids: [60001, 170004], places: [5, 9] },
  )
  assert.equal(earrings.place, 9)
  assert.equal(leglets.place, 5)
  assert.equal(lost.place, -1)
  assert.equal(parseItems([row(1, 'Plain', 0)])[0].place, -1)
})

test('an item is named by its place, or by its slot when its place is unknown', () => {
  assert.equal(placeName({ slot: 8, place: 10 }, places), 'Held, right')
  assert.equal(placeName({ slot: 5, place: 5 }, places), 'Leglets')
  assert.equal(placeName({ slot: 8, place: -1 }, places), 'Accessory')
  assert.equal(placeName({ slot: 8, place: 40 }, places), 'Accessory')
})

test('slots with one place stay one choice, and slots with several list each place', () => {
  const options = slotOptions(places)
  assert.deepEqual(options[0], { value: ANY_SLOT, label: 'any slot' })
  assert.deepEqual(options[1], { value: 's0', label: 'hair' })
  assert.deepEqual(options.find((o) => o.label === 'hosiery'), {
    label: 'hosiery',
    options: [
      { value: 's5', label: 'any hosiery' },
      { value: 'p5', label: 'leglets' },
      { value: 'p6', label: 'hosiery' },
    ],
  })
  assert.deepEqual(options.find((o) => o.label === 'accessory'), {
    label: 'accessory',
    options: [
      { value: 's8', label: 'any accessory' },
      { value: 'p9', label: 'earrings' },
      { value: 'p10', label: 'held, right' },
    ],
  })
  assert.deepEqual(options.at(-1), { value: 's9', label: 'spirit' })
  assert.equal(options.length, 11)
})

test('a choice filters by slot or by place', () => {
  assert.deepEqual(slotChoice(ANY_SLOT), {})
  assert.deepEqual(slotChoice('s8'), { slots: [8] })
  assert.deepEqual(slotChoice('p10'), { places: [10] })
  assert.deepEqual(slotChoice('p'), {})
  assert.deepEqual(slotChoice('x3'), {})
  const held = { slot: 8, place: 10 }
  assert.ok(inChoice(ANY_SLOT, held))
  assert.ok(inChoice('s8', held))
  assert.ok(inChoice('p10', held))
  assert.ok(!inChoice('p9', held))
  assert.ok(!inChoice('s5', held))
})

test('the Items tab drops its count on phones so the three tabs fit at 320 px', () => {
  assert.equal(itemsTabLabel(34012, false), 'Items (34,012)')
  assert.equal(itemsTabLabel(34012, true), 'Items')
  assert.equal(itemsTabLabel(null, false), 'Items')
  assert.equal(itemsTabLabel(null, true), 'Items')
})

test('on phones the grades read as one line under the name, which wraps only after a dot', () => {
  const line = gradesLine({ attrs: [1, 3, 5, 7, 8], grades: ['S', 'A', 'A', 'A', 'A'] })
  assert.equal(line.replaceAll('\u00a0', ' '), 'Simple S · Lively A · Cute A · Pure A · Warm A')
  assert.deepEqual(line.split(' '), ['Simple\u00a0S\u00a0·', 'Lively\u00a0A\u00a0·', 'Cute\u00a0A\u00a0·', 'Pure\u00a0A\u00a0·', 'Warm\u00a0A'])
  const plain = (it: { attrs: number[]; grades: string[] }) => gradesLine(it).replaceAll('\u00a0', ' ')
  assert.equal(plain(parseItems([row(1, 'Plain', 0)])[0]), 'Gorgeous S · Elegant A · Mature B · Sexy C · Warm SS')
  assert.equal(plain({ attrs: [1, 3, 5, 7, 8], grades: ['S', '', 'A', '', ''] }), 'Simple S · Cute A')
  assert.equal(gradesLine({ attrs: [0, 1, 2, 3, 4], grades: ['', '', '', '', ''] }), '')
})

test('the Own box names its item', () => {
  assert.equal(ownLabel('Moon Drop'), 'Own Moon Drop')
})

test('on phones the list grows 50 items at a time', () => {
  assert.equal(ITEM_STEP, 50)
  assert.equal(moreItemsText(34012, 50), 'Show 50 more')
  assert.equal(moreItemsText(62, 50), 'Show 12 more')
  assert.equal(moreItemsText(50, 50), null)
  assert.equal(moreItemsText(30, 50), null)
})

test('the Items list goes back to the first 50 whenever a filter changes, also back to an earlier one', () => {
  const m = itemPage({ filters: 'm', shown: ITEM_STEP }, 'm')
  assert.deepEqual(m, { filters: 'm', shown: 50 })
  const more = { filters: 'm', shown: m.shown + ITEM_STEP }
  assert.equal(itemPage(more, 'm'), more)
  const mo = itemPage(more, 'mo')
  assert.deepEqual(mo, { filters: 'mo', shown: 50 })
  assert.deepEqual(itemPage(mo, 'm'), { filters: 'm', shown: 50 })
})

test('the tick hint appears or goes only while the Items tab is not shown, so a first tick never moves the rows', () => {
  assert.equal(tickHint(false, true, true), false)
  assert.equal(tickHint(true, false, true), true)
  assert.equal(tickHint(false, true, false), true)
  assert.equal(tickHint(true, false, false), false)
})

test('a wait names the chosen slot or place with a capital, and says nothing for any slot', () => {
  assert.equal(choiceLabel('s0', places), 'Hair')
  assert.equal(choiceLabel('s8', places), 'Accessory')
  assert.equal(choiceLabel('p10', places), 'Held, right')
  assert.equal(choiceLabel('p0', [{ name: 'leglets', slot: 5 }]), 'Leglets')
  assert.equal(choiceLabel('p40', places), null)
  assert.equal(choiceLabel(ANY_SLOT, places), null)
})
