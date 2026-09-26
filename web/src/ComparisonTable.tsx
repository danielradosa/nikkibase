import { CheckOutlined } from '@ant-design/icons'
import { Space, Table, Tag, Typography } from 'antd'
import { alternativesLabel, bestNote, copyLines, expandLabel, rowOpens, type ComparisonRow } from './comparison'
import type { Alternative } from './engine'
import { usePhone } from './usePhone'

type Props = { rows: ComparisonRow[]; busy: boolean; names: ReadonlyMap<number, string> }

type DetailsProps = { row: ComparisonRow; phone: boolean; names: ReadonlyMap<number, string> }

function Details({ row, phone, names }: DetailsProps) {
  return (
    <Space direction="vertical" size={4} className="nb-alts">
      {phone &&
        copyLines(row).map((line) => (
          <div key={line.label} className="nb-alt">
            <Typography.Text type="secondary">{line.label}:</Typography.Text>
            <Typography.Text className="nb-alt-name" copyable={{ text: line.name }}>
              {line.name}
            </Typography.Text>
          </div>
        ))}
      {row.alts.length > 0 && (
        <Typography.Text type="secondary">
          Other <Typography.Text strong>{row.slot}</Typography.Text> you own, and the points you&apos;d lose:
        </Typography.Text>
      )}
      {row.alts.map((alt: Alternative) => (
        <div key={alt.id} className="nb-alt">
          <Typography.Text className="nb-alt-name" copyable={{ text: names.get(alt.id) ?? `#${alt.id}` }}>
            {names.get(alt.id) ?? `#${alt.id}`}
          </Typography.Text>
          <Tag color={alt.delta === 0 ? 'default' : 'volcano'}>
            {alt.delta === 0 ? 'ties' : alt.delta.toLocaleString('en-US')}
          </Tag>
        </div>
      ))}
    </Space>
  )
}

export default function ComparisonTable({ rows, busy, names }: Props) {
  const phone = usePhone()

  return (
    <Table
      size="small"
      className="nb-outfit-table"
      pagination={false}
      loading={busy}
      dataSource={rows}
      rowClassName={(row: ComparisonRow) => (rowOpens(row, phone) ? 'nb-row-tap' : '')}
      columns={[
        { title: 'Slot', dataIndex: 'slot', width: phone ? 96 : 150 },
        {
          title: 'Your best',
          dataIndex: 'mine',
          render: (name: string | null, row: ComparisonRow) => (
            <>
              {name ? (
                <Typography.Text copyable={phone ? false : { text: name }}>{name}</Typography.Text>
              ) : (
                <Typography.Text type="secondary">{row.unworn}</Typography.Text>
              )}
              {phone && (
                <Typography.Text type={row.same ? 'success' : 'secondary'} className="nb-best-line">
                  {row.same ? (
                    <>
                      <CheckOutlined /> {bestNote(row)}
                    </>
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
          render: (name: string | null, row: ComparisonRow) =>
            row.same ? (
              <Typography.Text type="success">same item</Typography.Text>
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
        rowExpandable: (row: ComparisonRow) => rowOpens(row, phone),
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
        expandedRowRender: (row: ComparisonRow) => <Details row={row} phone={phone} names={names} />,
      }}
    />
  )
}
