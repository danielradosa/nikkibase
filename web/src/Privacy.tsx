import { useEffect, useState } from 'react'
import { Modal, Typography } from 'antd'

export default function Privacy({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [text, setText] = useState<string | null>(null)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    if (!open || text !== null) return
    fetch('/privacy.txt')
      .then((r) => (r.ok ? r.text() : Promise.reject(new Error(String(r.status)))))
      .then(setText)
      .catch(() => setFailed(true))
  }, [open, text])

  const blocks = (text ?? '').split(/\n{2,}/).map((b) => b.trim()).filter(Boolean)
  return (
    <Modal open={open} onCancel={onClose} footer={null} title="Privacy" width={720}>
      {failed && (
        <Typography.Paragraph>
          The policy could not be loaded here; it is also in the repository as{' '}
          <a href="https://github.com/danielradosa/nikkibase/blob/main/PRIVACY.md" target="_blank" rel="noreferrer">
            PRIVACY.md
          </a>
          .
        </Typography.Paragraph>
      )}
      {blocks.map((b, i) => {
        if (b.startsWith('# ')) return null
        if (b.startsWith('## ')) return <Typography.Title key={i} level={5}>{b.slice(3)}</Typography.Title>
        if (b.startsWith('- ')) {
          return (
            <ul key={i} className="nb-privacy-list">
              {b.split(/\n- /).map((item, j) => (
                <li key={j}>{item.replace(/^- /, '').replace(/\n\s*/g, ' ')}</li>
              ))}
            </ul>
          )
        }
        return <Typography.Paragraph key={i}>{b.replace(/\n/g, ' ')}</Typography.Paragraph>
      })}
    </Modal>
  )
}
