import { useLayoutEffect, useMemo } from 'react'
import { Alert, Empty } from 'antd'
import { useStore } from './store'
import { ATTRS, SLOTS, type Item } from './items'
import { comparisonRows, outfitText, type Place } from './comparison'
import { skillsLine } from './skills'
import { missingMessage, stageKey, type Stage } from './stages'
import { manyUnscored, unscored } from './wardrobeText'
import WardrobeImport from './WardrobeImport'
import StagePicker from './StagePicker'
import StageSummary from './StageSummary'
import OutfitScore from './OutfitScore'
import ComparisonTable from './ComparisonTable'

type Props = {
  stages: Stage[]
  items: Item[]
  tagNames: string[]
  places: Place[]
  onFile: (text: string) => Promise<void>
}

export default function BestOutfitTab({ stages, items, tagNames, places, onFile }: Props) {
  const { owned, decoded, stage, outfit, ideal, busy, difficulty, mode, tab, jump, set } = useStore()

  const names = useMemo(() => new Map(items.map((it) => [it.id, it.name])), [items])
  const chosen = stages.find((s) => stageKey(s) === stage)
  const rows = useMemo(() => comparisonRows(outfit, ideal, names, places, SLOTS), [outfit, ideal, names, places])
  const copyText = useMemo(
    () => outfitText(outfit, ideal, chosen ?? null, difficulty, names, places, SLOTS, skillsLine(outfit?.skills, ATTRS)),
    [outfit, ideal, chosen, difficulty, names, places],
  )

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

      {chosen && <StageSummary stage={chosen} tagNames={tagNames} names={names} />}

      {outfit ? (
        <>
          {outfit.missing?.length ? (
            <Alert
              type="warning"
              showIcon
              className="nb-alert"
              message={missingMessage(outfit.missing, names)}
              description="The outfit below can't pass the stage."
            />
          ) : null}
          <OutfitScore outfit={outfit} ideal={ideal} copyText={copyText} />
          <ComparisonTable rows={rows} busy={busy} names={names} />
        </>
      ) : (
        owned.length > 0 &&
        !busy && (
          <Empty description="Pick a stage to see your best outfit and the best possible one (using every item in the game)." />
        )
      )}
    </>
  )
}
