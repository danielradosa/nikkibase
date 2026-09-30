import { useEffect, useState } from 'react'
import { Button, Input, Typography } from 'antd'
import { useStore } from '../store'
import { SITE_HOST } from '../outfit/skills'
import { fileName, wardrobeCode, wardrobeFile } from './wardrobeFile'
import { CODE_TEXT } from './wardrobeText'

const COPIED_FOR = 2000
const REVOKE_AFTER = 40000

function download(text: string, name: string) {
  const url = URL.createObjectURL(new Blob([text], { type: 'text/plain;charset=utf-8' }))
  const link = document.createElement('a')
  link.href = url
  link.download = name
  link.hidden = true
  document.body.append(link)
  link.click()
  link.remove()
  setTimeout(() => URL.revokeObjectURL(url), REVOKE_AFTER)
}

export default function WardrobeCode({ onLoad }: { onLoad: (text: string) => Promise<boolean> }) {
  const owned = useStore((s) => s.owned)
  const importing = useStore((s) => s.importing)
  const loaded = owned.length > 0
  const [copies, setCopies] = useState(0)
  const copied = copies > 0
  const [byHand, setByHand] = useState<{ ids: number[]; code: string } | null>(null)
  const [paste, setPaste] = useState<{ owned: number[]; text: string } | null>(null)
  const pasting = paste?.owned === owned
  const pasted = pasting ? paste.text : ''

  useEffect(() => {
    if (!copies) return
    const t = setTimeout(() => setCopies(0), COPIED_FOR)
    return () => clearTimeout(t)
  }, [copies])

  const save = () => {
    const now = new Date()
    download(wardrobeFile(owned, now, SITE_HOST), fileName(now))
  }

  const copy = () => {
    const ids = owned
    const code = wardrobeCode(ids)
    let writing: Promise<void>
    try {
      writing = navigator.clipboard.writeText(code)
    } catch (e) {
      writing = Promise.reject(e)
    }
    writing.then(
      () => {
        setByHand(null)
        setCopies((n) => n + 1)
      },
      () => {
        setCopies(0)
        setByHand({ ids, code })
      },
    )
  }

  const load = async () => {
    if (await onLoad(pasted)) setPaste(null)
  }

  return (
    <>
      <div className={`nb-code-row${loaded ? '' : ' is-empty'}`}>
        {loaded && (
          <>
            <button type="button" className="nb-why nb-code-link" onClick={save}>
              {CODE_TEXT.save}
            </button>
            <button type="button" className="nb-why nb-code-link" onClick={copy}>
              <span className="nb-code-swap" aria-live="polite">
                <span className={copied ? 'is-off' : undefined}>{CODE_TEXT.copy}</span>
                <span className={copied ? undefined : 'is-off'}>{CODE_TEXT.copied}</span>
              </span>
            </button>
          </>
        )}
        <button type="button" className="nb-why nb-code-link" aria-expanded={pasting} onClick={() => setPaste(pasting ? null : { owned, text: '' })}>
          {CODE_TEXT.paste}
        </button>
      </div>

      {loaded && byHand?.ids === owned && (
        <div className="nb-code-box">
          <Typography.Text type="secondary" className="nb-code-note">
            {CODE_TEXT.blocked}
          </Typography.Text>
          <Input.TextArea
            readOnly
            autoFocus
            value={byHand.code}
            onFocus={(e) => e.currentTarget.select()}
            autoSize={{ minRows: 2, maxRows: 4 }}
            aria-label={CODE_TEXT.field}
            spellCheck={false}
          />
        </div>
      )}

      {pasting && (
        <div className="nb-code-box">
          <Input.TextArea
            autoFocus
            value={pasted}
            onChange={(e) => setPaste({ owned, text: e.target.value })}
            autoSize={{ minRows: 2, maxRows: 4 }}
            placeholder={CODE_TEXT.placeholder}
            aria-label={CODE_TEXT.field}
            spellCheck={false}
            autoComplete="off"
            autoCorrect="off"
            autoCapitalize="off"
          />
          <Button type="primary" onClick={load} disabled={!pasted.trim()} loading={importing}>
            {CODE_TEXT.load}
          </Button>
        </div>
      )}
    </>
  )
}
