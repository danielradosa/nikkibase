import { useState } from 'react'
import { Alert, Typography, Upload } from 'antd'
import { UploadOutlined } from '@ant-design/icons'
import Petals from '../components/Petals'
import { useStore } from '../store'
import { useLate } from '../hooks/useLate'
import { usePhone } from '../hooks/usePhone'
import WaitLine from '../components/WaitLine'
import { PasteCode, WardrobeExports } from './WardrobeCode'
import {
  ENGINE_DOWN,
  ENGINE_WAIT,
  IMPORT_CARDS,
  KEEP_CURRENT,
  NO_FILE,
  TAGLINE,
  dropText,
  importingText,
  loadedLabel,
  sourceLine,
  stripNotes,
} from './wardrobeText'

export default function WardrobeImport({ onFile }: { onFile: (text: string, pasted?: boolean) => Promise<boolean> }) {
  const owned = useStore((s) => s.owned)
  const source = useStore((s) => s.source)
  const savedAt = useStore((s) => s.savedAt)
  const decoded = useStore((s) => s.decoded)
  const importing = useStore((s) => s.importing)
  const engine = useStore((s) => s.engine)
  const saveFailing = useStore((s) => s.saveFailing)
  const set = useStore((s) => s.set)
  const phone = usePhone()
  const [replacing, setReplacing] = useState<number[] | null>(null)
  const loaded = owned.length > 0
  const choosing = !loaded || replacing === owned
  const slow = useLate(importing)
  const reading = importingText(engine === 'ready')
  const openItems = () => {
    set({ tab: 'items' })
    if (phone) window.scrollTo({ top: 0 })
  }
  const take = async (file: File) => {
    if (useStore.getState().importing) return false
    await onFile(await file.text())
    return false
  }

  return (
    <>
      {!choosing ? (
        <>
          <Upload.Dragger
            accept="*"
            maxCount={1}
            showUploadList={false}
            openFileDialogOnClick={false}
            beforeUpload={take}
            className={`nb-import is-loaded${slow ? ' is-busy' : ''}`}
          >
            <span className="nb-strip">
              {slow ? <Petals size={14} className="nb-strip-icon" /> : <UploadOutlined className="nb-strip-icon" />}
              {slow ? (
                <span className="nb-strip-text">{reading}</span>
              ) : (
                <span className="nb-strip-text">
                  <strong>{loadedLabel(owned.length)}</strong>
                  {[...sourceLine(source, savedAt), ...stripNotes(decoded)].map((note) => (
                    <span key={note} className="nb-strip-note">
                      · {note}
                    </span>
                  ))}
                </span>
              )}
              {!slow && (
                <button type="button" className="nb-strip-replace" onClick={() => setReplacing(owned)}>
                  Replace
                </button>
              )}
            </span>
          </Upload.Dragger>
          <WardrobeExports />
        </>
      ) : slow ? (
        <Upload.Dragger accept="*" maxCount={1} showUploadList={false} openFileDialogOnClick={false} className="nb-import is-busy">
          <p className="ant-upload-drag-icon">
            <Petals size={32} />
          </p>
          <p className="ant-upload-text">{reading}</p>
        </Upload.Dragger>
      ) : (
        <>
          <div className="nb-import-cards">
            {IMPORT_CARDS.map((card) => (
              <Upload.Dragger
                key={card.key}
                accept="*"
                maxCount={1}
                showUploadList={false}
                openFileDialogOnClick={!importing}
                beforeUpload={take}
                className="nb-import nb-import-card"
              >
                <p className="nb-import-title">{card.title}</p>
                <p className="ant-upload-text">{card.file}</p>
                <p className="nb-import-drop">
                  <UploadOutlined /> {dropText(phone)}
                </p>
                <p className="ant-upload-hint">{card.hint}</p>
              </Upload.Dragger>
            ))}
            <div className="nb-import-paste">
              <PasteCode onLoad={(text) => onFile(text, true)} />
            </div>
          </div>
          {loaded && (
            <div className="nb-code-row">
              <button type="button" className="nb-why nb-code-link" onClick={() => setReplacing(null)}>
                {KEEP_CURRENT}
              </button>
            </div>
          )}
        </>
      )}

      {engine === 'loading' && !loaded && !importing && <WaitLine text={ENGINE_WAIT} className="nb-engine-wait" />}
      {engine === 'failed' && !loaded && (
        <Typography.Paragraph type="secondary" className="nb-engine-down">
          {ENGINE_DOWN.import}
        </Typography.Paragraph>
      )}

      {!loaded && phone && <Typography.Paragraph className="nb-tagline nb-tagline-under">{TAGLINE}</Typography.Paragraph>}

      {!loaded && (
        <Alert
          type="info"
          showIcon
          className="nb-import-hint"
          message={
            <>
              {NO_FILE.before}
              <button type="button" className="nb-why" onClick={openItems}>
                {NO_FILE.link}
              </button>
              {NO_FILE.after}
            </>
          }
          description={saveFailing ? NO_FILE.unsaved : NO_FILE.saved}
        />
      )}
    </>
  )
}
