import { useCallback, useRef } from 'react'
import { engine } from './engine'
import { useStore } from './store'
import { clearWardrobe, saveWardrobe, type WardrobeSource } from './storage'
import { tick, tickState } from './ticks'
import { importError } from './wardrobeText'

export function useWardrobe(version: string) {
  const set = useStore((s) => s.set)
  const ticks = useRef(tickState())
  const before = useRef<WardrobeSource | null>(null)

  const adopt = useCallback(
    async (ids: number[], stats: { items: number; unresolved: number; known: number }, src: WardrobeSource) => {
      set({ owned: ids, source: src, decoded: stats, busy: false, outfit: null, ideal: null })
      if (version) await saveWardrobe({ version, ids, source: src, savedAt: Date.now() })
    },
    [set, version],
  )

  const ingest = useCallback(
    async (text: string) => {
      set({ busy: true, error: null, notice: null })
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
        set({ error: importError(e), busy: false })
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
          save: (ids) => (version ? saveWardrobe({ version, ids, source: 'manual', savedAt: Date.now() }) : Promise.resolve()),
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
