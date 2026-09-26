import { useLayoutEffect, useMemo, useRef, useState, type Dispatch, type SetStateAction } from 'react'
import { flushSync } from 'react-dom'
import { DownOutlined, RightOutlined } from '@ant-design/icons'
import { Alert, Button, Empty, Grid, Segmented, Select, Space, Switch, Table, Typography } from 'antd'
import type { Place } from './comparison'
import type { WorthRow } from './engine'
import { ANY_SLOT, choiceLabel, placeName, slotChoice, slotOptions, type Item } from './items'
import Petals from './Petals'
import Skel from './Skel'
import { worthSettings, worthSkillsText } from './skills'
import {
  orderModes, resolveStage, rulesUnchecked, stageKey, variantStages, worthSkip, worthVersions,
  type Difficulty, type Stage,
} from './stages'
import { useStore } from './store'
import { usePhone } from './usePhone'
import { runner, useAcquire, useIdeals, useWorthRanking, useWorthRun, type Acquire } from './useWorth'
import WaitLine from './WaitLine'
import { LIST_FAILED, LIST_WAIT, SCORES_WAIT } from './wardrobeText'
import {
  ALL_MODES, FIRST_ROWS, MORE_ROWS, MORE_SUITS, NO_SOURCE, OWNED_KEY, PAST_NOTE, ROW_STEP, SCORE_F, UNLOCK_NOTE, chipText,
  detailsLabel, filterMode, filterSuits, gainText, groupPieces, groupTail, hardToGet, hideLabel, howToGet, improvesParts, itemMeta,
  neededLine, nothingText, openRows, openTarget, ownsAnyPart, pieceList, piecesText, rankedName, rankingNote, recipeText,
  rowKey, rowName, scoreFNote, stageLabel, suitPieces, unlockRanking, unlockText, worthFilter, worthKey, worthSuits,
  type AcquireTable, type OpenTarget, type UnlockRow, type WorthRequest,
  checkingText, handedOver, keptRows, moreText, rankingText, waitPercent, worthWait,
} from './worth'

type Props = {
  stages: Stage[]
  items: Item[] | null
  itemsFailed: boolean
  places: Place[]
  owned: ReadonlySet<number>
  version: string
}

const SETTLED_FRAMES = 5
const MAX_RESTORE_FRAMES = 60
const SKEL_ROWS = [
  [64, 34, 78, 56],
  [52, 30, 82, 44],
  [46, 38, 70, 60],
]

type RankRow = { key: string; rank: number; row: WorthRow }
type UnlockRank = { key: string; rank: number; row: UnlockRow }
type View = 'raise' | 'unlock'

function StageRow({ label, note, onOpen }: { label: string; note?: string; onOpen: () => void }) {
  return (
    <button type="button" className="nb-stage-row" onClick={onOpen}>
      <span className="nb-stage-row-text">
        <span>{label}</span>
        {note && <span className="nb-stage-row-note">{note}</span>}
      </span>
      <RightOutlined className="nb-stage-row-chevron" aria-hidden />
    </button>
  )
}

function Chevron({ open, label, onToggle }: { open: boolean; label: string; onToggle: () => void }) {
  return (
    <button
      type="button"
      className={open ? 'nb-chevron is-open' : 'nb-chevron'}
      aria-label={detailsLabel(label)}
      aria-expanded={open}
      onClick={(e) => {
        onToggle()
        e.stopPropagation()
      }}
    >
      <DownOutlined />
    </button>
  )
}

function AcquireWait({ acquire }: { acquire: Acquire }) {
  if (acquire.loading) {
    return (
      <div className="nb-skel-lines">
        <Skel width="72%" />
        <Skel width="48%" />
      </div>
    )
  }
  return <Typography.Text type="secondary">How to get items couldn't be loaded. Open this tab again to retry.</Typography.Text>
}

function OwnedKey() {
  return (
    <Typography.Text type="secondary" className="nb-worth-meta">
      {OWNED_KEY}
    </Typography.Text>
  )
}

