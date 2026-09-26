import { Fragment } from 'react'
import { CopyOutlined } from '@ant-design/icons'
import { Button, Progress, Statistic, Typography, message } from 'antd'
import Sparkle from './Sparkle'
import { closeCall, closeCallText } from './comparison'
import type { Outfit } from './engine'
import { ATTRS } from './items'
import { MAX_LEVELS, bestElsewhere, levelsShort, placementPhrases } from './skills'
import { useStore } from './store'
import type { Ideal } from './stages'

type Props = { outfit: Outfit; ideal: Ideal | null; copyText: string; busy: boolean }

export default function OutfitScore({ outfit, ideal, copyText, busy }: Props) {
  const reachable = ideal && ideal.score > 0 ? outfit.score / ideal.score : null
  const close = closeCall(outfit)
  const skills = outfit.skills
  const levels = skills?.levels ?? MAX_LEVELS
  const phrases = skills ? placementPhrases(skills, levels, ATTRS) : []
  const elsewhere = bestElsewhere(outfit, ideal, ATTRS)
  const switchedOn = useStore((s) => s.skills.on)
  const [toast, toastHolder] = message.useMessage()
  const copy = () =>
    Promise.resolve()
      .then(() => navigator.clipboard.writeText(copyText))
      .then(
        () => toast.success('Outfit copied'),
        () => toast.error("Couldn't copy the outfit."),
      )

  return (
    <>
      <div className="nb-score-row">
        <Sparkle trigger={outfit.score}>
          <Statistic
            title="Your best"
            value={outfit.score}
            suffix={
              <Typography.Text type="secondary" className="nb-count">
                {outfit.items.length} items
              </Typography.Text>
            }
            className="nb-score nb-score-pop"
          />
        </Sparkle>
        <Statistic
          className="nb-ideal"
          title="Best possible"
          value={ideal?.score ?? 0}
          suffix={
            ideal ? (
              <Typography.Text type="secondary" className="nb-count">
                {ideal.items.length} items
              </Typography.Text>
            ) : undefined
          }
        />
        {reachable !== null && (
          <div className="nb-reach">
            <Typography.Text type="secondary">Of best possible</Typography.Text>
            <Progress percent={Math.min(100, Math.round(reachable * 100))} status="normal" />
          </div>
        )}
        <Button icon={<CopyOutlined />} onClick={copy} disabled={busy}>
          Copy outfit
        </Button>
      </div>
      {toastHolder}

      {close !== null && (
        <Typography.Paragraph type="secondary" className="nb-close-call" ellipsis={{ rows: 1, expandable: true, symbol: 'more' }}>
          {closeCallText(close)}
        </Typography.Paragraph>
      )}

      <Typography.Paragraph type="secondary" className="nb-skills-note" ellipsis={{ rows: 1, expandable: true, symbol: 'more' }}>
        {skills && phrases.length ? (
          <>
            With{' '}
            {phrases.map((phrase, i) => (
              <Fragment key={phrase}>
                {i > 0 && ' and '}
                <strong>{phrase}</strong>
              </Fragment>
            ))}{' '}
            ({levelsShort(levels)}). Skills don&apos;t boost tag or spirit bonuses.{elsewhere}
          </>
        ) : switchedOn ? (
          <>
            <strong>No skills counted</strong>: Smile and Charming are both off.
          </>
        ) : (
          <>
            <strong>No skills counted</strong>. Smile and Charming usually add about 25%. Turn Skills on above to
            include them.
          </>
        )}
      </Typography.Paragraph>
    </>
  )
}
