import { Button, Layout, Popconfirm, Typography } from 'antd'
import { useStore } from '../store'
import { usePhone } from '../hooks/usePhone'
import { FORGET_SAVE, TAGLINE } from '../wardrobe/wardrobeText'
import { saveWardrobeFile } from '../wardrobe/download'
import NotifyButton from '../notify/NotifyButton'

export default function Header({ onForget }: { onForget: () => Promise<void> }) {
  const owned = useStore((s) => s.owned)
  const source = useStore((s) => s.source)
  const phone = usePhone()

  return (
    <Layout.Header className="nb-header">
      <Typography.Title level={4} className="nb-wordmark">
        NikkiBase
      </Typography.Title>
      {!phone && <Typography.Text className="nb-tagline">{TAGLINE}</Typography.Text>}
      <NotifyButton />
      {owned.length > 0 && (
        <Popconfirm
          placement="bottomRight"
          classNames={{ root: 'nb-forget-confirm' }}
          title="Remove your wardrobe from this device?"
          description={
            <>
              {source === 'manual' && <div>Your ticked items will be lost.</div>}
              <button type="button" className="nb-why nb-code-link" onClick={() => saveWardrobeFile(owned)}>
                {FORGET_SAVE}
              </button>
            </>
          }
          okText="Remove"
          okButtonProps={{ danger: true, size: 'middle' }}
          cancelText="Keep"
          cancelButtonProps={{ size: 'middle' }}
          onConfirm={onForget}
        >
          <Button type={phone ? 'text' : 'default'} className="nb-forget">
            {phone ? 'Forget wardrobe' : 'Forget my wardrobe'}
          </Button>
        </Popconfirm>
      )}
    </Layout.Header>
  )
}
