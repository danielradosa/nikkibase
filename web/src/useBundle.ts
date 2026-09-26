import { useEffect, useState } from 'react'
import { engine } from './engine'
import { engineReady, items, startup, version } from './boot'
import type { Item } from './items'
import { useStore } from './store'
import { clearWardrobe, loadWardrobe } from './storage'
import type { Place } from './comparison'
import type { Stage } from './stages'
import { discardNotice } from './wardrobeText'

export type Bundle = {
  stages: Stage[]
  items: Item[] | null
  tagNames: string[]
  places: Place[]
  version: string
}

const EMPTY: Bundle = { stages: [], items: null, tagNames: [], places: [], version }

export function useBundle(): Bundle {
  const set = useStore((s) => s.set)
  const [bundle, setBundle] = useState<Bundle>(EMPTY)

  useEffect(() => {
    items.then(
      (itemList) => setBundle((b) => ({ ...b, items: itemList })),
      (e) => set({ error: String(e) }),
    )
    engineReady.catch((e) => set({ error: String(e) }))
    ;(async () => {
      try {
        const { stages, tagNames, places } = await startup
        setBundle((b) => ({ ...b, stages, tagNames, places }))

        const saved = await loadWardrobe(version)
        if (saved.status === 'ok') {
          const { ids, source } = saved.entry
          engine.setWardrobe(ids).then(
            (stats) => useStore.getState().owned === ids && set({ decoded: { ...stats, unresolved: 0 } }),
            (e) => set({ error: String(e) }),
          )
          set({ owned: ids, source, decoded: null })
        } else {
          engine.loadKeystream().catch(() => {})
        }
        const notice = discardNotice(saved)
        if (notice) {
          set({ notice })
          await clearWardrobe()
        }
        set({ ready: true })
      } catch (e) {
        set({ error: String(e) })
      }
    })()
  }, [set])

  return bundle
}
