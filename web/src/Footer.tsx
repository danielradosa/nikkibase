import { useState } from 'react'
import { Layout, Typography } from 'antd'
import Credits from './Credits'
import Privacy from './Privacy'

export default function Footer({ version }: { version: string }) {
  const [creditsOpen, setCreditsOpen] = useState(false)
  const [privacyOpen, setPrivacyOpen] = useState(false)

  return (
    <Layout.Footer className="nb-footer">
      <Typography.Paragraph type="secondary" className="nb-footer-note">
        <strong>Scores are estimates.</strong> The game doesn&apos;t publish exact item numbers, so NikkiBase
        works them out from each item&apos;s letter grades. Close items may rank differently in game.
      </Typography.Paragraph>
      <Typography.Paragraph type="secondary" className="nb-footer-sources">
        Data from the{' '}
        <a href="https://lovenikki.fandom.com" target="_blank" rel="noreferrer">
          Love Nikki Wiki
        </a>{' '}
        (
        <a href="https://creativecommons.org/licenses/by-sa/3.0/" target="_blank" rel="noreferrer">
          CC BY-SA 3.0
        </a>
        ) and five community sources with no licence located.{' '}
        <a
          href="#credits"
          onClick={(e) => {
            e.preventDefault()
            setCreditsOpen(true)
          }}
        >
          Credits &amp; licences
        </a>
        {' · '}
        <a
          href="#privacy"
          onClick={(e) => {
            e.preventDefault()
            setPrivacyOpen(true)
          }}
        >
          Privacy
        </a>
      </Typography.Paragraph>
      <Typography.Text type="secondary">
        Love Nikki and its game content are proprietary to their respective rights holders.
        NikkiBase is an unofficial fan project and is not affiliated with or endorsed by them.
      </Typography.Text>
      <Credits open={creditsOpen} onClose={() => setCreditsOpen(false)} version={version} />
      <Privacy open={privacyOpen} onClose={() => setPrivacyOpen(false)} />
    </Layout.Footer>
  )
}