function Ways({
  id, acquire, owned, names, onOpen, showKey = true,
}: {
  id: number
  acquire: Acquire
  owned: ReadonlySet<number>
  names: ReadonlyMap<number, string>
  onOpen: (target: OpenTarget) => void
  showKey?: boolean
}) {
  if (!acquire.table) return <AcquireWait acquire={acquire} />
  const lines = howToGet(acquire.table[String(id)], owned, names)
  if (!lines.length) return <Typography.Text type="secondary">{NO_SOURCE}</Typography.Text>

  return (
    <ul className="nb-worth-list">
      {lines.map((line, i) => {
        const stage = line.stage
        const recipe = line.recipe ? recipeText(line.recipe) : null
        const past = line.past ? PAST_NOTE : undefined
        return (
          <li key={i}>
            {stage && !recipe ? (
              <StageRow label={line.text} note={past} onOpen={() => onOpen(stage)} />
            ) : (
              <div className="nb-worth-line">
                <span>
                  {line.text}
                  {past && <Typography.Text type="secondary"> · {past}</Typography.Text>}
                </span>
              </div>
            )}
            {recipe &&
              (stage ? (
                <StageRow label={recipe} onOpen={() => onOpen(stage)} />
              ) : (
                <Typography.Text type="secondary" className="nb-worth-recipe nb-worth-block">
                  {recipe}
                </Typography.Text>
              ))}
            {line.from.length > 0 && (
              <div className="nb-worth-from">
                {line.from.map((part) => (
                  <span key={part.id} className={part.owned ? 'nb-chip is-owned' : 'nb-chip'}>
                    {chipText(part)}
                  </span>
                ))}
              </div>
            )}
          </li>
        )
      })}
      {showKey && ownsAnyPart(lines) && (
        <li>
          <OwnedKey />
        </li>
      )}
    </ul>
  )
}

function Improves({ row, mode, variants, top }: { row: WorthRow; mode: string; variants: ReadonlySet<string>; top: number }) {
  const [lead, stage, tail] = improvesParts(row, mode, variants)
  return (
    <div className="nb-worth-improves">
      <span>
        {lead}
        <span className="nb-worth-stage">{stage}</span>
        {tail}
      </span>
      <div className="nb-worth-bar" aria-hidden="true">
        <span style={{ width: `${top > 0 ? Math.max(2, (row.worth / top) * 100) : 0}%` }} />
      </div>
    </div>
  )
}

function FirstWays({ ids, acquire }: { ids: number[]; acquire: Acquire }) {
  if (acquire.loading) return <Skel width="64%" />
  if (!acquire.table) return <Typography.Text type="secondary">—</Typography.Text>
  return (
    <div className="nb-worth-item">
      {ids.map((id) => {
        const first = acquire.table?.[String(id)]?.[0]
        return first ? (
          <span key={id}>
            {first.t}
            {first.past === 1 && <Typography.Text type="secondary"> · {PAST_NOTE}</Typography.Text>}
          </span>
        ) : (
          <Typography.Text key={id} type="secondary">
            {NO_SOURCE}
          </Typography.Text>
        )
      })}
    </div>
  )
}

