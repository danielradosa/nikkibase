import { useEffect, useRef, useState } from 'react'
import {
  DOT,
  PETAL,
  SIDE_PETAL,
  SPIN,
  bloomFrame,
  budOutline,
  circleOutline,
  dotOpacity,
  morphOutline,
  petalOutline,
  settleAngle,
  toPath,
} from './petal'

const DOT_SHAPE = circleOutline(DOT.orbit, DOT.radius)
const PETAL_SHAPE = petalOutline(PETAL.length, PETAL.halfWidth, PETAL.notch, PETAL.base)
const BUD_SHAPE = budOutline(PETAL.length, PETAL.halfWidth, PETAL.base)
const DOT_PATH = toPath(DOT_SHAPE)
const PLACES = [45, 135, 225, 315]
const STAMENS = [0, 72, 144, 216, 288]
const QUIET_FADE = 240

export default function Blossom({
  done,
  failed,
  onUncover,
}: {
  done: boolean
  failed: boolean
  onUncover: () => void
}) {
  const [gone, setGone] = useState(false)
  const [quiet] = useState(
    () => typeof window !== 'undefined' && !!window.matchMedia?.('(prefers-reduced-motion: reduce)').matches,
  )
  const screen = useRef<HTMLDivElement>(null)
  const turn = useRef<SVGGElement>(null)
  const shape = useRef<SVGPathElement>(null)
  const petals = useRef<(SVGUseElement | null)[]>([])
  const centre = useRef<SVGGElement>(null)
  const doneRef = useRef(done)
  doneRef.current = done
  const uncovered = useRef(false)
  const uncoverRef = useRef(onUncover)
  uncoverRef.current = onUncover
  const uncover = () => {
    if (uncovered.current) return
    uncovered.current = true
    uncoverRef.current()
  }

  useEffect(() => {
    if (!failed) return
    setGone(true)
    uncover()
  }, [failed]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (quiet || failed || gone) return
    let frame = 0
    let last = performance.now()
    let angle = 0
    let doneAt: number | null = null
    let angleAtDone = 0
    let drawnMorph = 0
    let finalLength = PLACES.map(() => 1)

    const tick = (now: number) => {
      const dt = Math.min(now - last, 64)
      last = now
      if (doneAt === null && doneRef.current) {
        doneAt = now
        angleAtDone = angle
        const rest = settleAngle(angleAtDone)
        finalLength = PLACES.map((deg) => (Math.round((deg + rest) / 90) % 2 === 0 ? 1 : SIDE_PETAL))
        centre.current?.setAttribute('transform', `rotate(${(-rest).toFixed(2)})`)
      }
      let morph = 0
      let stamens = 0
      let scale = 1
      let opacity = 1
      if (doneAt === null) {
        angle = (angle + SPIN * dt) % 360
      } else {
        const f = bloomFrame(now - doneAt, angleAtDone)
        if (f.finished) {
          uncover()
          setGone(true)
          return
        }
        if (f.opacity < 1 && !uncovered.current) {
          if (screen.current) screen.current.style.pointerEvents = 'none'
          uncover()
        }
        angle = f.angle
        morph = f.morph
        stamens = f.stamens
        scale = f.scale
        opacity = f.opacity
      }

      turn.current?.setAttribute('transform', `rotate(${angle.toFixed(2)}) scale(${scale.toFixed(4)})`)
      const changed = morph !== drawnMorph
      if (changed) {
        shape.current?.setAttribute('d', toPath(morphOutline(DOT_SHAPE, BUD_SHAPE, PETAL_SHAPE, morph)))
        shape.current?.setAttribute('stroke-opacity', (0.55 * morph).toFixed(3))
        drawnMorph = morph
      }
      petals.current.forEach((el, i) => {
        const o = dotOpacity(i, now) * (1 - morph) + morph
        el?.setAttribute('opacity', o.toFixed(3))
        if (changed) {
          const length = 1 + (finalLength[i] - 1) * morph
          el?.setAttribute('transform', `rotate(${PLACES[i]}) scale(1 ${length.toFixed(4)})`)
        }
      })
      centre.current?.setAttribute('opacity', stamens.toFixed(3))
      if (screen.current) screen.current.style.opacity = opacity.toFixed(3)
      frame = requestAnimationFrame(tick)
    }
    frame = requestAnimationFrame(tick)
    return () => cancelAnimationFrame(frame)
  }, [quiet, failed, gone])

  useEffect(() => {
    if (!quiet || !done || failed) return
    uncover()
    const t = window.setTimeout(() => setGone(true), QUIET_FADE)
    return () => window.clearTimeout(t)
  }, [quiet, done, failed]) // eslint-disable-line react-hooks/exhaustive-deps

  if (gone || failed) return null
  return (
    <div
      ref={screen}
      className={`nb-blossom${done ? ' is-done' : ''}${done && quiet ? ' is-quiet-fade' : ''}`}
      aria-hidden="true"
    >
      <svg className="nb-blossom-art" viewBox="-50 -50 100 100" focusable="false">
        <defs>
          <radialGradient id="nb-blossom-fill" gradientUnits="userSpaceOnUse" cx="0" cy="0" r="40">
            <stop offset="0" stopColor="#b93c6f" />
            <stop offset="0.35" stopColor="#d9538a" />
            <stop offset="0.72" stopColor="#f2a9c6" />
            <stop offset="1" stopColor="#fde8f0" />
          </radialGradient>
          <path
            ref={shape}
            id="nb-blossom-petal"
            d={DOT_PATH}
            fill="url(#nb-blossom-fill)"
            stroke="#ffffff"
            strokeWidth="0.6"
            strokeOpacity="0"
            strokeLinejoin="round"
          />
        </defs>
        <g ref={turn}>
          {PLACES.map((deg, i) => (
            <use
              key={deg}
              ref={(el) => {
                petals.current[i] = el
              }}
              href="#nb-blossom-petal"
              transform={`rotate(${deg})`}
              opacity={quiet ? 1 - i * 0.2 : dotOpacity(i, 0)}
            />
          ))}
          <g ref={centre} opacity="0">
            {STAMENS.map((deg) => (
              <g key={deg} transform={`rotate(${deg})`}>
                <line x1="0" y1="-1.5" x2="0" y2="-6.2" className="nb-blossom-filament" />
                <circle cx="0" cy="-6.6" r="1.05" className="nb-blossom-anther" />
              </g>
            ))}
            <circle r="2.2" className="nb-blossom-heart" />
          </g>
        </g>
      </svg>
      <p className="nb-blossom-caption">loading NikkiBase</p>
    </div>
  )
}
