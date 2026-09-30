import { useEffect, useState } from 'react'
import { Button, Drawer, Popover, Switch, Typography } from 'antd'
import { BellFilled, BellOutlined } from '@ant-design/icons'
import { usePhone } from '../hooks/usePhone'
import { useNotify, type Notify } from './useNotify'
import { NOTIFY_INTRO, NOTIFY_PRIVACY, TOPICS, TOPIC_LABELS, statusText, withTopic } from './notify'

function Panel({ n }: { n: Notify }) {
  const trouble = n.status.kind === 'failed' || n.status.kind === 'denied'
  return (
    <div className="nb-notify-panel">
      <Typography.Paragraph className="nb-notify-intro">{NOTIFY_INTRO}</Typography.Paragraph>
      <div className="nb-notify-topics">
        {TOPICS.map((topic) => (
          <label key={topic} className="nb-toggle">
            <Switch
              checked={n.topics.includes(topic)}
              disabled={n.status.kind === 'install' || (n.pending !== null && n.pending !== topic)}
              loading={n.pending === topic}
              onChange={(on) => n.apply(withTopic(n.topics, topic, on), topic)}
            />
            <span>{TOPIC_LABELS[topic]}</span>
          </label>
        ))}
      </div>
      <Typography.Paragraph className="nb-notify-status" type={trouble ? 'danger' : undefined} role="status">
        {statusText(n.status)}
      </Typography.Paragraph>
      <Typography.Paragraph type="secondary" className="nb-notify-privacy">
        {NOTIFY_PRIVACY}
      </Typography.Paragraph>
    </div>
  )
}

export default function NotifyButton() {
  const n = useNotify()
  const phone = usePhone()
  const [sheet, setSheet] = useState(false)
  useEffect(() => {
    if (!phone) setSheet(false)
  }, [phone])

  if (!n.available) return null
  const on = n.topics.length > 0
  const bell = (
    <Button
      type={phone ? 'text' : 'default'}
      className="nb-notify"
      aria-label={on ? 'Notifications, on' : 'Notifications'}
      icon={on ? <BellFilled /> : <BellOutlined />}
      onClick={phone ? () => setSheet(true) : undefined}
    />
  )

  if (!phone) {
    return (
      <Popover trigger="click" placement="bottomRight" title="Notifications" rootClassName="nb-notify-pop" content={<Panel n={n} />}>
        {bell}
      </Popover>
    )
  }
  return (
    <>
      {bell}
      <Drawer
        open={sheet}
        onClose={() => setSheet(false)}
        placement="bottom"
        height="auto"
        title="Notifications"
        rootClassName="nb-notify-sheet"
        styles={{
          content: { maxHeight: '85vh' },
          body: { padding: '16px 16px max(16px, env(safe-area-inset-bottom))' },
        }}
      >
        <Panel n={n} />
        <Button type="primary" block className="nb-sheet-done" onClick={() => setSheet(false)}>
          Done
        </Button>
      </Drawer>
    </>
  )
}
