import { Button } from 'antd'
import Petals from './Petals'

type Props = {
  text: string
  percent?: number | null
  onStop?: () => void
  stopping?: boolean
  className?: string
}

export default function WaitLine({ text, percent, onStop, stopping, className }: Props) {
  const known = typeof percent === 'number'
  return (
    <div className={className ? `nb-wait ${className}` : 'nb-wait'} role="status" aria-live="polite">
      <div className="nb-wait-line">
        <Petals size={14} />
        <span className="nb-wait-text">{text}</span>
        {onStop && (
          <Button type="link" className="nb-wait-stop" onClick={onStop} disabled={stopping}>
            {stopping ? 'Stopping' : 'Stop'}
          </Button>
        )}
      </div>
      <div className={known ? 'nb-wait-bar is-known' : 'nb-wait-bar'}>
        <span className="nb-wait-fill" style={known ? { width: `${percent}%` } : undefined} />
      </div>
    </div>
  )
}
