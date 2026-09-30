import { test } from 'node:test'
import assert from 'node:assert/strict'
import { EDGE, PULL_MAX, SLOP, TRACK_MS, blocksSwipe, nearEdge, swipeAxis, swipePull, swipeSpeed, swipeStep, tabAfter, track } from '../../src/hooks/swipe.ts'

const TABS = ['outfit', 'worth', 'items']

type Fake = {
  parentElement: Fake | null
  scrollWidth: number
  clientWidth: number
  tag: string
  overflowX: string
  matches(selector: string): boolean
}

function fake(tag: string, parent: Fake | null, size: { scrollWidth?: number; clientWidth?: number; overflowX?: string } = {}): Fake {
  return {
    parentElement: parent,
    scrollWidth: size.scrollWidth ?? 300,
    clientWidth: size.clientWidth ?? 300,
    tag,
    overflowX: size.overflowX ?? 'visible',
    matches: (selector) => selector.split(',').map((s) => s.trim()).includes(tag),
  }
}

const el = (f: Fake | null) => f as unknown as Element
const overflowOf = (e: Element) => (e as unknown as Fake).overflowX

test('a gesture has no axis until it leaves the slop, then goes to the clearly longer side', () => {
  assert.equal(swipeAxis(SLOP - 1, 0), null)
  assert.equal(swipeAxis(-(SLOP - 1), SLOP - 1), null)
  assert.equal(swipeAxis(SLOP, 0), 'x')
  assert.equal(swipeAxis(-30, 10), 'x')
  assert.equal(swipeAxis(0, SLOP), 'y')
  assert.equal(swipeAxis(30, 25), 'y')
  assert.equal(swipeAxis(15, 10), 'y')
})

test('a swipe left goes to the next tab and a swipe right to the one before', () => {
  assert.equal(swipeStep(-120, -0.3, 375), 1)
  assert.equal(swipeStep(120, 0.3, 375), -1)
})

test('a slow short drag does not switch, a long one or a quick flick does', () => {
  assert.equal(swipeStep(-60, -0.1, 375), 0)
  assert.equal(swipeStep(-94, -0.1, 375), 1)
  assert.equal(swipeStep(-50, -0.5, 375), 1)
  assert.equal(swipeStep(-39, -2, 375), 0)
  assert.equal(swipeStep(0, 0, 375), 0)
})

test('a flick back the other way at the end does not count as a flick', () => {
  assert.equal(swipeStep(-60, 0.8, 375), 0)
  assert.equal(swipeStep(60, -0.8, 375), 0)
})

test('the distance needed is a quarter of the width, at most 96 px', () => {
  assert.equal(swipeStep(-79, 0, 320), 0)
  assert.equal(swipeStep(-80, 0, 320), 1)
  assert.equal(swipeStep(-95, 0, 800), 0)
  assert.equal(swipeStep(-96, 0, 800), 1)
})

test('speed is measured over the last moments of the gesture only', () => {
  let samples = [{ x: 300, t: 0 }]
  samples = track(samples, { x: 298, t: 300 })
  samples = track(samples, { x: 270, t: 340 })
  samples = track(samples, { x: 240, t: 380 })
  assert.deepEqual(samples.map((s) => s.t), [300, 340, 380])
  assert.equal(swipeSpeed(samples), -58 / 80)
  assert.ok(swipeSpeed(samples) <= -0.35)
  assert.equal(swipeSpeed(track(samples, { x: 240, t: 380 + TRACK_MS + 1 })), 0)
})

test('a gesture with one sample has no speed', () => {
  assert.equal(swipeSpeed([]), 0)
  assert.equal(swipeSpeed([{ x: 10, t: 5 }]), 0)
  assert.equal(swipeSpeed([{ x: 10, t: 5 }, { x: 30, t: 5 }]), 20)
})

test('tabs follow their order and stop at either end', () => {
  assert.equal(tabAfter(TABS, 'outfit', 1), 'worth')
  assert.equal(tabAfter(TABS, 'worth', 1), 'items')
  assert.equal(tabAfter(TABS, 'worth', -1), 'outfit')
  assert.equal(tabAfter(TABS, 'items', 1), null)
  assert.equal(tabAfter(TABS, 'outfit', -1), null)
  assert.equal(tabAfter(TABS, 'worth', 0), null)
  assert.equal(tabAfter(TABS, 'lost', 1), null)
})

test('the page follows the finger less at the last tab and never past the limit', () => {
  assert.equal(swipePull(-100, true), -35)
  assert.equal(swipePull(100, false), 12)
  assert.equal(swipePull(-1000, true), -PULL_MAX)
  assert.equal(swipePull(1000, false), PULL_MAX)
  assert.equal(swipePull(0, true), 0)
})

test('gestures from the screen edges are left to the browser', () => {
  assert.equal(nearEdge(EDGE - 1, 375), true)
  assert.equal(nearEdge(EDGE, 375), false)
  assert.equal(nearEdge(375 - EDGE, 375), false)
  assert.equal(nearEdge(375 - EDGE + 1, 375), true)
})

test('a gesture that starts inside something that scrolls sideways stays with it', () => {
  const root = fake('div', null)
  const strip = fake('div', root, { scrollWidth: 520, clientWidth: 300, overflowX: 'auto' })
  const button = fake('button', strip)
  assert.equal(blocksSwipe(el(button), el(root), null, overflowOf), true)
  const scroller = fake('div', root, { scrollWidth: 520, clientWidth: 300, overflowX: 'scroll' })
  assert.equal(blocksSwipe(el(scroller), el(root), null, overflowOf), true)
})

test('content that overflows without scrolling, or fits, does not stop a swipe', () => {
  const root = fake('div', null)
  const clipped = fake('span', root, { scrollWidth: 520, clientWidth: 300, overflowX: 'hidden' })
  assert.equal(blocksSwipe(el(clipped), el(root), null, overflowOf), false)
  const fits = fake('div', root, { scrollWidth: 300, clientWidth: 300, overflowX: 'auto' })
  assert.equal(blocksSwipe(el(fits), el(root), null, overflowOf), false)
  const rounding = fake('div', root, { scrollWidth: 301, clientWidth: 300, overflowX: 'auto' })
  assert.equal(blocksSwipe(el(rounding), el(root), null, overflowOf), false)
})

test('a focused text field keeps the gesture, an unfocused one does not', () => {
  const root = fake('div', null)
  const input = fake('input', root)
  const area = fake('textarea', root)
  assert.equal(blocksSwipe(el(input), el(root), el(input), overflowOf), true)
  assert.equal(blocksSwipe(el(area), el(root), el(area), overflowOf), true)
  assert.equal(blocksSwipe(el(input), el(root), null, overflowOf), false)
  const button = fake('button', root)
  assert.equal(blocksSwipe(el(button), el(root), el(button), overflowOf), false)
})

test('only the elements inside the swipe area are looked at', () => {
  const page = fake('div', null, { scrollWidth: 900, clientWidth: 300, overflowX: 'auto' })
  const root = fake('div', page)
  const text = fake('p', root)
  assert.equal(blocksSwipe(el(text), el(root), null, overflowOf), false)
  assert.equal(blocksSwipe(null, el(root), null, overflowOf), false)
})
