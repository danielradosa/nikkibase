import { test } from 'node:test'
import assert from 'node:assert/strict'
import { jumpOptions, pickerTree, placeOf, stepStage, type Stage } from '../../src/outfit/stages.ts'

const W = [1, 2, 3, 4, 5]
const A = [0, 1, 2, 3, 4]

const stage = (mode: Stage['mode'], name: string): Stage => ({ name, mode, weights: W, attrs: A })
const story = (name: string) => stage('Story', name)

const fixture: Stage[] = [
  story('II-5-3'),
  story('5-10'),
  story('5-Side 1'),
  story('5-2'),
  story('5-1'),
  story('6-1'),
  story('19-Side 3'),
  story('II-1-1'),
  story('III-3-Side 2'),
  story('III-3-5'),
  story('II-5-1'),
  story('9-9-1'),
  stage('Commission', '10-1'),
  stage('Commission', '2-7'),
  stage('Commission', '2-1'),
  stage('Dreamweaver', 'Lunar - Millennium Dream 6'),
  stage('Dreamweaver', 'Lunar - Hidden String 2'),
  stage('Dreamweaver', 'Orlando - Officer & Wine 4'),
  stage('Arena', 'Beach Party'),
]

const shape = (mode: string) =>
  pickerTree(fixture, mode).map((b) => ({
    label: b.label,
    groups: b.groups.map((g) => [g.label, g.leaves.map((l) => l.label)]),
  }))

test('story stages sit under their volume and chapter, labelled without the volume', () => {
  assert.deepEqual(shape('Story'), [
    { label: 'I', groups: [['5', ['5-1', '5-2', '5-10', 'Side 1']], ['6', ['6-1']], ['9', ['9-9-1']], ['19', ['Side 3']]] },
    { label: 'II', groups: [['1', ['1-1']], ['5', ['5-1', '5-3']]] },
    { label: 'III', groups: [['3', ['3-5', 'Side 2']]] },
  ])
  const leaf = pickerTree(fixture, 'Story')[1].groups[1].leaves[1]
  assert.deepEqual(leaf, { key: 'Story/II-5-3', label: '5-3', name: 'II-5-3' })
})

test('commission stages sit under their act, in number order', () => {
  assert.deepEqual(shape('Commission'), [{ label: '', groups: [['2', ['2-1', '2-7']], ['10', ['10-1']]] }])
})

test('dream weaver stages sit under their character, in the order the data gives them', () => {
  assert.deepEqual(shape('Dreamweaver'), [
    { label: '', groups: [['Lunar', ['Millennium Dream 6', 'Hidden String 2']], ['Orlando', ['Officer & Wine 4']]] },
  ])
})

test('modes picked from a list have no chips', () => {
  assert.deepEqual(pickerTree(fixture, 'Arena'), [])
  assert.deepEqual(pickerTree(fixture, 'Co-op'), [])
})

test('a stage is found under its branch and group', () => {
  const tree = pickerTree(fixture, 'Story')
  assert.deepEqual(placeOf(tree, 'Story/II-5-3'), { branch: tree[1].key, group: tree[1].groups[1].key })
  assert.equal(placeOf(tree, 'Story/99-1'), null)
  const acts = pickerTree(fixture, 'Commission')
  assert.deepEqual(placeOf(acts, 'Commission/10-1'), { branch: acts[0].key, group: acts[0].groups[1].key })
})

test('the arrows step through stages in game order, across chapters and volumes, and stop at the ends', () => {
  assert.equal(stepStage(fixture, 'Story', 'Story/5-2', 1), 'Story/5-10')
  assert.equal(stepStage(fixture, 'Story', 'Story/5-10', 1), 'Story/5-Side 1')
  assert.equal(stepStage(fixture, 'Story', 'Story/5-Side 1', 1), 'Story/6-1')
  assert.equal(stepStage(fixture, 'Story', 'Story/19-Side 3', 1), 'Story/II-1-1')
  assert.equal(stepStage(fixture, 'Story', 'Story/II-1-1', -1), 'Story/19-Side 3')
  assert.equal(stepStage(fixture, 'Story', 'Story/5-1', -1), null)
  assert.equal(stepStage(fixture, 'Story', 'Story/III-3-Side 2', 1), null)
  assert.equal(stepStage(fixture, 'Commission', 'Commission/2-7', 1), 'Commission/10-1')
  assert.equal(stepStage(fixture, 'Dreamweaver', 'Dreamweaver/Lunar - Hidden String 2', 1), 'Dreamweaver/Orlando - Officer & Wine 4')
  assert.equal(stepStage(fixture, 'Story', 'Story/99-1', 1), null)
})

const labels = (mode: string, query: string, limit?: number) => jumpOptions(fixture, mode, query, limit).map((o) => o.label)

test('jumping puts the stage named first, then the ones it starts, then other volumes', () => {
  assert.deepEqual(labels('Story', '5-1'), ['5-1', '5-10', 'II-5-1'])
  assert.deepEqual(labels('Story', 'II-5-3'), ['II-5-3'])
  assert.deepEqual(labels('Story', '5 side 1'), ['5-Side 1'])
  assert.deepEqual(labels('Story', 'v2 5-3'), ['II-5-3'])
  assert.deepEqual(labels('Commission', '2'), ['2-1', '2-7'])
  assert.deepEqual(labels('Dreamweaver', 'hidden'), ['Lunar - Hidden String 2'])
})

test('jumping lists a few stages at most, and nothing for an empty query', () => {
  assert.deepEqual(labels('Story', '5', 2), ['5-1', '5-2'])
  assert.deepEqual(labels('Story', '  '), [])
  assert.deepEqual(jumpOptions(fixture, 'Story', '6-1')[0], { value: 'Story/6-1', label: '6-1' })
})
