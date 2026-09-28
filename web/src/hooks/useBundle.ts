import { useEffect, useState } from 'react'
import { engine } from '../engine/engine'
import { engineReady, items, startup, version } from '../boot'
import type { Item } from '../items/items'
import { useStore } from '../store'
import { clearWardrobe, loadWardrobe } from '../wardrobe/storage'
import type { Place } from '../outfit/comparison'
import type { Stage } from '../outfit/stages'
import { READ_FAILED, discardNotice, engineReason } from '../wardrobe/wardrobeText'

export type Bundle = {
  stages: Stage[]
  items: Item[] | null
  itemsFailed: boolean
  tagNames: string[]
  places: Place[]
  version: string
}

const EMPTY: Bundle = { stages: [], items: null, itemsFailed: false, tagNames: [], places: [], version }

export function useBundle(): Bundle {
  const set = useStore((s) => s.set)
  const [bundle, setBundle] = useState<Bundle>(EMPTY)

  useEffect(() => {
    const report = (e: unknown) =>
      engineReady.then(
        () => set({ error: String(e) }),
        () => {},
      )
    items.then(
      (itemList) => setBundle((b) => ({ ...b, items: itemList })),
      (e) => {
        setBundle((b) => ({ ...b, itemsFailed: true }))
        report(e)
      },
    )
    engineReady.then(
      () => set({ engine: 'ready' }),
      (e) => set({ engine: 'failed', engineError: engineReason(e) }),
    )
    ;(async () => {
      try {
        const { stages, tagNames, places } = await startup
        setBundle((b) => ({ ...b, stages, tagNames, places }))

        const saved = await loadWardrobe(version)
        if (saved.status === 'ok') {
          const { ids, source } = saved.entry
          engine.setWardrobe(ids).then(
            (stats) => useStore.getState().owned === ids && set({ decoded: { ...stats, unresolved: 0 } }),
            report,
          )
          set({ owned: ids, source, decoded: null })
        } else {
          engine.loadKeystream().catch(() => {})
        }
        const notice = discardNotice(saved)
        if (notice) {
          set({ notice })
          await clearWardrobe()
        } else if (saved.status === 'unreadable') {
          set({ notice: READ_FAILED })
        }
        set({ ready: true })
      } catch (e) {
        set({ error: String(e) })
      }
    })()
  }, [set])

  return bundle
}
