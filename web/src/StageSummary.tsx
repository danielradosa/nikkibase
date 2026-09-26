import { WarningOutlined } from '@ant-design/icons'
import { Tag, Tooltip, Typography } from 'antd'
import { ATTRS } from './items'
import Skel from './Skel'
import { useStore } from './store'
import { caveatText, requirementLabel, resolveStage, rulesUnchecked, tagLabel, weightLabel, type Stage } from './stages'

type Props = { stage: Stage; tagNames: string[]; names: ReadonlyMap<number, string>; naming: boolean }

export default function StageSummary({ stage, tagNames, names, naming }: Props) {
  const difficulty = useStore((s) => s.difficulty)
  const scored = resolveStage(stage, difficulty)
  const required = scored.rules?.require ?? []

  return (
    <div className="nb-stage-summary" tabIndex={-1}>
      <Typography.Text type="secondary" className="nb-judged">
        Judged on
      </Typography.Text>
      {scored.weights.map((w, p) =>
        w > 0 ? (
          <Tag key={p} color="magenta">
            {weightLabel(ATTRS[scored.attrs[p]], w)}
          </Tag>
        ) : null,
      )}
      {Object.entries(scored.tags ?? {}).map(([id, award]) => (
        <Tooltip
          key={id}
          title={`Each ${tagNames[Number(id)] ?? 'tagged'} item you wear adds about ${award.toLocaleString('en-US')} points, scaled by the slot.`}
        >
          <Tag color="gold">{tagLabel(tagNames[Number(id)] ?? `tag ${id}`, award)}</Tag>
        </Tooltip>
      ))}
      {rulesUnchecked(scored.rules) && (
        <Typography.Paragraph type="secondary" className="nb-stage-caveat">
          <WarningOutlined className="nb-stage-caveat-icon" /> {caveatText(scored.rules?.styles ?? [])}
        </Typography.Paragraph>
      )}
      {required.length > 0 && (
        <Typography.Paragraph className="nb-requires">
          Requires: {naming ? <Skel width="50%" /> : requirementLabel(required, names)}
        </Typography.Paragraph>
      )}
    </div>
  )
}
