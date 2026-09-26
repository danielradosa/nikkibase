import { Alert, Typography, Upload } from 'antd'
import { InboxOutlined, UploadOutlined } from '@ant-design/icons'
import Petals from './Petals'
import { useStore } from './store'
import { useLate } from './useLate'
import { usePhone } from './usePhone'
import WaitLine from './WaitLine'
import { DROP_HINT, ENGINE_WAIT, NO_FILE, TAGLINE, dropText, importingText, loadedLabel, stripNotes } from './wardrobeText'

export default function WardrobeImport({ onFile }: { onFile: (text: string) => Promise<void> }) {
  const owned = useStore((s) => s.owned)
  const decoded = useStore((s) => s.decoded)
  const importing = useStore((s) => s.importing)
  const engine = useStore((s) => s.engine)
  const set = useStore((s) => s.set)
  const phone = usePhone()
  const loaded = owned.length > 0
  const slow = useLate(importing)
  const reading = importingText(engine === 'ready')
  const openItems = () => {
    set({ tab: 'items' })
    if (phone) window.scrollTo({ top: 0 })
  }

  return (
    <>
      <Upload.Dragger
        accept="*"
        maxCount={1}
        showUploadList={false}
        openFileDialogOnClick={!importing}
        beforeUpload={async (file) => {
          if (useStore.getState().importing) return false
          await onFile(await file.text())
          return false
        }}
        className={`nb-import${loaded ? ' is-loaded' : ''}${slow ? ' is-busy' : ''}`}
      >
        {loaded ? (
          <span className="nb-strip">
            {slow ? <Petals size={14} className="nb-strip-icon" /> : <UploadOutlined className="nb-strip-icon" />}
            {slow ? (
              <span className="nb-strip-text">{reading}</span>
            ) : (
              <span className="nb-strip-text">
                <strong>{loadedLabel(owned.length)}</strong>
                {stripNotes(decoded).map((note) => (
                  <span key={note} className="nb-strip-note">
                    · {note}
                  </span>
                ))}
              </span>
            )}
            {!slow && <span className="nb-strip-replace">Replace</span>}
          </span>
        ) : (
          <>
            <p className="ant-upload-drag-icon">{slow ? <Petals size={32} /> : <InboxOutlined />}</p>
            <p className="ant-upload-text">{slow ? reading : dropText(phone)}</p>
            <p className="ant-upload-hint">{DROP_HINT}</p>
          </>
        )}
      </Upload.Dragger>

      {engine === 'loading' && !loaded && !importing && <WaitLine text={ENGINE_WAIT} className="nb-engine-wait" />}

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
          description={NO_FILE.saved}
        />
      )}
    </>
  )
}
