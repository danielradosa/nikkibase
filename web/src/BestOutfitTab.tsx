import { useLayoutEffect, useMemo } from 'react'
import { Alert, Empty } from 'antd'
import { useStore } from './store'
import { ATTRS, SLOTS, type Item } from './items'
import { FINDING, NAMES_WAIT, comparisonRows, outfitText, resultView, type Place } from './comparison'
import { skillsLine } from './skills'
import { missingMessage, stageKey, type Stage } from './stages'
import { manyUnscored, unscored } from './wardrobeText'
import WardrobeImport from './WardrobeImport'
import StagePicker from './StagePicker'
import StageSummary from './StageSummary'
import OutfitScore from './OutfitScore'
import ScoreWait from './ScoreWait'
import ComparisonTable from './ComparisonTable'
import WaitLine from './WaitLine'
import Skel from './Skel'

type Props = {
  stages: Stage[]
  items: Item[] | null
  itemsFailed: boolean
  tagNames: string[]
  places: Place[]
  onFile: (text: string) => Promise<void>
}

export default function BestOutfitTab({ stages, items, itemsFailed, tagNames, places, onFile }: Props) {
  const { owned, decoded, stage, outfit, ideal, busy, difficulty, mode, tab, jump, set } = useStore()

  const names = useMemo(() => new Map((items ?? []).map((it) => [it.id, it.name])), [items])
  const naming = items === null && !itemsFailed
  const chosen = stages.find((s) => stageKey(s) === stage)
  const rows = useMemo(() => comparisonRows(outfit, ideal, names, places, SLOTS), [outfit, ideal, names, places])
  const copyText = useMemo(
    () => outfitText(outfit, ideal, chosen ?? null, difficulty, names, places, SLOTS, skillsLine(outfit?.skills, ATTRS)),
    [outfit, ideal, chosen, difficulty, names, places],
  )
  const view = resultView({ owned: owned.length, chosen: !!chosen, outfit: !!outfit, busy })

  useLayoutEffect(() => {
    if (!jump) return
    if (tab !== 'outfit') {
      set({ jump: false })
      return
    }
    const frame = requestAnimationFrame(() => {
      const summary = document.querySelector<HTMLElement>('.nb-stage-summary')
      summary?.scrollIntoView({ block: 'start' })
      if (busy) return
      summary?.focus({ preventScroll: true })
      set({ jump: false })
    })
    return () => cancelAnimationFrame(frame)
  }, [jump, tab, busy, set])

  return (
    <>
      <WardrobeImport onFile={onFile} />

      {owned.length > 0 && (
        <div className="nb-wardrobe-row">
          <StagePicker stages={stages} chosen={chosen} mode={mode} onModeChange={(next) => set({ mode: next })} />
        </div>
      )}

      {manyUnscored(decoded) && (
        <Alert
          type="warning"
          showIcon
          className="nb-alert"
          message={`${unscored(decoded).toLocaleString('en-US')} of your items have no published stats`}
          description="They cannot be scored, so they never appear below. Your real best may be slightly better."
        />
      )}

      {chosen && <StageSummary stage={chosen} tagNames={tagNames} names={names} naming={naming} />}

      {view === 'first' && <ScoreWait />}
      {view === 'first' && <WaitLine text={FINDING} className="nb-outfit-wait" />}

      {outfit ? (
        <div className="nb-result-box">
          {view === 'stale' && <WaitLine text={FINDING} className="nb-outfit-wait is-over" />}
          <div className={view === 'stale' ? 'nb-result is-stale' : 'nb-result'} aria-busy={view === 'stale' || undefined}>
            {outfit.missing?.length ? (
              <Alert
                type="warning"
                showIcon
                className="nb-alert"
                message={naming ? <Skel width="60%" /> : missingMessage(outfit.missing, names)}
                description="The outfit below can't pass the stage."
              />
            ) : null}
            <OutfitScore outfit={outfit} ideal={ideal} copyText={copyText} busy={busy} naming={naming} />
            {naming && <WaitLine text={NAMES_WAIT} className="nb-names-wait" />}
            <ComparisonTable rows={rows} names={names} naming={naming} />
          </div>
        </div>
      ) : (
        owned.length > 0 &&
        !busy && (
          <Empty description="Pick a stage to see your best outfit and the best possible one (using every item in the game)." />
        )
      )}
    </>
  )
}
