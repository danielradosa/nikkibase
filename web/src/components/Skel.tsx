export default function Skel({ width, className }: { width?: number | string; className?: string }) {
  return (
    <span
      className={className ? `nb-skel ${className}` : 'nb-skel'}
      aria-hidden="true"
      style={width === undefined ? undefined : { width }}
    />
  )
}
