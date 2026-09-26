import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Alert, Layout, Tabs } from 'antd'
import { useStore } from './store'
import { useBundle } from './useBundle'
import { useBestOutfit } from './useBestOutfit'
import { useWardrobe } from './useWardrobe'
import { usePhone } from './usePhone'
import Blossom from './Blossom'
import Header from './Header'
import Footer from './Footer'
import BestOutfitTab from './BestOutfitTab'
import ItemBrowser from './ItemBrowser'
import WorthTab from './WorthTab'
import WaitLine from './WaitLine'
import { itemsTabLabel } from './items'
import { LIST_FAILED, LIST_WAIT } from './wardrobeText'

export default function App() {
  const { ready, owned, source, notice, error, tab, set } = useStore()
  const { stages, items, itemsFailed, tagNames, places, version } = useBundle()
  useBestOutfit(stages)
  const { ingest, toggleOwned, forget } = useWardrobe(version)
  const ownedSet = useMemo(() => new Set(owned), [owned])
  const phone = usePhone()

  const page = useRef<HTMLElement>(null)
  const [covered, setCovered] = useState(true)
  const uncover = useCallback(() => setCovered(false), [])
  useEffect(() => {
    page.current?.toggleAttribute('inert', covered)
  }, [covered])

  const [announcement, setAnnouncement] = useState('')
  useEffect(() => {
    setAnnouncement(ready ? 'NikkiBase is ready.' : 'Loading NikkiBase.')
  }, [ready])

  return (
    <>
      <Blossom done={ready} failed={!!error && !ready} onUncover={uncover} />
      <p className="nb-visually-hidden" role="status" aria-live="polite">
        {announcement}
      </p>
      <Layout ref={page} className="nb-page">
        <Header onForget={forget} />

        <Layout.Content className="nb-content">
          {error && (
            <Alert type="error" message={error} className="nb-alert" showIcon closable onClose={() => set({ error: null })} />
          )}
          {notice && (
            <Alert
              type="warning"
              message={notice}
              className="nb-alert"
              showIcon
              closable
              onClose={() => set({ notice: null })}
            />
          )}

          {ready && (
            <Tabs
              activeKey={tab}
              onChange={(key) => {
                set({ tab: key })
                if (phone) window.scrollTo({ top: 0 })
              }}
              items={[
                {
                  key: 'outfit',
                  label: 'Best outfit',
                  children: (
                    <BestOutfitTab
                      stages={stages}
                      items={items}
                      itemsFailed={itemsFailed}
                      tagNames={tagNames}
                      places={places}
                      onFile={ingest}
                    />
                  ),
                },
                {
                  key: 'worth',
                  label: 'Worth getting',
                  children: (
                    <WorthTab
                      stages={stages}
                      items={items}
                      itemsFailed={itemsFailed}
                      places={places}
                      owned={ownedSet}
                      version={version}
                    />
                  ),
                },
                {
                  key: 'items',
                  label: itemsTabLabel(items ? items.length : null, phone),
                  children: items ? (
                    <ItemBrowser
                      items={items}
                      places={places}
                      owned={ownedSet}
                      onToggle={toggleOwned}
                      manual={source === 'manual' || owned.length === 0}
                    />
                  ) : itemsFailed ? (
                    <Alert type="info" showIcon className="nb-alert" message={LIST_FAILED} description="Reload the page to try again." />
                  ) : (
                    <WaitLine text={LIST_WAIT} />
                  ),
                },
              ]}
            />
          )}
        </Layout.Content>

        <Footer version={version} />
      </Layout>
    </>
  )
}
