import { Alert, Typography, Upload } from 'antd'
import { InboxOutlined, UploadOutlined } from '@ant-design/icons'
import { useStore } from './store'
import { usePhone } from './usePhone'
import { DROP_HINT, NO_FILE, TAGLINE, dropText, loadedLabel, stripNotes } from './wardrobeText'

export default function WardrobeImport({ onFile }: { onFile: (text: string) => Promise<void> }) {
  const owned = useStore((s) => s.owned)
  const decoded = useStore((s) => s.decoded)
  const set = useStore((s) => s.set)
  const phone = usePhone()
  const loaded = owned.length > 0
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
        beforeUpload={async (file) => {
          await onFile(await file.text())
          return false
        }}
        className={loaded ? 'nb-import is-loaded' : 'nb-import'}
      >
        {loaded ? (
          <span className="nb-strip">
            <UploadOutlined className="nb-strip-icon" />
            <span className="nb-strip-text">
              <strong>{loadedLabel(owned.length)}</strong>
              {stripNotes(decoded).map((note) => (
                <span key={note} className="nb-strip-note">
                  · {note}
                </span>
              ))}
            </span>
            <span className="nb-strip-replace">Replace</span>
          </span>
        ) : (
          <>
            <p className="ant-upload-drag-icon">
              <InboxOutlined />
            </p>
            <p className="ant-upload-text">{dropText(phone)}</p>
            <p className="ant-upload-hint">{DROP_HINT}</p>
          </>
        )}
      </Upload.Dragger>

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
