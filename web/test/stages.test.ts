import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  caveatText,
  compareStages,
  coverageLabel,
  groupOptions,
  hasVariants,
  matches,
  missingMessage,
  orderModes,
  requirementLabel,
  resolveStage,
  rulesUnchecked,
  stagePlaceholder,
  stagesInMode,
  stageKey,
  tagLabel,
  weightLabel,
  type Stage,
} from '../src/stages.ts'

const W = [1, 2, 3, 4, 5]
const A = [0, 1, 2, 3, 4]

function stage(mode: Stage['mode'], name: string, extra: Partial<Stage> = {}): Stage {
  return { name, mode, weights: W, attrs: A, ...extra }
}

const story = (name: string, extra?: Partial<Stage>) => stage('Story', name, extra)
const names = (list: Stage[]) => list.map((s) => s.name)
const sorted = (list: Stage[]) => names([...list].sort(compareStages))

const shuffle = <T,>(list: T[]) => [...list].reverse()

test('a stage is keyed by mode and name, so Story 1-1 and Commission 1-1 stay apart', () => {
  assert.equal(stageKey(story('6-9')), 'Story/6-9')
  assert.equal(stageKey(stage('Commission', '6-9')), 'Commission/6-9')
})

const princessOnly = story('6-9', { tags: { 3: 1500 } })
const withVariant = story('6-9', {
  weights: [9, 9, 9, 9, 9],
  attrs: [5, 6, 7, 8, 9],
  tags: { 3: 1500 },
  variants: { maiden: { weights: [1, 1, 1, 1, 1], attrs: [0, 2, 4, 6, 8], tags: { 7: 800 } } },
})

test('Maiden on a Story stage with a variant scores the variant', () => {
  const r = resolveStage(withVariant, 'Maiden')
  assert.deepEqual(r.weights, [1, 1, 1, 1, 1])
  assert.deepEqual(r.attrs, [0, 2, 4, 6, 8])
  assert.deepEqual(r.tags, { 7: 800 })
  assert.equal(r.name, '6-9')
  assert.equal(r.mode, 'Story')
})

test('Princess always scores the top-level numbers', () => {
  assert.equal(resolveStage(withVariant, 'Princess'), withVariant)
  assert.equal(resolveStage(princessOnly, 'Princess'), princessOnly)
})

test('Maiden without a variant shares the Princess numbers', () => {
  assert.equal(resolveStage(princessOnly, 'Maiden'), princessOnly)
})

test('a variant without tags pays no tag, whatever Princess pays', () => {
  const s = story('6-9', { tags: { 3: 1500 }, variants: { maiden: { weights: [2, 2, 2, 2, 2], attrs: A } } })
  const r = resolveStage(s, 'Maiden')
  assert.equal(r.tags, undefined)
  assert.deepEqual(r.weights, [2, 2, 2, 2, 2])
})

test('Maiden takes the variant rules when it has its own, and the stage rules otherwise', () => {
  const rules = { styles: [], require: [[40080], [50091]] }
  const inherits = story('2-7', { rules, variants: { maiden: { weights: [2, 2, 2, 2, 2], attrs: A } } })
  assert.deepEqual(resolveStage(inherits, 'Maiden').rules, rules)
  assert.deepEqual(resolveStage(inherits, 'Maiden').weights, [2, 2, 2, 2, 2])

  const own = { styles: ['Unisex'], require: [[20165]] }
  const replaces = story('2-7', { rules, variants: { maiden: { weights: [2, 2, 2, 2, 2], attrs: A, rules: own } } })
  assert.deepEqual(resolveStage(replaces, 'Maiden').rules, own)
  assert.deepEqual(resolveStage(replaces, 'Princess').rules, rules)
})

const itemNames = new Map([
  [40080, 'Sleepy Top'],
  [50091, 'Sleepy Pants'],
  [83283, 'Wizard Hat'],
  [83287, 'Wizard Staff'],
])

