import { Button, Layout, Popconfirm, Typography } from 'antd'
import { useStore } from './store'
import { usePhone } from './usePhone'
import { TAGLINE } from './wardrobeText'

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
      {owned.length > 0 && (
        <Popconfirm
          placement="bottomRight"
          classNames={{ root: 'nb-forget-confirm' }}
          title="Remove your wardrobe from this device?"
          description={source === 'manual' ? 'Your ticked items will be lost.' : undefined}
          okText="Remove"
          okButtonProps={{ danger: true, size: 'middle' }}
          cancelText="Keep"
          cancelButtonProps={{ size: 'middle' }}
          onConfirm={onForget}
        >
          <Button type={phone ? 'text' : 'default'} className="nb-forget">
            {phone ? 'Forget' : 'Forget my wardrobe'}
          </Button>
        </Popconfirm>
      )}
    </Layout.Header>
  )
}
