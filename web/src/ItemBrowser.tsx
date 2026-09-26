import { useLayoutEffect, useMemo, useRef, useState, type MouseEvent } from 'react'
import { Button, Checkbox, Empty, Grid, Input, Select, Space, Switch, Table, Tag, Tooltip, Typography } from 'antd'
import type { Place } from './comparison'
import { ANY_SLOT, ATTRS, ITEM_STEP, gradesLine, inChoice, itemPage, moreItemsText, ownLabel, tickHint, placeName, slotOptions, type Item } from './items'
import { useStore } from './store'
import { usePhone } from './usePhone'

export default function ItemBrowser({
  items,
  owned,
  onToggle,
  manual,
  places,
}: {
  items: Item[]
  places: readonly Place[]
  owned: Set<number>
  onToggle: (id: number) => void
  manual: boolean
}) {
  const [query, setQuery] = useState('')
  const [slot, setSlot] = useState(ANY_SLOT)
  const choices = useMemo(() => slotOptions(places), [places])
  const [ownedOnly, setOwnedOnly] = useState(false)
  const [scoreableOnly, setScoreableOnly] = useState(false)
  const [rarity, setRarity] = useState('all')
  const screens = Grid.useBreakpoint()
  const phone = usePhone()
  const onItems = useStore((s) => s.tab) === 'items'
  const [hint, setHint] = useState(manual)
  const showHint = tickHint(hint, manual, onItems)
  if (showHint !== hint) setHint(showHint)
  const filters = JSON.stringify([query, slot, ownedOnly, scoreableOnly, rarity])
  const [paging, setPaging] = useState({ filters, shown: ITEM_STEP })
  const page = itemPage(paging, filters)
  if (page !== paging) setPaging(page)
  const shown = page.shown
  const focusRow = useRef<number | null>(null)

  useLayoutEffect(() => {
    const index = focusRow.current
    if (index === null) return
    focusRow.current = null
    document
      .querySelectorAll<HTMLInputElement>('.nb-items-table .ant-table-tbody > tr.ant-table-row input')
      [index]?.focus({ preventScroll: true })
  }, [shown])

  const results = useMemo(() => {
    const needle = query.trim().toLowerCase()
    return items.filter(
      (it) =>
        inChoice(slot, it) &&
        (!ownedOnly || owned.has(it.id)) &&
        (!scoreableOnly || it.scoreable) &&
        (rarity === 'all' || String(it.rarity) === rarity) &&
        (needle === '' || it.search.includes(needle) || it.suit.toLowerCase().includes(needle)),
    )
  }, [items, owned, query, slot, ownedOnly, scoreableOnly, rarity])
  const more = phone ? moreItemsText(results.length, shown) : null

  const columns = [
    {
      title: 'Own',
      key: 'own',
      width: 44,
      className: 'nb-own-cell',
      onCell: (it: Item) => ({
        onClick: (e: MouseEvent<HTMLElement>) => {
          if (!(e.target as HTMLElement).closest('.ant-checkbox-wrapper')) onToggle(it.id)
        },
      }),
      render: (_: unknown, it: Item) => (
        <Checkbox checked={owned.has(it.id)} onChange={() => onToggle(it.id)} aria-label={ownLabel(it.name)} />
      ),
    },
    {
      title: 'Item',
      dataIndex: 'name',
      width: phone ? undefined : screens.md ? 240 : 150,
      render: (name: string, it: Item) => (
        <>
          <Space size={4} wrap>
            <span>{name}</span>
            {it.suit && (
              <Tooltip title={`Part of ${it.suit}`}>
                <Tag color="purple" className="nb-tag-tight">
                  {it.suit}
                </Tag>
              </Tooltip>
            )}
            {!it.scoreable && (
              <Tooltip title="This item's stats are not published anywhere, so it cannot be scored or recommended.">
                <Tag color="default">no stats</Tag>
              </Tooltip>
            )}
          </Space>
          {phone && it.scoreable && (
            <Typography.Text type="secondary" className="nb-grades">
              {gradesLine(it)}
            </Typography.Text>
          )}
        </>
      ),
    },
    {
      title: 'Slot',
      key: 'slot',
      width: 140,
      responsive: ['sm' as const],
      render: (_: unknown, it: Item) => placeName(it, places),
    },
    {
      title: 'Rarity',
      dataIndex: 'rarity',
      width: 90,
      responsive: ['lg' as const],
      sorter: (a: Item, b: Item) => a.rarity - b.rarity,
      render: (r: number) => (r ? '★'.repeat(r) : <Typography.Text type="secondary">—</Typography.Text>),
    },
    {
      title: 'Stats',
      key: 'attributes',
      responsive: ['sm' as const],
      render: (_: unknown, it: Item) => (
        <Space size={[4, 4]} wrap>
          {it.grades.map((grade, pair) =>
            grade ? (
              <Tag key={pair} className="nb-tag-tight">
                {ATTRS[it.attrs[pair]]} <strong>{grade}</strong>
              </Tag>
            ) : null,
          )}
        </Space>
      ),
    },
    { title: 'ID', dataIndex: 'id', width: 80, responsive: ['lg' as const] },
  ]

  return (
    <>
      <Space wrap className="nb-item-filters" size={[12, 12]}>
        <Input.Search
          allowClear
          placeholder="search by name or suit"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          className="nb-item-search"
        />
        <Select
          value={slot}
          onChange={setSlot}
          className="nb-slot-select"
          popupMatchSelectWidth={false}
          options={choices}
          virtual={false}
        />
        <label className="nb-toggle">
          <Switch checked={ownedOnly} onChange={setOwnedOnly} disabled={owned.size === 0} />
          <Typography.Text type={owned.size === 0 ? 'secondary' : undefined}>
            only what I own
          </Typography.Text>
        </label>
        <Select
          value={rarity}
          onChange={setRarity}
          className="nb-rarity-select"
          options={[
            { value: 'all', label: 'any rarity' },
            ...[1, 2, 3, 4, 5, 6].map((r) => ({ value: String(r), label: '★'.repeat(r) })),
          ]}
        />
        <label className="nb-toggle">
          <Switch checked={scoreableOnly} onChange={setScoreableOnly} />
          <Typography.Text>only items with stats</Typography.Text>
        </label>
      </Space>

      {showHint && (
        <Typography.Paragraph type="secondary">Tick what you own. It&apos;s saved in this browser.</Typography.Paragraph>
      )}

      <Typography.Paragraph type="secondary">
        Showing {results.length.toLocaleString('en-US')} of {items.length.toLocaleString('en-US')} items
      </Typography.Paragraph>

      {results.length === 0 ? (
        <Empty description="Nothing matches" />
      ) : (
        <>
          <Table
            virtual={!phone}
            size="small"
            rowKey="id"
            pagination={false}
            scroll={phone ? undefined : { y: 460 }}
            dataSource={phone ? results.slice(0, shown) : results}
            columns={columns}
            className="nb-items-table"
          />
          {more && (
            <Button
              className="nb-items-more"
              onClick={() => {
                focusRow.current = shown
                setPaging({ filters, shown: shown + ITEM_STEP })
              }}
            >
              {more}
            </Button>
          )}
        </>
      )}
    </>
  )
}
