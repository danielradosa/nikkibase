import { useEffect, useMemo, useRef, useState } from 'react'
import { Button, Segmented, Select, Typography, type RefSelectProps } from 'antd'
import { LeftOutlined, RightOutlined } from '@ant-design/icons'
import Petals from '../components/Petals'
import SkillControls from './SkillControls'
import { useStore } from '../store'
import { usePhone } from '../hooks/usePhone'
import {
  coverageLabel, groupOptions, hasVariants, jumpOptions, matches, modeLabel, orderModes, pickerTree, placeOf, stagePlaceholder,
  stagesInMode, stepStage, type Difficulty, type PickerBranch, type Stage,
} from './stages'

type Props = {
  stages: Stage[]
  chosen: Stage | undefined
  mode: string
  onModeChange: (mode: string) => void
}

type Place = { branch: string; group: string }

const GROUP_LABEL: Record<string, string> = { Story: 'Chapter', Commission: 'Act', Dreamweaver: 'Character' }

const JUMP_EXAMPLE: Record<string, string> = { Story: '5-11', Commission: '3-7', Dreamweaver: 'Hidden String' }

function openPlace(tree: readonly PickerBranch[], saved: Place | undefined, chosen: string | null): Place | null {
  const valid = (p: Place | null | undefined) =>
    p && tree.some((b) => b.key === p.branch && b.groups.some((g) => g.key === p.group)) ? p : null
  return valid(saved) ?? (chosen ? placeOf(tree, chosen) : null) ?? (tree[0]?.groups[0] ? { branch: tree[0].key, group: tree[0].groups[0].key } : null)
}

export default function StagePicker({ stages, chosen, mode, onModeChange }: Props) {
  const { stage, difficulty, busy, setDifficulty, set } = useStore()
  const modes = useMemo(() => orderModes(stages), [stages])
  const tree = useMemo(() => pickerTree(stages, mode), [stages, mode])
  const coverage = useMemo(() => coverageLabel(stages), [stages])
  const phone = usePhone()
  const box = useRef<HTMLDivElement>(null)
  const select = useRef<RefSelectProps>(null)
  const [saved, setSaved] = useState<Record<string, Place>>({})
  const [search, setSearch] = useState('')

  useEffect(() => {
    const p = stage ? placeOf(tree, stage) : null
    if (p) setSaved((s) => (s[mode]?.group === p.group ? s : { ...s, [mode]: p }))
  }, [stage, tree, mode])

  const open = openPlace(tree, saved[mode], stage)
  const branch = tree.find((b) => b.key === open?.branch)
  const group = branch?.groups.find((g) => g.key === open?.group)
  const reveal = () => {
    if (phone) requestAnimationFrame(() => box.current?.querySelector('.nb-stage-step')?.scrollIntoView({ block: 'start', behavior: 'smooth' }))
  }
  const pick = (key: string | null) => {
    if (!key) return
    set({ stage: key })
    reveal()
  }
  const lift = (opened: boolean) => {
    if (opened && phone) box.current?.querySelector('.nb-stage-select, .nb-stage-jump')?.scrollIntoView({ block: 'start' })
  }
  const chip = (key: string, label: string, on: boolean, onClick: () => void) => (
    <button key={key} type="button" className={on ? 'nb-chip is-on' : 'nb-chip'} aria-pressed={on} onClick={onClick}>
      {label}
    </button>
  )

  return (
    <div className="nb-stage-picker" ref={box}>
      <Typography.Text type="secondary">Stage</Typography.Text>
      <div className="nb-chips nb-mode-chips" role="group" aria-label="Mode">
        {modes.map((m) =>
          chip(m, modeLabel(m), m === mode, () => {
            if (m === mode) return
            onModeChange(m)
            set({ stage: null, outfit: null, ideal: null })
            setSearch('')
          }),
        )}
      </div>
      <div className="nb-stage-controls">
        {mode === 'Story' && (
          <Segmented<Difficulty>
            value={difficulty}
            onChange={setDifficulty}
            options={['Maiden', 'Princess']}
          />
        )}
        <SkillControls chosen={chosen} mode={mode} />
        {tree.length === 0 && (
          <Select
            showSearch
            aria-label="Stage"
            placeholder={stagePlaceholder(mode)}
            options={groupOptions(stagesInMode(stages, mode))}
            value={stage}
            onChange={(value) => {
              set({ stage: value })
              if (phone) setTimeout(() => select.current?.blur())
            }}
            filterOption={matches}
            notFoundContent={`No such stage. ${coverage}`}
            suffixIcon={busy ? <Petals size={14} /> : undefined}
            onOpenChange={lift}
            className="nb-stage-select"
            ref={select}
          />
        )}
      </div>
      {tree.length > 0 && (
        <div className="nb-chip-rows">
          {tree.length > 1 && (
            <div role="group" aria-label="Volume">
              <span className="nb-chip-label">Volume</span>
              <div className="nb-chips">
                {tree.map((b) =>
                  chip(b.key, b.label, b.key === branch?.key, () =>
                    setSaved((s) => ({ ...s, [mode]: { branch: b.key, group: b.groups[0].key } })),
                  ),
                )}
              </div>
            </div>
          )}
          {branch && (
            <div role="group" aria-label={GROUP_LABEL[mode]}>
              <span className="nb-chip-label">{GROUP_LABEL[mode]}</span>
              <div className="nb-chips">
                {branch.groups.map((g) =>
                  chip(g.key, g.label, g.key === group?.key, () =>
                    setSaved((s) => ({ ...s, [mode]: { branch: branch.key, group: g.key } })),
                  ),
                )}
              </div>
            </div>
          )}
          {group && (
            <div role="group" aria-label="Stage">
              <span className="nb-chip-label">Stage</span>
              <div className="nb-chips">{group.leaves.map((l) => chip(l.key, l.label, l.key === stage, () => pick(l.key)))}</div>
            </div>
          )}
          <div className="nb-stage-step">
            {chosen && (
              <>
                <Button
                  shape="circle"
                  icon={<LeftOutlined />}
                  aria-label="Previous stage"
                  disabled={!stepStage(stages, mode, stage ?? '', -1)}
                  onClick={() => pick(stepStage(stages, mode, stage ?? '', -1))}
                />
                <span className="nb-stage-step-name">{chosen.name}</span>
                <Button
                  shape="circle"
                  icon={<RightOutlined />}
                  aria-label="Next stage"
                  disabled={!stepStage(stages, mode, stage ?? '', 1)}
                  onClick={() => pick(stepStage(stages, mode, stage ?? '', 1))}
                />
                {busy && <Petals size={14} />}
              </>
            )}
            <Select
              showSearch
              aria-label="Jump to a stage"
              placeholder={`Jump to a stage, e.g. ${JUMP_EXAMPLE[mode]}`}
              options={jumpOptions(stages, mode, search)}
              value={null}
              searchValue={search}
              onSearch={setSearch}
              filterOption={false}
              notFoundContent={search.trim() ? `No such stage. ${coverage}` : null}
              onChange={(value: string) => {
                setSearch('')
                if (phone) setTimeout(() => select.current?.blur())
                pick(value)
              }}
              onOpenChange={lift}
              className="nb-stage-jump"
              ref={select}
            />
          </div>
        </div>
      )}
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
