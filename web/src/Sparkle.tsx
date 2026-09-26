import { useEffect, useRef, useState } from 'react'

const GLYPHS = ['✦', '✧', '⋆', '✦', '·']

type Particle = { id: number; glyph: string; dx: string; dy: string; rot: string; scale: number }

function burst(seed: number): Particle[] {
  return Array.from({ length: 10 }, (_, i) => {
    const angle = (i / 10) * Math.PI * 2 + (seed % 7) * 0.31
    const reach = 26 + (i % 3) * 13
    return {
      id: seed * 100 + i,
      glyph: GLYPHS[i % GLYPHS.length],
      dx: `${Math.cos(angle) * reach}px`,
      dy: `${Math.sin(angle) * reach * 0.72}px`,
      rot: `${(i % 2 ? 1 : -1) * (90 + i * 24)}deg`,
      scale: 0.5 + (i % 4) * 0.22,
    }
  })
}

export default function Sparkle({
  trigger,
  children,
}: {
  trigger: number
  children: React.ReactNode
}) {
  const [particles, setParticles] = useState<Particle[]>([])
  const seen = useRef(trigger)

  useEffect(() => {
    if (trigger === seen.current) return
    seen.current = trigger
    if (window.matchMedia?.('(prefers-reduced-motion: reduce)').matches) return
    setParticles(burst(trigger))
    const done = window.setTimeout(() => setParticles([]), 340)
    return () => window.clearTimeout(done)
  }, [trigger])

  return (
    <span className="nb-sparkle-host">
      {children}
      {particles.map((p) => (
        <span
          key={p.id}
          className="nb-spark"
          aria-hidden="true"
          style={
            {
              '--dx': p.dx,
              '--dy': p.dy,
              '--rot': p.rot,
              '--scale': p.scale,
            } as React.CSSProperties
          }
        >
          {p.glyph}
        </span>
      ))}
    </span>
  )
}
