import { useEffect, useState } from 'react'
import { Button, Input, Typography } from 'antd'
import { useStore } from '../store'
import { download, saveWardrobeFile } from './download'
import { selectionsFile, selectionsFileName, wardrobeCode } from './wardrobeFile'
import { CODE_TEXT } from './wardrobeText'

const COPIED_FOR = 2000

export function WardrobeExports() {
  const owned = useStore((s) => s.owned)
  const [copies, setCopies] = useState(0)
  const copied = copies > 0
  const [byHand, setByHand] = useState<{ ids: number[]; code: string } | null>(null)

  useEffect(() => {
    if (!copies) return
    const t = setTimeout(() => setCopies(0), COPIED_FOR)
    return () => clearTimeout(t)
  }, [copies])

  const saveForCalc = () => download(selectionsFile(owned), selectionsFileName(new Date()))

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

  return (
    <>
      <div className="nb-code-row">
        <button type="button" className="nb-why nb-code-link" onClick={() => saveWardrobeFile(owned)}>
          {CODE_TEXT.save}
        </button>
        <button type="button" className="nb-why nb-code-link" onClick={copy}>
          <span className="nb-code-swap" aria-live="polite">
            <span className={copied ? 'is-off' : undefined}>{CODE_TEXT.copy}</span>
            <span className={copied ? undefined : 'is-off'}>{CODE_TEXT.copied}</span>
          </span>
        </button>
        <button type="button" className="nb-why nb-code-link" onClick={saveForCalc}>
          {CODE_TEXT.calc}
        </button>
      </div>

      {byHand?.ids === owned && (
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
    </>
  )
}

export function PasteCode({ onLoad }: { onLoad: (text: string) => Promise<boolean> }) {
  const owned = useStore((s) => s.owned)
  const importing = useStore((s) => s.importing)
  const [paste, setPaste] = useState<{ owned: number[]; text: string } | null>(null)
  const pasting = paste?.owned === owned
  const pasted = pasting ? paste.text : ''

  const load = async () => {
    if (await onLoad(pasted)) setPaste(null)
  }

  return (
    <>
      <div className="nb-code-row">
        <button type="button" className="nb-why nb-code-link" aria-expanded={pasting} onClick={() => setPaste(pasting ? null : { owned, text: '' })}>
          {CODE_TEXT.paste}
        </button>
      </div>

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
