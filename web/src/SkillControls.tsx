import { useEffect, useState } from 'react'
import { SlidersOutlined } from '@ant-design/icons'
import { Button, Drawer, Popover, Segmented, Select, Typography } from 'antd'
import { ATTRS } from './items'
import { CHARMING_PERCENT, SMILE_PERCENT, levelsLabel, pickerDefault, skillsOff, type SkillLevels } from './skills'
import { useStore } from './store'
import { placementKey, resolveStage, type Stage } from './stages'
import { usePhone } from './usePhone'

type Props = { chosen: Stage | undefined }

type Container = (node: HTMLElement) => HTMLElement

const smileOptions = SMILE_PERCENT.map((pct, level) => ({
  value: level,
  label: level ? `Level ${level} · +${pct}%` : 'Not used',
}))

const charmingOptions = CHARMING_PERCENT.map((pct, level) => ({
  value: level,
  label: level ? `Level ${level} · +${pct}%` : 'Locked or not used',
}))

const inPanel = (node: HTMLElement) => node.closest<HTMLElement>('.nb-skill-panel') ?? document.body

function Placement({ chosen, levels, container }: { chosen: Stage | undefined; levels: SkillLevels; container?: Container }) {
  const { difficulty, outfit, placements, setPlacement } = useStore()
  if (!chosen) {
    return (
      <Typography.Text type="secondary" className="nb-skill-wide">
        Pick a stage to choose where they go.
      </Typography.Text>
    )
  }
  const stage = resolveStage(chosen, difficulty)
  const key = placementKey(chosen, difficulty)
  const options = stage.attrs
    .map((code, p) => ({ value: code, label: ATTRS[code], weight: stage.weights[p] }))
    .filter((o) => o.weight > 0)
  const current = pickerDefault(
    options.map((o) => o.value),
    levels,
    placements[key],
    outfit?.skills,
  )
  const pick = (first: number, second: number) => setPlacement(key, { charmSmile: first, smile: second })
  const firstLabel = levels.smile === 0 ? 'Charming on' : levels.charming === 0 ? 'Smile on' : 'Charming + Smile on'
  const secondLabel = levels.charming === 0 ? 'Second Smile on' : 'Smile on'

  return (
    <>
      <Typography.Text>{firstLabel}</Typography.Text>
      <Select
        aria-label={firstLabel}
        value={current.charmSmile >= 0 ? current.charmSmile : undefined}
        options={options}
        onChange={(v) => pick(v, v === current.smile ? current.charmSmile : current.smile)}
        getPopupContainer={container}
        virtual={false}
      />
      {levels.smile > 0 && options.length > 1 && (
        <>
          <Typography.Text>{secondLabel}</Typography.Text>
          <Select
            aria-label={secondLabel}
            value={current.smile >= 0 ? current.smile : undefined}
            options={options}
            onChange={(v) => pick(v === current.charmSmile ? current.smile : current.charmSmile, v)}
            getPopupContainer={container}
            virtual={false}
          />
        </>
      )}
    </>
  )
}

function WhyNote() {
  const [open, setOpen] = useState(false)
  return (
    <Typography.Text type="secondary" className="nb-skill-wide nb-skill-foot">
      Only Smile and Charming raise your score.{' '}
      {open ? (
        'Other skills act on your opponent. In the Arena skills fire by themselves, so Arena scores are a best case.'
      ) : (
        <button type="button" className="nb-why" onClick={() => setOpen(true)}>
          Why?
        </button>
      )}
    </Typography.Text>
  )
}

function Panel({ chosen, container }: { chosen: Stage | undefined; container?: Container }) {
  const { skills, setSkills } = useStore()
  const setLevels = (levels: SkillLevels) => setSkills({ ...skills, levels })

  return (
    <div className="nb-skill-panel">
      <Typography.Text>Smile</Typography.Text>
      <Select
        aria-label="Smile level"
        value={skills.levels.smile}
        options={smileOptions}
        onChange={(smile) => setLevels({ ...skills.levels, smile })}
        getPopupContainer={container}
        virtual={false}
      />
      <Typography.Text>Charming</Typography.Text>
      <Select
        aria-label="Charming level"
        value={skills.levels.charming}
        options={charmingOptions}
        onChange={(charming) => setLevels({ ...skills.levels, charming })}
        getPopupContainer={container}
        virtual={false}
      />
      {skillsOff(skills) ? (
        <Typography.Text type="secondary" className="nb-skill-wide">
          Both off: no skills counted.
        </Typography.Text>
      ) : (
        <>
          <Typography.Text>Use on</Typography.Text>
          <Segmented<'auto' | 'choose'>
            value={skills.manual ? 'choose' : 'auto'}
            onChange={(v) => setSkills({ ...skills, manual: v === 'choose' })}
            options={[
              { label: 'Best fit', value: 'auto' },
              { label: "I'll pick", value: 'choose' },
            ]}
          />
          {skills.manual && <Placement chosen={chosen} levels={skills.levels} container={container} />}
        </>
      )}
      <WhyNote />
    </div>
  )
}

export default function SkillControls({ chosen }: Props) {
  const { skills, setSkills } = useStore()
  const phone = usePhone()
  const [sheet, setSheet] = useState(false)
  useEffect(() => {
    if (!phone) setSheet(false)
  }, [phone])
  const levels = (
    <Button icon={<SlidersOutlined />} className="nb-skill-levels" onClick={phone ? () => setSheet(true) : undefined}>
      {levelsLabel(skills.levels)}
    </Button>
  )

  return (
    <>
      <Segmented<'off' | 'on'>
        value={skills.on ? 'on' : 'off'}
        onChange={(v) => setSkills({ ...skills, on: v === 'on' })}
        options={[
          { label: 'Skills off', value: 'off' },
          { label: 'Skills on', value: 'on' },
        ]}
      />
      {skills.on && !phone && (
        <Popover trigger="click" placement="bottomLeft" title="Your skills" content={<Panel chosen={chosen} container={inPanel} />}>
          {levels}
        </Popover>
      )}
      {skills.on && phone && (
        <>
          {levels}
          <Drawer
            open={sheet}
            onClose={() => setSheet(false)}
            placement="bottom"
            height="auto"
            title="Your skills"
            rootClassName="nb-skill-sheet"
            styles={{
              content: { maxHeight: '85vh' },
              body: { padding: '16px 16px max(16px, env(safe-area-inset-bottom))' },
            }}
          >
            <Panel chosen={chosen} />
            <Button type="primary" block className="nb-sheet-done" onClick={() => setSheet(false)}>
              Done
            </Button>
          </Drawer>
        </>
      )}
    </>
  )
}
