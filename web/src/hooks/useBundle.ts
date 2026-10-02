import { useEffect, useState } from 'react'
import { engine } from '../engine/engine'
import { engineReady, items, startup, version } from '../boot'
import type { Item } from '../items/items'
import { useStore } from '../store'
import { clearWardrobe, loadWardrobe, saveWardrobe, type WardrobeSource } from '../wardrobe/storage'
import type { Place } from '../outfit/comparison'
import type { Stage } from '../outfit/stages'
import { noteSave } from '../wardrobe/useWardrobe'
import { engineReason, restorePlan, updatedWardrobe } from '../wardrobe/wardrobeText'

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
    const send = (ids: number[]) =>
      engine.setWardrobe(ids).then(
        (stats) => useStore.getState().owned === ids && set({ decoded: { ...stats, unresolved: 0 } }),
        report,
      )
    const resave = (ids: number[], source: WardrobeSource, savedAt: number) =>
      items.then(
        async (list) => {
          if (useStore.getState().owned !== ids) return
          const kept = updatedWardrobe(ids, new Set(list.map((it) => it.id)))
          if (kept.notice) {
            set({ owned: kept.ids, source: kept.ids.length ? source : null, savedAt: kept.ids.length ? savedAt : null, decoded: null, notice: kept.notice })
            send(kept.ids)
          }
          noteSave(await saveWardrobe({ version, ids: kept.ids, source, savedAt }))
        },
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

        const plan = restorePlan(await loadWardrobe(version))
        if (plan.action === 'load') {
          send(plan.ids)
          set({ owned: plan.ids, source: plan.source, savedAt: plan.savedAt, decoded: null })
          if (plan.resave) resave(plan.ids, plan.source, plan.savedAt ?? Date.now())
        } else {
          engine.loadKeystream().catch(() => {})
        }
        if (plan.action === 'clear') {
          set({ notice: plan.notice })
          await clearWardrobe()
        } else if (plan.action === 'warn') {
          set({ notice: plan.notice })
        }
        set({ ready: true })
      } catch (e) {
        set({ error: String(e) })
      }
    })()
  }, [set])

  return bundle
}
