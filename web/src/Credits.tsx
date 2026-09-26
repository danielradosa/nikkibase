import { useEffect, useState } from 'react'
import { Collapse, Modal, Tag, Typography } from 'antd'

type SourceRecord = {
  id: string
  name: string
  by?: string
  url: string
  licence: string
  licenceUrl?: string
  permissionUrl?: string
  status: string
  attribution: string
  fields: string[]
  included: boolean
  basis?: 'licence' | 'permission' | 'recorded-exception' | 'research-only'
  exception?: { decidedOn: string }
  summary: string
}
type Provenance = { version: string; builtAt: string; mode: string; sources: SourceRecord[] }
type Contributors = { editors: string[]; anonymousEditors: number; note: string }

function Terms({ s }: { s: SourceRecord }) {
  if (s.basis === 'licence') {
    return (
      <Tag color="green">
        {s.licenceUrl ? (
          <a href={s.licenceUrl} target="_blank" rel="noreferrer">
            {s.licence}
          </a>
        ) : (
          s.licence
        )}
      </Tag>
    )
  }
  if (s.basis === 'permission') {
    return (
      <Tag color="green">
        {s.permissionUrl ? (
          <a href={s.permissionUrl} target="_blank" rel="noreferrer">
            Used with permission
          </a>
        ) : (
          'Used with permission'
        )}
      </Tag>
    )
  }
  if (s.basis === 'recorded-exception') {
    return <Tag color="gold">No licence located · included pending permission</Tag>
  }
  return <Tag>{s.status}</Tag>
}

export default function Credits({ open, onClose, version }: { open: boolean; onClose: () => void; version: string }) {
  const [prov, setProv] = useState<Provenance | null>(null)
  const [failed, setFailed] = useState(false)
  const [contributors, setContributors] = useState<Contributors | null>(null)

  useEffect(() => {
    if (!open || !version || prov) return
    fetch(`/data/${version}/provenance.json`)
      .then((r) => (r.ok ? r.json() : Promise.reject(new Error(String(r.status)))))
      .then(setProv)
      .catch(() => setFailed(true))
    fetch('/credits/wiki-contributors.json')
      .then((r) => r.json())
      .then(setContributors)
      .catch(() => {})
  }, [open, version, prov])

  const included = prov?.sources.filter((s) => s.included) ?? []
  const excluded = prov?.sources.filter((s) => !s.included && s.url) ?? []

  return (
    <Modal open={open} onCancel={onClose} footer={null} title="Credits & licences" width={760}>
      <Typography.Paragraph>
        Love Nikki and its game content are proprietary to their respective rights holders. NikkiBase
        is an unofficial fan project and is not affiliated with or endorsed by them. It shows item, suit,
        stage and style names only, so you can recognise them; it carries no artwork, item descriptions
        or story text.
      </Typography.Paragraph>

      <Typography.Title level={5}>Where the data comes from</Typography.Title>
      {failed && (
        <Typography.Paragraph type="danger">
          The source list couldn&apos;t be loaded.
        </Typography.Paragraph>
      )}
      {!prov && !failed && <Typography.Paragraph type="secondary">Loading…</Typography.Paragraph>}
      {prov && (
        <>
          <Typography.Paragraph type="secondary" className="nb-small">
            Data version {prov.version}, built {prov.builtAt.slice(0, 10)}. Full record:{' '}
            <a href={`/data/${prov.version}/provenance.json`} target="_blank" rel="noreferrer">
              provenance.json
            </a>
            .
          </Typography.Paragraph>
          {included.map((s) => (
            <div key={s.id} className="nb-credit">
              <Typography.Text strong>
                <a href={s.url} target="_blank" rel="noreferrer">
                  {s.name}
                </a>
              </Typography.Text>
              {s.by && <Typography.Text> by {s.by}</Typography.Text>} <Terms s={s} />
              <br />
              <Typography.Text type="secondary">Used for: {s.fields.join(', ')}.</Typography.Text>
              <br />
              {(s.basis === 'licence' || s.basis === 'permission') && s.attribution && (
                <>
                  <Typography.Text type="secondary" className="nb-small">
                    {s.attribution}
                  </Typography.Text>
                  <br />
                </>
              )}
              <Typography.Text type="secondary" className="nb-small">
                {s.summary}
                {s.exception && ` Decision recorded ${s.exception.decidedOn}.`}
              </Typography.Text>
            </div>
          ))}
          <Typography.Paragraph type="secondary" className="nb-small">
            &ldquo;No licence located&rdquo; means no licence permitting reuse was found. Crediting a
            source grants no right to reuse its data.
          </Typography.Paragraph>
          {excluded.length > 0 && (
            <>
              <Typography.Title level={5}>Not included</Typography.Title>
              {excluded.map((s) => (
                <Typography.Paragraph key={s.id} type="secondary">
                  {s.name}: {s.summary}
                </Typography.Paragraph>
              ))}
            </>
          )}
        </>
      )}

      <Collapse
        size="small"
        className="nb-editors"
        items={[
          {
            key: 'editors',
            label: contributors
              ? `Love Nikki Wiki editors (${contributors.editors.length} named, ${contributors.anonymousEditors.toLocaleString('en-US')} anonymous)`
              : 'Love Nikki Wiki editors',
            children: contributors ? (
              <>
                <Typography.Paragraph type="secondary">{contributors.note}</Typography.Paragraph>
                <Typography.Paragraph className="nb-small">{contributors.editors.join(', ')}</Typography.Paragraph>
              </>
            ) : (
              <Typography.Text type="secondary">Loading…</Typography.Text>
            ),
          },
        ]}
      />

      <Typography.Title level={5}>NikkiBase code</Typography.Title>
      <Typography.Paragraph>
        <a href="https://github.com/danielradosa/nikkibase" target="_blank" rel="noreferrer">
          MIT licence
        </a>
        , covering the source code only, not the data above. Bundled software (Go, React, Ant Design,
        Zustand, the Cormorant Garamond and Nunito Sans fonts) is listed with its licences in{' '}
        <a href="/third-party-notices.txt" target="_blank" rel="noreferrer">
          third-party-notices.txt
        </a>
        .
      </Typography.Paragraph>

      <Typography.Paragraph type="secondary" className="nb-credits-contact">
        Maintain one of these sources, or hold rights in anything shown here, and want it credited
        differently or removed?{' '}
        <a href="https://github.com/danielradosa/nikkibase/issues" target="_blank" rel="noreferrer">
          Open an issue
        </a>{' '}
        or email <a href="mailto:nikkibaseproject@gmail.com">nikkibaseproject@gmail.com</a>.
      </Typography.Paragraph>
    </Modal>
  )
}