test('required items read as names, sets joined with and, choices within a set with or', () => {
  assert.equal(requirementLabel([[40080]], itemNames), 'Sleepy Top')
  assert.equal(requirementLabel([[40080], [50091]], itemNames), 'Sleepy Top and Sleepy Pants')
  assert.equal(requirementLabel([[83283, 83287]], itemNames), 'Wizard Hat or Wizard Staff')
  assert.equal(
    requirementLabel([[40080], [50091], [83283, 83287]], itemNames),
    'Sleepy Top, Sleepy Pants and Wizard Hat or Wizard Staff',
  )
})

test('an item without a name is shown by its number', () => {
  assert.equal(requirementLabel([[40080, 7]], itemNames), 'Sleepy Top or #7')
  assert.equal(requirementLabel([[1, 2, 3]], new Map()), '#1, #2 or #3')
})

test('the missing message names what the player lacks, one item or several', () => {
  assert.equal(missingMessage([[40080]], itemNames), "You don't own Sleepy Top, which this stage requires")
  assert.equal(
    missingMessage([[40080], [83283, 83287]], itemNames),
    "You don't own Sleepy Top and Wizard Hat or Wizard Staff, which this stage requires",
  )
})

test('only rules beyond required items are flagged as not checked', () => {
  assert.equal(rulesUnchecked(undefined), false)
  assert.equal(rulesUnchecked({ styles: [] }), true)
  assert.equal(rulesUnchecked({ styles: ['Unisex'] }), true)
  assert.equal(rulesUnchecked({ styles: [], require: [[40080]] }), false)
  assert.equal(rulesUnchecked({ require: [[40080]] }), false)
  assert.equal(rulesUnchecked({ styles: ['Pajamas'], require: [[40080]] }), true)
})

test('only Story has a difficulty', () => {
  const odd = stage('Commission', '6-7', { variants: withVariant.variants })
  assert.equal(resolveStage(odd, 'Maiden'), odd)
  assert.equal(hasVariants(odd), false)
})

test('hasVariants is true only for a stage that carries a Maiden variant', () => {
  assert.equal(hasVariants(withVariant), true)
  assert.equal(hasVariants(princessOnly), false)
  assert.equal(hasVariants(story('6-9', { variants: {} })), false)
})

test('Story runs by volume, chapter, main stages, then side stages', () => {
  const order = [
    '1-1', '1-9', '2-1', '2-9', '2-Side 1', '2-Side 2', '3-1', '3-2', '3-10', '3-12', '3-Side 1',
    '9-5', '9-6-1', '9-6-2', '9-7', '9-9-1', '9-9-2', '9-9-3', '9-Side 1', '9-Side 3',
    '10-1', '10-9-1', '10-9-2', '10-Side 1', '19-9', '19-Side 3',
    'II-1-1', 'II-1-7', 'II-1-Side 1', 'II-4-Side 2', 'II-5-1', 'II-10-1',
    'III-1-1', 'III-3-Side 2',
  ]
  assert.deepEqual(sorted(shuffle(order).map((n) => story(n))), order)
})

test('the orderings players notice first', () => {
  const before = (a: string, b: string) => assert.ok(compareStages(story(a), story(b)) < 0, `${a} before ${b}`)
  before('19-Side 3', 'II-1-1')
  before('9-9-3', '10-1')
  before('2-9', '2-Side 1')
  before('9-6', '9-6-1')
  before('9-6-2', '9-7')
  before('II-9-Side 3', 'III-1-1')
})

test('Commission runs by act, then stage, numerically', () => {
  const order = ['1-1', '1-7', '2-1', '9-1', '9-7', '10-1', '10-7', '12-3', '20-7']
  assert.deepEqual(sorted(shuffle(order).map((n) => stage('Commission', n))), order)
})

test('a name that does not parse goes last rather than breaking the sort', () => {
  assert.deepEqual(sorted([story('Prologue'), story('2-1'), story('1-1')]), ['1-1', '2-1', 'Prologue'])
})

