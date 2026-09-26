import { useMemo, useRef } from 'react'
import { Segmented, Select, Typography, type RefSelectProps } from 'antd'
import SkillControls from './SkillControls'
import { useStore } from './store'
import { usePhone } from './usePhone'
import {
  coverageLabel, groupOptions, hasVariants, matches, orderModes, stagePlaceholder, stagesInMode, type Difficulty, type Stage,
} from './stages'

type Props = {
  stages: Stage[]
  chosen: Stage | undefined
  mode: string
  onModeChange: (mode: string) => void
}

export default function StagePicker({ stages, chosen, mode, onModeChange }: Props) {
  const { stage, difficulty, busy, setDifficulty, set } = useStore()
  const modes = useMemo(() => orderModes(stages), [stages])
  const modeStages = useMemo(() => stagesInMode(stages, mode), [stages, mode])
  const stageOptions = useMemo(() => groupOptions(modeStages), [modeStages])
  const coverage = useMemo(() => coverageLabel(stages), [stages])
  const phone = usePhone()
  const box = useRef<HTMLDivElement>(null)
  const select = useRef<RefSelectProps>(null)
  const lift = (open: boolean) => {
    if (open && phone) box.current?.querySelector('.nb-stage-select')?.scrollIntoView({ block: 'start' })
  }

  return (
    <div className="nb-stage-picker" ref={box}>
      <Typography.Text type="secondary">Stage</Typography.Text>
      <div className="nb-stage-controls">
        <Segmented
          value={mode}
          onChange={(value) => {
            onModeChange(String(value))
            set({ stage: null, outfit: null, ideal: null })
          }}
          options={modes}
        />
        {mode === 'Story' && (
          <Segmented<Difficulty>
            value={difficulty}
            onChange={setDifficulty}
            options={['Maiden', 'Princess']}
          />
        )}
        <SkillControls chosen={chosen} />
        <Select
          showSearch
          aria-label="Stage"
          placeholder={stagePlaceholder(mode)}
          options={stageOptions}
          value={stage}
          onChange={(value) => {
            set({ stage: value })
            if (phone) setTimeout(() => select.current?.blur())
          }}
          filterOption={matches}
          notFoundContent={`No such stage. ${coverage}`}
          loading={busy}
          onOpenChange={lift}
          className="nb-stage-select"
          ref={select}
        />
      </div>
      {chosen && hasVariants(chosen) && (
        <Typography.Text type="secondary" className="nb-stage-variant">
          Maiden and Princess differ on this stage. Showing {difficulty}.
        </Typography.Text>
      )}
      {!chosen && coverage && (
        <Typography.Text type="secondary" className="nb-stage-coverage">
          {coverage}
        </Typography.Text>
      )}
    </div>
  )
}