export default function WorthTab({ stages, items, itemsFailed, places, owned, version }: Props) {
  const ownedIds = useStore((s) => s.owned)
  const tab = useStore((s) => s.tab)
  const difficulty = useStore((s) => s.difficulty)
  const setDifficulty = useStore((s) => s.setDifficulty)
  const skills = useStore((s) => s.skills)
  const openStage = useStore((s) => s.openStage)
  const worthY = useStore((s) => s.worthY)
  const set = useStore((s) => s.set)
  const active = tab === 'worth'
  const screens = Grid.useBreakpoint()
  const table = useIdeals()
  const acquire = useAcquire(active)

  useLayoutEffect(() => {
    if (!active || worthY === null) return
    const roots = [document.documentElement, document.body]
    roots.forEach((el) => el.style.setProperty('overflow-anchor', 'none'))
    let frame = 0
    let height = -1
    let still = 0
    let count = 0
    let last: number | null = null
    const restore = () => {
      if (last !== null && Math.abs(window.scrollY - last) > 1) {
        set({ worthY: null })
        return
      }
      window.scrollTo({ top: worthY })
      last = window.scrollY
      const now = document.documentElement.scrollHeight
      still = now === height ? still + 1 : 0
      height = now
      count += 1
      if (still >= SETTLED_FRAMES || count >= MAX_RESTORE_FRAMES) set({ worthY: null })
      else frame = requestAnimationFrame(restore)
    }
    frame = requestAnimationFrame(restore)
    return () => {
      cancelAnimationFrame(frame)
      roots.forEach((el) => el.style.removeProperty('overflow-anchor'))
    }
  }, [active, worthY, set])

  const [mode, setMode] = useState(ALL_MODES)
  const [slot, setSlot] = useState(ANY_SLOT)
  const [bySuit, setBySuit] = useState(false)
  const choices = useMemo(() => slotOptions(places), [places])
  const [shown, setShown] = useState(FIRST_ROWS)
  const [view, setView] = useState<View>('raise')
  const [openKeys, setOpenKeys] = useState<string[]>([])
  const [openUnlocks, setOpenUnlocks] = useState<string[]>([])
  const phone = usePhone()

  const settings = worthSettings(skills)
  const settingsKey = JSON.stringify(settings)
  const request = useMemo<WorthRequest | null>(
    () =>
      ownedIds.length && table && items
        ? {
            key: worthKey(version, settings, ownedIds),
            settings,
            versions: () => worthVersions(stages, table, settings ? 'max' : 'none'),
            suits: () => worthSuits(items),
          }
        : null,
    [ownedIds, table, items, version, stages, settingsKey],
  )
  const run = useWorthRun(active, request)

  const modes = useMemo(() => [ALL_MODES, ...orderModes(stages)], [stages])
  const variants = useMemo(() => variantStages(stages), [stages])
  const byKey = useMemo(() => new Map(stages.map((s) => [stageKey(s), s])), [stages])
  const skip = useMemo(() => worthSkip(stages, difficulty), [stages, difficulty])
  const raise = view === 'raise'
  const filter = useMemo(
    () => worthFilter(mode, raise ? slotChoice(slot) : {}, skip, raise && bySuit),
    [mode, slot, skip, bySuit, raise],
  )
  const most = bySuit ? MORE_SUITS : MORE_ROWS
  const limit = shown > FIRST_ROWS ? most : FIRST_ROWS
  const { ready, ranking, rankedFilter, error, loading, current, streaming, got, of, first } = useWorthRanking(run, active, filter, limit)
  const rankedMode = filterMode(rankedFilter)
  const rankedSuits = filterSuits(rankedFilter)

  const byId = useMemo(() => new Map((items ?? []).map((it) => [it.id, it])), [items])
  const names = useMemo(() => new Map((items ?? []).map((it) => [it.id, it.name])), [items])

  const needed = ranking?.needed ?? []
  const unlocks = useMemo<UnlockRank[]>(() => {
    const hard = (id: number) => (acquire.table ? hardToGet(acquire.table[String(id)]) : false)
    return unlockRanking(ranking?.needed ?? [], hard).map((row, i) => ({ key: row.items.join('+'), rank: i + 1, row }))
  }, [ranking, acquire.table])
  const listed = ranking?.rows.length ?? 0
  const stale = loading && !current
  const { lead, fetching, more } = worthWait({ raise, loading, current, streaming, limit, shown, most, listed })
  const kept = keptRows(shown, fetching)
  const rows = useMemo<RankRow[]>(
    () => (ranking?.rows ?? []).slice(0, kept).map((row, i) => ({ key: rowKey(row), rank: i + 1, row })),
    [ranking, kept],
  )
  const top = rows.reduce((most, r) => Math.max(most, r.row.worth), 0)
  const checking = run.phase === 'running' || run.phase === 'stopping'
  const handoff = useRef(false)
  handoff.current = handedOver(handoff.current, checking, lead)
  const leadClass = lead ? (handoff.current ? 'nb-worth-lead is-waiting is-now' : 'nb-worth-lead is-waiting') : 'nb-worth-lead'
  const ranked = {
    text: rankingText({
      suits: filterSuits(filter),
      mode,
      where: raise ? choiceLabel(slot, places) : null,
      got,
      of,
      first,
    }),
    percent: streaming ? waitPercent(got, of) : null,
  }
  const wait = checking
    ? {
        text: checkingText(run.done, run.total),
        percent: waitPercent(run.done, run.total),
        onStop: runner.stop,
        stopping: run.phase === 'stopping',
      }
    : null

  const open = (target: OpenTarget) => openStage(target, window.scrollY)
  const rowElement = (key: string) => document.querySelector(`.nb-worth-table tr[data-row-key="${CSS.escape(key)}"]`)
  const toggleRow = (key: string, show: boolean, update: Dispatch<SetStateAction<string[]>>) => {
    const before = rowElement(key)?.getBoundingClientRect().top
    flushSync(() => update((keys) => openRows(keys, key, show, phone)))
    const after = rowElement(key)?.getBoundingClientRect().top
    if (show && phone && before !== undefined && after !== undefined) window.scrollBy(0, after - before)
  }
  const selecting = () => window.getSelection()?.isCollapsed === false
  const hideRow = (key: string, update: Dispatch<SetStateAction<string[]>>) => {
    const row = rowElement(key)
    row?.scrollIntoView({ block: 'nearest' })
    flushSync(() => update((keys) => openRows(keys, key, false, phone)))
    row?.querySelector<HTMLElement>('.nb-chevron')?.focus({ preventScroll: true })
  }
  const suitBlocks = (row: WorthRow, table: AcquireTable) => groupPieces(suitPieces(row, byId, places), table)
  const suitWays = (row: WorthRow, table: AcquireTable) => {
    const blocks = suitBlocks(row, table)
    const keyAt = blocks.findIndex((block) => !('pieces' in block) && ownsAnyPart(howToGet(table[String(block.id)], owned, names)))
    return blocks.map((block, i) =>
      'pieces' in block ? (
        <div key={block.key} className="nb-worth-ways">
          <Typography.Text className="nb-worth-subhead">
            {block.text} <span className="nb-worth-tail">{groupTail(block)}</span>
          </Typography.Text>
          <Typography.Text className="nb-worth-block">{pieceList(block.pieces)}</Typography.Text>
        </div>
      ) : (
        <div key={block.id} className="nb-worth-ways">
          <Typography.Text className="nb-worth-subhead">{block.name}</Typography.Text>
          {block.meta && (
            <Typography.Text type="secondary" className="nb-worth-meta nb-worth-block">
              {block.meta}
            </Typography.Text>
          )}
          <Ways id={block.id} acquire={acquire} owned={owned} names={names} onOpen={open} showKey={i === keyAt} />
        </div>
      ),
    )
  }
  const unchecked = (key: string) => {
    const target = openTarget(key, variants)
    const found = byKey.get(target.stage)
    return found ? rulesUnchecked(resolveStage(found, target.difficulty === 'Maiden' ? 'Maiden' : 'Princess').rules) : false
  }
  const refilter = (apply: () => void) => {
    apply()
    setShown(FIRST_ROWS)
  }
  const placesOf = (ids: readonly number[]) =>
    ids.map((id) => {
      const it = byId.get(id)
      return it ? placeName(it, places) : ''
    })

  if (!ownedIds.length) {
    return (
      <Empty className="nb-worth-empty" description="Import your wardrobe first. This tab then ranks what you don't own by how much it would raise your best scores.">
        <Button type="primary" onClick={() => set({ tab: 'outfit' })}>
          Import a wardrobe
        </Button>
      </Empty>
    )
  }
  if (table === undefined) return <WaitLine text={SCORES_WAIT} />
  if (table === null) {
    return (
      <Alert
        type="info"
        showIcon
        className="nb-alert"
        message="This ranking needs the table of best possible scores, which didn't load."
        description="Reload the page to try again."
      />
    )
  }
  if (!items) {
    return itemsFailed ? (
      <Alert type="info" showIcon className="nb-alert" message={LIST_FAILED} description="Reload the page to try again." />
    ) : (
      <WaitLine text={LIST_WAIT} />
    )
  }

  const columns = [
    {
      title: '#',
      key: 'rank',
      width: 48,
      responsive: ['sm' as const],
      render: (_: unknown, r: RankRow) => <span className="nb-worth-rank">{r.rank}</span>,
    },
    {
      title: rankedSuits ? 'Suit' : 'Item',
      key: 'item',
      width: screens.md ? '28%' : screens.sm ? '40%' : undefined,
      render: (_: unknown, r: RankRow) => {
        const ids = r.row.items
        const meta = r.row.suit ? piecesText(ids.length) : itemMeta(r.row, places, byId.get(ids[0])?.rarity ?? 0)
        const name = rowName(r.row, names)
        return (
          <div className="nb-worth-item">
            <span className="nb-worth-name">{phone ? rankedName(r.rank, name) : name}</span>
            {meta && (
              <Typography.Text type="secondary" className="nb-worth-meta">
                {meta}
              </Typography.Text>
            )}
            {!screens.sm && <Improves row={r.row} mode={rankedMode} variants={variants} top={top} />}
          </div>
        )
      },
    },
    {
      title: 'Gain',
      key: 'improves',
      responsive: ['sm' as const],
      render: (_: unknown, r: RankRow) => <Improves row={r.row} mode={rankedMode} variants={variants} top={top} />,
    },
    {
      title: 'How to get',
      key: 'how',
      width: '28%',
      responsive: ['md' as const],
      render: (_: unknown, r: RankRow) =>
        r.row.suit ? (
          <Typography.Text type="secondary">Open the row to see how to get each piece</Typography.Text>
        ) : (
          <FirstWays ids={r.row.items} acquire={acquire} />
        ),
    },
    Table.EXPAND_COLUMN,
  ]

  const details = (r: RankRow) => (
    <div className="nb-worth-detail">
      <div>
        <Typography.Text strong>Where it helps most</Typography.Text>
        <ul className="nb-stage-rows">
          {r.row.examples.map((ex) => (
            <li key={ex.key}>
              <StageRow
                label={stageLabel(ex.key, variants)}
                note={gainText(ex, unchecked(ex.key))}
                onOpen={() => open(openTarget(ex.key, variants))}
              />
            </li>
          ))}
        </ul>
        {r.row.examples.some((ex) => unchecked(ex.key)) && (
          <Typography.Paragraph type="secondary" className="nb-score-f">
            {scoreFNote(true)}
          </Typography.Paragraph>
        )}
      </div>
      {r.row.suit ? (
        <div>
          <Typography.Text strong>How to get each piece</Typography.Text>
          {acquire.table ? (
            suitWays(r.row, acquire.table)
          ) : (
            <div className="nb-worth-ways">
              <AcquireWait acquire={acquire} />
            </div>
          )}
        </div>
      ) : (
        <div>
          <Typography.Text strong>How to get {r.row.items.length > 1 ? 'them' : 'it'}</Typography.Text>
          {r.row.items.map((id) => (
            <div key={id} className="nb-worth-ways">
              {r.row.items.length > 1 && <Typography.Text className="nb-worth-subhead">{names.get(id) ?? `#${id}`}</Typography.Text>}
              <Ways id={id} acquire={acquire} owned={owned} names={names} onOpen={open} />
            </div>
          ))}
        </div>
      )}
      <Button
        block
        className="nb-worth-hide"
        aria-label={hideLabel(rowName(r.row, names))}
        onClick={() => hideRow(r.key, setOpenKeys)}
      >
        Hide details
      </Button>
    </div>
  )

  const unlockColumns = [
    {
      title: '#',
      key: 'rank',
      width: 48,
      responsive: ['sm' as const],
      render: (_: unknown, r: UnlockRank) => <span className="nb-worth-rank">{r.rank}</span>,
    },
    {
      title: 'Get',
      key: 'item',
      width: screens.md ? '32%' : screens.sm ? '45%' : undefined,
      render: (_: unknown, r: UnlockRank) => (
        <div className="nb-worth-item">
          <span className="nb-worth-name">{phone ? rankedName(r.rank, rowName(r.row, names)) : rowName(r.row, names)}</span>
          <Typography.Text type="secondary" className="nb-worth-meta">
            {placesOf(r.row.items).join(' + ')}
          </Typography.Text>
          {!screens.sm && <span className="nb-worth-unlocks">{unlockText(r.row, variants)}</span>}
        </div>
      ),
    },
    {
      title: 'Unlocks',
      key: 'unlocks',
      responsive: ['sm' as const],
      render: (_: unknown, r: UnlockRank) => (
        <span className="nb-worth-unlocks">
          {r.row.stages.length === 1 ? unlockText(r.row, variants) : `${unlockText(r.row, variants)}: ${r.row.stages.map((k) => stageLabel(k, variants)).join(', ')}`}
        </span>
      ),
    },
    {
      title: 'How to get',
      key: 'how',
      width: '28%',
      responsive: ['md' as const],
      render: (_: unknown, r: UnlockRank) => <FirstWays ids={r.row.items} acquire={acquire} />,
    },
    Table.EXPAND_COLUMN,
  ]

  const unlockDetails = (r: UnlockRank) => (
    <div className="nb-worth-detail">
      <div>
        <Typography.Text strong>{r.row.stages.length === 1 ? 'Stage it unlocks' : 'Stages they unlock'}</Typography.Text>
        <ul className="nb-stage-rows">
          {r.row.stages.map((key) => (
            <li key={key}>
              <StageRow
                label={stageLabel(key, variants)}
                note={unchecked(key) ? SCORE_F : undefined}
                onOpen={() => open(openTarget(key, variants))}
              />
            </li>
          ))}
        </ul>
        {r.row.stages.some((key) => unchecked(key)) && (
          <Typography.Paragraph type="secondary" className="nb-score-f">
            {scoreFNote(false)}
          </Typography.Paragraph>
        )}
      </div>
      <div>
        <Typography.Text strong>How to get {r.row.items.length > 1 ? 'them' : 'it'}</Typography.Text>
        {r.row.items.map((id) => (
          <div key={id} className="nb-worth-ways">
            {r.row.items.length > 1 && <Typography.Text className="nb-worth-subhead">{names.get(id) ?? `#${id}`}</Typography.Text>}
            <Ways id={id} acquire={acquire} owned={owned} names={names} onOpen={open} />
          </div>
        ))}
      </div>
      <Button
        block
        className="nb-worth-hide"
        aria-label={hideLabel(rowName(r.row, names))}
        onClick={() => hideRow(r.key, setOpenUnlocks)}
      >
        Hide details
      </Button>
    </div>
  )

  const total = run.total.toLocaleString('en-US')
  const done = run.done.toLocaleString('en-US')

  return (
    <>
      <Segmented<View>
        value={view}
        onChange={setView}
        options={[
          { value: 'raise', label: 'Raise scores' },
          { value: 'unlock', label: ranking && current ? `Unlock stages (${needed.length.toLocaleString('en-US')})` : 'Unlock stages' },
        ]}
        className="nb-worth-view"
      />
      <Space wrap className="nb-item-filters" size={[12, 12]}>
        <Segmented
          value={mode}
          onChange={(value) => refilter(() => setMode(String(value)))}
          options={modes}
          className="nb-worth-modes"
        />
        {(mode === ALL_MODES || mode === 'Story') && (
          <Segmented<Difficulty>
            value={difficulty}
            onChange={(value) => refilter(() => setDifficulty(value))}
            options={['Maiden', 'Princess']}
          />
        )}
        {view === 'raise' && (
          <Select
            aria-label="Slot"
            value={slot}
            onChange={(value) => refilter(() => setSlot(value))}
            className="nb-slot-select"
            popupMatchSelectWidth={false}
            options={choices}
            virtual={false}
          />
        )}
        {view === 'raise' && (
          <label className="nb-toggle">
            <Switch checked={bySuit} onChange={(value) => refilter(() => setBySuit(value))} aria-label="Group by suit" />
            <Typography.Text>group by suit</Typography.Text>
          </label>
        )}
      </Space>

      {raise && (
        <Typography.Paragraph type="secondary" className="nb-worth-skills">
          {worthSkillsText(skills)}
        </Typography.Paragraph>
      )}

      {wait && <WaitLine className="nb-worth-progress" {...wait} />}

      {run.phase === 'stopped' && (
        <Alert
          type="warning"
          showIcon
          className="nb-alert"
          message={run.done ? `Stopped after ${done} of ${total} stages.` : 'Stopped before any stage was checked.'}
          description={run.done ? 'The list below covers only the stages checked so far.' : undefined}
          action={
            <Button onClick={runner.resume}>
              Continue
            </Button>
          }
        />
      )}

      {run.phase === 'failed' && (
        <Alert
          type="error"
          showIcon
          className="nb-alert"
          message={run.error}
          action={
            <Button onClick={runner.retry}>
              Try again
            </Button>
          }
        />
      )}

      {error && <Alert type="error" showIcon className="nb-alert" message={error} />}

      {ready && view === 'unlock' && (
        <>
          <div className={leadClass}>
            {lead && <WaitLine className="nb-worth-progress" {...ranked} />}
            <Typography.Paragraph type="secondary" ellipsis={{ rows: 1, expandable: true, symbol: "how it's ranked" }}>
              {UNLOCK_NOTE}
            </Typography.Paragraph>
          </div>
          {ranking && !unlocks.length && !lead ? (
            <Empty description="You can pass every stage here with what you own." />
          ) : (
            <Table<UnlockRank>
              size="small"
              pagination={false}
              rowKey="key"
              aria-busy={stale || undefined}
              dataSource={unlocks}
              columns={unlockColumns}
              rowClassName="nb-row-tap"
              expandable={{
                expandedRowKeys: openUnlocks,
                onExpand: (show, r) => {
                  if (!selecting()) toggleRow(r.key, show, setOpenUnlocks)
                },
                expandRowByClick: true,
                columnWidth: 44,
                expandIcon: ({ expanded, record }) => (
                  <Chevron
                    open={expanded}
                    label={rowName(record.row, names)}
                    onToggle={() => toggleRow(record.key, !expanded, setOpenUnlocks)}
                  />
                ),
                expandedRowRender: unlockDetails,
              }}
              locale={{ emptyText: ' ' }}
              className={stale ? 'nb-worth-table is-stale' : 'nb-worth-table'}
            />
          )}
        </>
      )}

      {ready && view === 'raise' && (
        <>
          <div className={leadClass}>
            {lead && <WaitLine className="nb-worth-progress" {...ranked} />}
            <Typography.Paragraph type="secondary" ellipsis={{ rows: 1, expandable: true, symbol: "how it's ranked" }}>
              {rankingNote(rankedSuits, !!(rankedFilter.slots || rankedFilter.places))}
            </Typography.Paragraph>
          </div>

          {needed.length > 0 && (
            <div className="nb-worth-needed">
              <Typography.Text type="secondary">{neededLine(needed.length)}</Typography.Text>
              <Button type="link" className="nb-worth-needed-link" onClick={() => setView('unlock')}>
                See what unlocks them
              </Button>
            </div>
          )}

          {ranking && !rows.length && !loading ? (
            <Empty description={nothingText(rankedSuits)} />
          ) : (
            <Table<RankRow>
              size="small"
              pagination={false}
              rowKey="key"
              aria-busy={stale || undefined}
              dataSource={rows}
              columns={columns}
              rowClassName="nb-row-tap"
              expandable={{
                expandedRowKeys: openKeys,
                onExpand: (show, r) => {
                  if (!selecting()) toggleRow(r.key, show, setOpenKeys)
                },
                expandRowByClick: true,
                columnWidth: 44,
                expandIcon: ({ expanded, record }) => (
                  <Chevron
                    open={expanded}
                    label={rowName(record.row, names)}
                    onToggle={() => toggleRow(record.key, !expanded, setOpenKeys)}
                  />
                ),
                expandedRowRender: details,
              }}
              locale={{ emptyText: ' ' }}
              className={stale ? 'nb-worth-table is-stale' : 'nb-worth-table'}
            />
          )}

          {more && (
            <Button
              className="nb-worth-more"
              loading={fetching && { icon: <Petals size={14} /> }}
              disabled={fetching}
              onClick={() => setShown(shown + ROW_STEP)}
            >
              {fetching ? moreText(got, of, kept) : 'Show more'}
            </Button>
          )}

          {fetching && (
            <div className="nb-skel-rows" aria-hidden="true">
              {SKEL_ROWS.map((widths, i) => (
                <div key={i} className="nb-skel-row">
                  {widths.map((w, j) => (
                    <Skel key={j} width={`${w}%`} />
                  ))}
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </>
  )
}