test('Story options are grouped by volume and chapter, in game order', () => {
  const opts = groupOptions(shuffle(['6-2', '6-1', '6-Side 1', 'II-6-3', '10-1', 'III-3-Side 2'].map((n) => story(n))))
  assert.deepEqual(opts, [
    {
      label: 'Volume I · Chapter 6',
      options: [
        { value: 'Story/6-1', label: '6-1' },
        { value: 'Story/6-2', label: '6-2' },
        { value: 'Story/6-Side 1', label: '6-Side 1' },
      ],
    },
    { label: 'Volume I · Chapter 10', options: [{ value: 'Story/10-1', label: '10-1' }] },
    { label: 'Volume II · Chapter 6', options: [{ value: 'Story/II-6-3', label: 'II-6-3' }] },
    { label: 'Volume III · Chapter 3', options: [{ value: 'Story/III-3-Side 2', label: 'III-3-Side 2' }] },
  ])
})

test('Commission options are grouped by act', () => {
  const opts = groupOptions(['10-1', '9-7', '12-1', '12-2'].map((n) => stage('Commission', n)))
  assert.deepEqual(
    opts.map((o) => o.label),
    ['Act 9', 'Act 10', 'Act 12'],
  )
  assert.deepEqual(opts[2], {
    label: 'Act 12',
    options: [
      { value: 'Commission/12-1', label: '12-1' },
      { value: 'Commission/12-2', label: '12-2' },
    ],
  })
})

test('Co-op and Arena are flat lists in the order the data gives them', () => {
  const coop = ['Orlando - Unisex', 'Ace - Traditional Style', 'Bobo - Pet'].map((n) => stage('Co-op', n))
  assert.deepEqual(groupOptions(coop), [
    { value: 'Co-op/Orlando - Unisex', label: 'Orlando - Unisex' },
    { value: 'Co-op/Ace - Traditional Style', label: 'Ace - Traditional Style' },
    { value: 'Co-op/Bobo - Pet', label: 'Bobo - Pet' },
  ])
  assert.deepEqual(groupOptions([stage('Arena', 'Beach Party')]), [{ value: 'Arena/Beach Party', label: 'Beach Party' }])
})

test('an unparseable Story or Commission name still gets a place to live', () => {
  const opts = groupOptions([story('Prologue'), story('1-1')])
  assert.deepEqual(opts.map((o) => o.label), ['Volume I · Chapter 1', 'Other'])
})

const leaf = (s: Stage) => ({ value: stageKey(s), label: s.name })

test('search matches the name anywhere, ignoring case', () => {
  assert.equal(matches('6-3', leaf(story('II-6-3'))), true)
  assert.equal(matches('side', leaf(story('II-4-Side 2'))), true)
  assert.equal(matches('ace', leaf(stage('Co-op', 'Ace - Traditional Style'))), true)
  assert.equal(matches('ace trad', leaf(stage('Co-op', 'Ace - Traditional Style'))), true)
  assert.equal(matches('beach', leaf(stage('Arena', 'Beach Party'))), true)
  assert.equal(matches('office', leaf(stage('Arena', 'Beach Party'))), false)
  assert.equal(matches('', leaf(story('1-1'))), true)
})

test('a volume-qualified search finds the stage in that volume only', () => {
  assert.equal(matches('v2 6-3', leaf(story('II-6-3'))), true)
  assert.equal(matches('v2 6-3', leaf(story('6-3'))), false)
  assert.equal(matches('v2-6-3', leaf(story('II-6-3'))), true)
  assert.equal(matches('V2 6-3', leaf(story('II-16-3'))), false)
  assert.equal(matches('V1 6-9', leaf(story('6-9'))), true)
  assert.equal(matches('V1 6-9', leaf(story('II-6-9'))), false)
  assert.equal(matches('v3 3-side', leaf(story('III-3-Side 2'))), true)
  assert.equal(matches('v2 side', leaf(story('II-4-Side 2'))), true)
  assert.equal(matches('v2 side', leaf(story('4-Side 2'))), false)
  assert.equal(matches('v2', leaf(story('II-1-1'))), true)
  assert.equal(matches('v2', leaf(story('1-1'))), false)
})

