import { useCallback, useRef } from 'react'
import { engine } from '../engine/engine'
import { useStore } from '../store'
import { clearWardrobe, saveWardrobe, type WardrobeSource } from './storage'
import { tick, tickState } from '../items/ticks'
import { afterSave, importError } from './wardrobeText'

export function noteSave(ok: boolean, fresh = false) {
  const { saveFailing, set } = useStore.getState()
  const next = afterSave(saveFailing && !fresh, ok)
  set(next.notice ? { saveFailing: next.failing, notice: next.notice } : { saveFailing: next.failing })
}

export function useWardrobe(version: string) {
  const set = useStore((s) => s.set)
  const ticks = useRef(tickState())
  const before = useRef<WardrobeSource | null>(null)

  const adopt = useCallback(
    async (ids: number[], stats: { items: number; unresolved: number; known: number }, src: WardrobeSource) => {
      set({ owned: ids, source: src, decoded: stats, importing: false, outfit: null, ideal: null })
      if (version) noteSave(await saveWardrobe({ version, ids, source: src, savedAt: Date.now() }), true)
    },
    [set, version],
  )

  const ingest = useCallback(
    async (text: string) => {
      if (useStore.getState().importing) return
      set({ importing: true, error: null, notice: null })
      try {
        if (!text.trimStart().startsWith('@SEL')) {
          const result = await engine.decode(text)
          await adopt(result.ids, result, 'clothes_date')
          return
        }
        const result = await engine.selections(text)
        await adopt(result.ids, result, 'sel')
      } catch (e) {
        console.warn(e)
        set({ error: importError(e), importing: false })
      }
    },
    [adopt, set],
  )

  const toggleOwned = useCallback(
    (id: number) => {
      if (ticks.current.waiting === 0) before.current = useStore.getState().source
      return tick(
        id,
        {
          owned: () => useStore.getState().owned,
          show: (next) => set({ owned: next, source: 'manual' }),
          send: (ids) => engine.setWardrobe(ids),
          settle: (decoded) => set({ decoded }),
          save: async (ids) => {
            if (version) noteSave(await saveWardrobe({ version, ids, source: 'manual', savedAt: Date.now() }))
          },
          fail: (confirmed, e, ticked) => set({ owned: confirmed, source: ticked ? 'manual' : before.current, error: String(e) }),
        },
        ticks.current,
      )
    },
    [set, version],
  )

  const forget = useCallback(async () => {
    await clearWardrobe()
    await engine.setWardrobe([])
    set({ owned: [], source: null, decoded: null, outfit: null, ideal: null })
  }, [set])

  return { ingest, toggleOwned, forget }
}
