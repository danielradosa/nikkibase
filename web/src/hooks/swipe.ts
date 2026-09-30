export const SLOP = 10
export const EDGE = 24
export const HOLD_MS = 500
export const FLICK_PX = 40
export const FLICK_SPEED = 0.35
export const COMMIT_SHARE = 0.25
export const COMMIT_PX = 96
export const TRACK_MS = 100
export const PULL = 0.35
export const PULL_END = 0.12
export const PULL_MAX = 56

export type Axis = 'x' | 'y'
export type Step = -1 | 0 | 1
export type Sample = { x: number; t: number }

export function swipeAxis(dx: number, dy: number): Axis | null {
  if (Math.max(Math.abs(dx), Math.abs(dy)) < SLOP) return null
  return Math.abs(dx) > Math.abs(dy) * 1.5 ? 'x' : 'y'
}

export function track(samples: readonly Sample[], next: Sample): Sample[] {
  return [...samples.filter((s) => next.t - s.t <= TRACK_MS), next]
}

export function swipeSpeed(samples: readonly Sample[]): number {
  if (samples.length < 2) return 0
  const a = samples[0]
  const b = samples[samples.length - 1]
  return (b.x - a.x) / Math.max(b.t - a.t, 1)
}

export function swipeStep(dx: number, speed: number, width: number): Step {
  const far = Math.abs(dx) >= Math.min(width * COMMIT_SHARE, COMMIT_PX)
  const flick = Math.abs(dx) >= FLICK_PX && Math.abs(speed) >= FLICK_SPEED && Math.sign(speed) === Math.sign(dx)
  if (!far && !flick) return 0
  return dx < 0 ? 1 : -1
}

export function tabAfter(keys: readonly string[], current: string, step: Step): string | null {
  const i = keys.indexOf(current)
  if (i < 0 || step === 0) return null
  return keys[i + step] ?? null
}

export function swipePull(dx: number, open: boolean): number {
  const pull = dx * (open ? PULL : PULL_END)
  return Math.max(-PULL_MAX, Math.min(PULL_MAX, pull))
}

export function nearEdge(x: number, width: number): boolean {
  return x < EDGE || x > width - EDGE
}

export function blocksSwipe(start: Element | null, root: Element, focused: Element | null, overflowX: (el: Element) => string): boolean {
  for (let el = start; el && el !== root; el = el.parentElement) {
    if (el === focused && el.matches('input, textarea, [contenteditable]')) return true
    if (el.scrollWidth > el.clientWidth + 1) {
      const o = overflowX(el)
      if (o === 'auto' || o === 'scroll') return true
    }
  }
  return false
}
