import { CopyOutlined } from '@ant-design/icons'
import { Button, Statistic, Typography } from 'antd'
import Skel from './Skel'

export default function ScoreWait() {
  return (
    <div className="nb-score-row nb-score-wait" aria-hidden="true">
      <Statistic title="Your best" value={0} valueRender={() => <Skel className="nb-skel-score" />} className="nb-score" />
      <Statistic title="Best possible" value={0} valueRender={() => <Skel className="nb-skel-score" />} className="nb-ideal" />
      <div className="nb-reach">
        <Typography.Text type="secondary">Of best possible</Typography.Text>
        <Skel className="nb-skel-reach" />
      </div>
      <Button icon={<CopyOutlined />} disabled>
        Copy outfit
      </Button>
    </div>
  )
}
