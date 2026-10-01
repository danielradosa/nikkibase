import { CheckOutlined } from '@ant-design/icons'
import { Space, Table, Tag, Typography } from 'antd'
import { alternativesLabel, bestNote, expandLabel, rowOpens, type ComparisonRow } from './comparison'
import type { Alternative } from '../engine/engine'
import Skel from '../components/Skel'
import { usePhone } from '../hooks/usePhone'

type Props = { rows: ComparisonRow[]; names: ReadonlyMap<number, string>; naming: boolean }

type DetailsProps = { row: ComparisonRow; names: ReadonlyMap<number, string>; naming: boolean }

const NAME_WIDTHS = ['70%', '55%', '64%', '75%', '60%']

const nameSkel = (i: number) => <Skel width={NAME_WIDTHS[i % NAME_WIDTHS.length]} />

function Details({ row, names, naming }: DetailsProps) {
  return (
    <Space direction="vertical" size={4} className="nb-alts">
      {row.alts.length > 0 && (
        <Typography.Text type="secondary">
          Other <Typography.Text strong>{row.slot}</Typography.Text> you own, and the points you&apos;d lose:
        </Typography.Text>
      )}
      {row.alts.map((alt: Alternative, i) => (
        <div key={alt.id} className="nb-alt">
          {naming ? (
            nameSkel(i + 2)
          ) : (
            <Typography.Text className="nb-alt-name" copyable={{ text: names.get(alt.id) ?? `#${alt.id}` }}>
              {names.get(alt.id) ?? `#${alt.id}`}
            </Typography.Text>
          )}
          <Tag color={alt.delta === 0 ? 'default' : 'volcano'}>
            {alt.delta === 0 ? 'ties' : alt.delta.toLocaleString('en-US')}
          </Tag>
        </div>
      ))}
    </Space>
  )
}

export default function ComparisonTable({ rows, names, naming }: Props) {
  const phone = usePhone()

  return (
    <Table
      size="small"
      className="nb-outfit-table"
      pagination={false}
      aria-busy={naming || undefined}
      dataSource={rows}
      rowClassName={(row: ComparisonRow) => (rowOpens(row) ? 'nb-row-tap' : '')}
      columns={[
        { title: 'Slot', dataIndex: 'slot', width: phone ? 96 : 150 },
        {
          title: 'Your best',
          dataIndex: 'mine',
          render: (name: string | null, row: ComparisonRow, i: number) => (
            <>
              {name && naming ? (
                nameSkel(i)
              ) : name ? (
                <Typography.Text copyable={{ text: name }}>{name}</Typography.Text>
              ) : (
                <Typography.Text type="secondary">{row.unworn}</Typography.Text>
              )}
              {phone && (
                <Typography.Text
                  type={row.same ? 'success' : 'secondary'}
                  className="nb-best-line"
                  copyable={!row.same && row.best && !naming ? { text: row.best } : false}
                >
                  {row.same ? (
                    <>
                      <CheckOutlined /> {bestNote(row)}
                    </>
                  ) : naming && row.best ? (
                    nameSkel(i + 1)
                  ) : (
                    bestNote(row)
                  )}
                </Typography.Text>
              )}
            </>
          ),
        },
        {
          title: 'Best possible',
          dataIndex: 'best',
          responsive: ['sm' as const],
          render: (name: string | null, row: ComparisonRow, i: number) =>
            row.same ? (
              <Typography.Text type="success">same item</Typography.Text>
            ) : name && naming ? (
              nameSkel(i + 1)
            ) : name ? (
              <Typography.Text type="secondary" copyable={{ text: name }}>
                {name}
              </Typography.Text>
            ) : (
              <Typography.Text type="secondary">—</Typography.Text>
            ),
        },
        {
          title: 'Alternatives',
          dataIndex: 'alts',
          width: 120,
          responsive: ['md' as const],
          render: (alts: Alternative[], row: ComparisonRow) => (
            <Typography.Text type="secondary">
              {alternativesLabel(row.mine !== null, alts.length, row.moreAlts)}
            </Typography.Text>
          ),
        },
      ]}
      expandable={{
        expandRowByClick: true,
        rowExpandable: (row: ComparisonRow) => rowOpens(row),
        expandIcon: ({ prefixCls, expanded, expandable, onExpand, record }) => (
          <button
            type="button"
            className={`${prefixCls}-row-expand-icon ${
              expandable ? `${prefixCls}-row-expand-icon-${expanded ? 'expanded' : 'collapsed'}` : `${prefixCls}-row-expand-icon-spaced`
            }`}
            aria-label={expandLabel(record.slot)}
            aria-expanded={expanded}
            onClick={(e) => {
              onExpand(record, e)
              e.stopPropagation()
            }}
          />
        ),
        expandedRowRender: (row: ComparisonRow) => <Details row={row} names={names} naming={naming} />,
      }}
    />
  )
}