test('volume qualifiers mean nothing outside Story', () => {
  assert.equal(matches('v1 6-7', leaf(stage('Commission', '6-7'))), false)
})

test('a group is never matched itself, so search filters its stages', () => {
  const [group] = groupOptions([story('6-1'), story('6-2')])
  assert.equal(matches('Chapter 6', group), false)
  assert.equal(matches('', group), false)
})

test('the coverage line is worked out from the data', () => {
  const fixture = [
    story('1-1'),
    story('19-Side 3'),
    story('II-5-7'),
    story('III-3-Side 2'),
    story('III-1-1'),
    stage('Commission', '1-1'),
    stage('Commission', '20-7'),
    stage('Commission', '7-3'),
    stage('Co-op', 'Bobo - Pet'),
    stage('Arena', 'Beach Party'),
  ]
  assert.equal(
    coverageLabel(fixture),
    "Has Story up to Volume III Chapter 3, Commission Acts\u00a01\u2060–\u206020, Arena and Co-op. Newer stages aren't in yet.",
  )
})

test('the coverage line says only what is there', () => {
  assert.equal(
    coverageLabel([story('2-1'), story('II-5-Side 3'), stage('Commission', '15-7'), stage('Commission', '1-1')]),
    "Has Story up to Volume II Chapter 5 and Commission Acts\u00a01\u2060–\u206015. Newer stages aren't in yet.",
  )
  assert.equal(coverageLabel([story('1-1'), story('19-9')]), "Has Story up to Chapter 19. Newer stages aren't in yet.")
  assert.equal(coverageLabel([stage('Commission', '4-1')]), "Has Commission Act\u00a04. Newer stages aren't in yet.")
  assert.equal(coverageLabel([stage('Arena', 'Beach Party')]), "Has Arena. Newer stages aren't in yet.")
  assert.equal(coverageLabel([]), '')
})

test('the stage search asks for a stage of the chosen mode, with an example where names are numbers', () => {
  assert.equal(stagePlaceholder('Story'), 'Pick a Story stage, e.g. 5-11')
  assert.equal(stagePlaceholder('Commission'), 'Pick a Commission stage, e.g. 3-7')
  assert.equal(stagePlaceholder('Arena'), 'Pick an Arena stage')
  assert.equal(stagePlaceholder('Co-op'), 'Pick a Co-op stage')
})

test('modes are offered in the order a player wants them, whatever order the data has', () => {
  const list = [stage('Arena', 'a'), stage('Co-op', 'c'), stage('Story', '1-1'), stage('Commission', '1-1')]
  assert.deepEqual(orderModes(list), ['Story', 'Commission', 'Co-op', 'Arena'])
})

test('only modes present in the data are offered', () => {
  assert.deepEqual(orderModes([stage('Arena', 'a')]), ['Arena'])
  assert.deepEqual(orderModes([]), [])
})

test('a mode lists only its own stages', () => {
  const list = [story('1-1'), stage('Commission', '1-1'), story('1-2')]
  assert.deepEqual(names(stagesInMode(list, 'Story')), ['1-1', '1-2'])
  assert.deepEqual(names(stagesInMode(list, 'Arena')), [])
})


test('a stage weight reads as a multiplier on its style', () => {
  assert.equal(weightLabel('Lively', 3), 'Lively ×3')
  assert.equal(weightLabel('Cute', 1.3333), 'Cute ×1.33')
  assert.equal(weightLabel('Simple', 0.5), 'Simple ×0.5')
})

test('a tag award says it is paid for each tagged item', () => {
  assert.equal(tagLabel('Swordsman', 35000), 'Swordsman +35,000 each')
})

test('the rules caveat says what can fail, and names the styles when the stage lists them', () => {
  const base = "Some items or styles score F on this stage. NikkiBase doesn't check this yet, so a suggested item could fail."
  assert.equal(caveatText([]), base)
  assert.equal(caveatText(['Unisex']), `${base} Styles: Unisex.`)
  assert.equal(caveatText(['Pajamas', 'Swimsuit']), `${base} Styles: Pajamas, Swimsuit.`)
})
