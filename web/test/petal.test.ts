import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  BLOOM,
  BLOOM_END,
  DOT,
  PETAL,
  SEGMENTS,
  SPIN,
  DIAMOND,
  bloomFrame,
  budOutline,
  circleOutline,
  lerpOutline,
  morphOutline,
  petalOutline,
  settleAngle,
  toPath,
} from '../src/petal.ts'

const dot = circleOutline(DOT.orbit, DOT.radius)
const petal = petalOutline(PETAL.length, PETAL.halfWidth, PETAL.notch, PETAL.base)

test('the dot and the petal are drawn point for point alike, so one can become the other', () => {
  assert.equal(dot.length, 1 + SEGMENTS * 3)
  assert.equal(petal.length, dot.length)
  assert.equal(toPath(dot).match(/C/g)?.length, SEGMENTS)
  assert.match(toPath(petal), /^M[-\d. ]+(C[-\d. ]+){6}Z$/)
})

test('the dot is a circle at its place on the ring', () => {
  for (let i = 0; i < dot.length; i += 3) {
    const [x, y] = dot[i]
    assert.ok(Math.abs(Math.hypot(x, y + DOT.orbit) - DOT.radius) < 1e-9, `point ${i} is off the circle`)
  }
  assert.ok(Math.abs(dot[0][1] - (-DOT.orbit + DOT.radius)) < 1e-9)
})

test('the petal is symmetric, notched at its tip, and grows outward from the centre', () => {
  for (let i = 0; i < petal.length; i++) {
    const [x, y] = petal[i]
    const [mx, my] = petal[petal.length - 1 - i]
    assert.ok(Math.abs(x + mx) < 1e-9 && Math.abs(y - my) < 1e-9, `point ${i} has no mirror`)
  }
  const notch = petal[9]
  const lobe = petal[6]
  assert.equal(notch[0], 0)
  assert.ok(lobe[1] < notch[1], 'the lobes reach further out than the notch between them')
  assert.ok(petal.every(([, y]) => y < 0))
})

test('interpolation starts at the dot and ends at the petal', () => {
  assert.deepEqual(lerpOutline(dot, petal, 0), dot)
  assert.deepEqual(lerpOutline(dot, petal, 1), petal)
})

test('the bloom passes near the bud and lands exactly on the dot and the petal', () => {
  const bud = budOutline(PETAL.length, PETAL.halfWidth, PETAL.base)
  assert.equal(bud.length, dot.length)
  assert.deepEqual(morphOutline(dot, bud, petal, 0), dot)
  assert.deepEqual(morphOutline(dot, bud, petal, 1), petal)
  assert.ok(bud[9][1] <= bud[6][1] + 1e-9, 'the bud has no notch')
})

test('the spin always comes to rest facing the diamond, from any angle, never turning back', () => {
  for (let start = 0; start < 360; start += 3.7) {
    const rest = settleAngle(start)
    assert.equal(((rest - DIAMOND) % 90 + 90) % 90, 0, `from ${start} it rests at ${rest}`)
    let prev = bloomFrame(0, start).angle
    for (let t = 5; t <= BLOOM_END; t += 5) {
      const a = bloomFrame(t, start).angle
      assert.ok(a >= prev - 1e-9, `from ${start}, the spin turned back at ${t}ms`)
      prev = a
    }
    assert.ok(Math.abs(bloomFrame(BLOOM.settle, start).angle - rest) < 1e-9)
    assert.ok(Math.abs(bloomFrame(BLOOM_END, start).angle - rest) < 1e-9)
  }
})

test('the bloom slows the spin without a jolt, opens, and fades out', () => {
  const at = (t: number) => bloomFrame(t, 30)
  assert.equal(at(0).angle, 30)
  const speed = (at(1).angle - at(0).angle) / 1
  assert.ok(Math.abs(speed - SPIN) / SPIN < 0.02, `starts at ${speed} deg/ms, the spin was ${SPIN}`)
  assert.equal(at(BLOOM.morphAt).morph, 0)
  assert.equal(at(BLOOM.morphAt + BLOOM.morph).morph, 1)
  assert.equal(at(BLOOM.fadeAt).opacity, 1)
  let prev = at(0)
  for (let t = 10; t <= BLOOM_END; t += 10) {
    const f = at(t)
    assert.ok(f.morph >= prev.morph && f.opacity <= prev.opacity && f.angle >= prev.angle, `at ${t}ms`)
    prev = f
  }
  assert.equal(at(BLOOM_END).opacity, 0)
  assert.equal(at(BLOOM_END).finished, true)
  assert.equal(at(BLOOM_END - 1).finished, false)
})
