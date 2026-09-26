import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  BLOOM,
  BLOOM_END,
  BLOOM_KEY,
  DOT,
  FADE,
  PLACES,
  PETAL,
  SEGMENTS,
  SPIN,
  DIAMOND,
  bloomFrame,
  bloomSeen,
  blossomExit,
  budOutline,
  circleOutline,
  dotOpacity,
  fadeFrame,
  frameStep,
  lerpOutline,
  markBloomed,
  morphOutline,
  petalOutline,
  settleAngle,
  spinAngle,
  spinOrigin,
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

test('the four dots sit on the ring at the corners of a square, each pulsing a quarter second after the last', () => {
  assert.deepEqual(PLACES, [45, 135, 225, 315])
  for (let i = 0; i < 4; i++) {
    for (let t = 0; t < 2000; t += 37) {
      assert.ok(Math.abs(dotOpacity(i, t) - dotOpacity(0, t - 250 * i)) < 1e-9, `dot ${i} at ${t}ms`)
    }
  }
  assert.deepEqual([0, 1, 2, 3].map((i) => +dotOpacity(i, 0).toFixed(3)), [0.675, 0.35, 0.675, 1])
})

test('the quick fade keeps the spin going and fades out in 200 ms', () => {
  assert.equal(FADE, 200)
  const at = (t: number) => fadeFrame(t, 30)
  assert.equal(at(0).angle, 30)
  assert.equal(at(0).opacity, 1)
  assert.ok(Math.abs(at(100).angle - (30 + SPIN * 100)) < 1e-9)
  let prev = at(0)
  for (let t = 5; t <= FADE; t += 5) {
    const f = at(t)
    assert.ok(f.opacity <= prev.opacity && f.angle > prev.angle, `at ${t}ms`)
    assert.equal(f.morph, 0)
    assert.equal(f.stamens, 0)
    assert.equal(f.scale, 1)
    prev = f
  }
  assert.ok(at(1).opacity < 1)
  assert.equal(at(FADE).opacity, 0)
  assert.equal(at(FADE).finished, true)
  assert.equal(at(FADE - 1).finished, false)
})

test('the full bloom plays on the first visit only, and a blocked storage counts as a first visit', () => {
  assert.equal(BLOOM_KEY, 'nikkibase.bloomed')
  assert.equal(bloomSeen(() => '1'), true)
  assert.equal(bloomSeen(() => null), false)
  assert.equal(
    bloomSeen(() => {
      throw new Error('blocked')
    }),
    false,
  )
  let wrote = 0
  markBloomed(() => {
    wrote++
  })
  assert.equal(wrote, 1)
  assert.doesNotThrow(() =>
    markBloomed(() => {
      throw new Error('full')
    }),
  )
})

test('reduced motion stays quiet, a return visit fades, a first visit blooms', () => {
  assert.equal(blossomExit(true, false), 'quiet')
  assert.equal(blossomExit(true, true), 'quiet')
  assert.equal(blossomExit(false, true), 'fade')
  assert.equal(blossomExit(false, false), 'bloom')
})

test('the React spinner takes over the static spin from its start time, so the hand-off keeps the angle', () => {
  assert.equal(spinOrigin({ startTime: 120, currentTime: 900 }, 1500), 120)
  assert.equal(spinOrigin({ startTime: null, currentTime: 900 }, 1500), 600)
  assert.equal(spinOrigin({ startTime: null, currentTime: null }, 1500), null)
  assert.equal(spinOrigin({ startTime: null, currentTime: 900 }, null), null)
  assert.equal(spinOrigin(undefined, 1500), null)
  assert.equal(spinAngle(120, 120), 0)
  assert.ok(Math.abs(spinAngle(120 + 550, 120) - 180) < 1e-9)
  assert.ok(Math.abs(spinAngle(120 + 1100 + 275, 120) - 90) < 1e-9)
  assert.ok(Math.abs(spinAngle(100, 120) - (360 - 20 * SPIN)) < 1e-9)
  let prev = spinAngle(1000, 120)
  for (let now = 1016; now < 1600; now += 16) {
    const angle = spinAngle(now, 120)
    assert.ok(angle >= 0 && angle < 360)
    assert.ok(Math.abs(((angle - prev + 360) % 360) - SPIN * 16) < 1e-9, `at ${now}ms`)
    prev = angle
  }
})

test('a frame step is never negative and never longer than 64 ms, and the first frame does not move', () => {
  assert.equal(frameStep(500, null), 0)
  assert.equal(frameStep(516, 500), 16)
  assert.equal(frameStep(420, 500), 0)
  assert.equal(frameStep(900, 500), 64)
})
