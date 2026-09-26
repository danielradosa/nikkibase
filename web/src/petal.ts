export type Point = readonly [number, number]

export type Outline = readonly Point[]

export const SEGMENTS = 6

export const DOT = { orbit: 9, radius: 4.2 }
export const PETAL = { length: 36, halfWidth: 12.5, notch: 6, base: 2 }

export function circleOutline(orbit: number, radius: number): Outline {
  const k = (4 / 3) * Math.tan(Math.PI / 12) * radius
  const at = (deg: number): Point => {
    const a = (deg * Math.PI) / 180
    return [radius * Math.cos(a), -orbit + radius * Math.sin(a)]
  }
  const tangent = (deg: number): Point => {
    const a = (deg * Math.PI) / 180
    return [-Math.sin(a), Math.cos(a)]
  }
  const out: Point[] = [at(90)]
  for (let s = 0; s < SEGMENTS; s++) {
    const from = 90 + s * 60
    const to = from + 60
    const [px, py] = at(from)
    const [qx, qy] = at(to)
    const [tx, ty] = tangent(from)
    const [ux, uy] = tangent(to)
    out.push([px + k * tx, py + k * ty], [qx - k * ux, qy - k * uy], [qx, qy])
  }
  return out
}

export function petalOutline(length: number, halfWidth: number, notch: number, base: number): Outline {
  const L = length
  const w = halfWidth
  const cleft = L - notch
  const left: Point[] = [
    [0, -base],
    [-w * 0.42, -base - L * 0.1],
    [-w, -L * 0.4],
    [-w, -L * 0.64],
    [-w, -L * 0.84],
    [-w * 0.66, -L * 0.995],
    [-w * 0.34, -L],
    [-w * 0.16, -L * 1.002],
    [-w * 0.05, -cleft - notch * 0.3],
    [0, -cleft],
  ]
  const mirror = ([x, y]: Point): Point => [x === 0 ? 0 : -x, y]
  const right: Point[] = [
    mirror(left[8]),
    mirror(left[7]),
    mirror(left[6]),
    mirror(left[5]),
    mirror(left[4]),
    mirror(left[3]),
    mirror(left[2]),
    mirror(left[1]),
    mirror(left[0]),
  ]
  return [...left, ...right]
}

export function budOutline(length: number, halfWidth: number, base: number): Outline {
  return petalOutline(length * 0.8, halfWidth * 0.9, 0, base)
}

export function morphOutline(dot: Outline, bud: Outline, petal: Outline, t: number): Outline {
  const a = (1 - t) * (1 - t)
  const b = 2 * t * (1 - t)
  const c = t * t
  return dot.map(([x, y], i) => {
    const [bx, by] = bud[i]
    const [px, py] = petal[i]
    return [a * x + b * bx + c * px, a * y + b * by + c * py] as Point
  })
}

export function lerpOutline(a: Outline, b: Outline, t: number): Outline {
  return a.map(([ax, ay], i) => {
    const [bx, by] = b[i]
    return [ax * (1 - t) + bx * t, ay * (1 - t) + by * t] as Point
  })
}

export function toPath(o: Outline): string {
  const f = (n: number) => n.toFixed(2)
  let d = `M${f(o[0][0])} ${f(o[0][1])}`
  for (let i = 1; i < o.length; i += 3) {
    d += `C${f(o[i][0])} ${f(o[i][1])} ${f(o[i + 1][0])} ${f(o[i + 1][1])} ${f(o[i + 2][0])} ${f(o[i + 2][1])}`
  }
  return d + 'Z'
}

export const clamp01 = (x: number) => Math.min(1, Math.max(0, x))
export const easeOutCubic = (x: number) => 1 - (1 - clamp01(x)) ** 3
export const easeInOutCubic = (x: number) => {
  const t = clamp01(x)
  return t < 0.5 ? 4 * t * t * t : 1 - (-2 * t + 2) ** 3 / 2
}
export const easeInOutSine = (x: number) => (1 - Math.cos(Math.PI * clamp01(x))) / 2

export const SPIN = 360 / 1100

export const BLOOM = {
  settle: 950,
  morphAt: 150,
  morph: 950,
  stamensAt: 850,
  stamens: 400,
  pulseAt: 1100,
  pulse: 300,
  fadeAt: 1400,
  fade: 450,
}
export const BLOOM_END = BLOOM.fadeAt + BLOOM.fade

export const DIAMOND = 45

export const SIDE_PETAL = 0.86

export function settleAngle(angleAtDone: number): number {
  const ahead = angleAtDone + (SPIN * BLOOM.settle) / 3
  return DIAMOND + Math.ceil((ahead - DIAMOND) / 90) * 90
}

export type Frame = {
  angle: number
  morph: number
  stamens: number
  scale: number
  opacity: number
  finished: boolean
}

export function bloomFrame(t: number, angleAtDone: number): Frame {
  const u = clamp01(t / BLOOM.settle)
  const distance = settleAngle(angleAtDone) - angleAtDone
  const momentum = SPIN * BLOOM.settle
  const reach = -2 * u ** 3 + 3 * u ** 2
  const carry = u ** 3 - 2 * u ** 2 + u
  const fade = easeInOutCubic((t - BLOOM.fadeAt) / BLOOM.fade)
  const pulse = Math.sin(Math.PI * clamp01((t - BLOOM.pulseAt) / BLOOM.pulse))
  return {
    angle: angleAtDone + distance * reach + momentum * carry,
    morph: easeInOutSine((t - BLOOM.morphAt) / BLOOM.morph),
    stamens: easeOutCubic((t - BLOOM.stamensAt) / BLOOM.stamens),
    scale: 1 + 0.04 * pulse + 0.12 * fade,
    opacity: 1 - fade,
    finished: t >= BLOOM_END,
  }
}

export function dotOpacity(i: number, now: number): number {
  return 0.35 + 0.65 * (0.5 + 0.5 * Math.sin((now / 1000) * 2 * Math.PI - (i * Math.PI) / 2))
}
