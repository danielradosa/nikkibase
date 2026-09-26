import { DOT, PLACES } from './petal'

export default function Petals({ size, className }: { size?: number; className?: string }) {
  return (
    <span
      className={className ? `nb-petals ${className}` : 'nb-petals'}
      aria-hidden="true"
      style={size ? { fontSize: size } : undefined}
    >
      <svg viewBox="-14 -14 28 28" width="1em" height="1em" focusable="false">
        {PLACES.map((deg, i) => (
          <circle key={deg} className={`nb-pt d${i}`} cy={-DOT.orbit} r={DOT.radius} transform={`rotate(${deg})`} />
        ))}
      </svg>
    </span>
  )
}
