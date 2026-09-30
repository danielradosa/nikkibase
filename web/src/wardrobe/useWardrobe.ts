import { useCallback, useRef } from 'react'
import { engine } from '../engine/engine'
import { items } from '../boot'
import { useStore } from '../store'
import { clearWardrobe, saveWardrobe, type WardrobeSource } from './storage'
import { tick, tickState } from '../items/ticks'
import { wardrobeKind } from './wardrobeFile'
import { afterSave, codeLoad, importError } from './wardrobeText'

export function noteSave(ok: boolean, fresh = false) {
  const { saveFailing, set } = useStore.getState()
  const next = afterSave(saveFailing && !fresh, ok)
  set(next.notice ? { saveFailing: next.failing, notice: next.notice } : { saveFailing: next.failing })
}

export function useWardrobe(version: string) {
  const set = useStore((s) => s.set)
  const ticks = useRef(tickState())
  const before = useRef<WardrobeSource | null>(null)

  const ingest = useCallback(
    async (text: string) => {
      if (useStore.getState().importing) return false
      set({ importing: true, error: null, notice: null })
      try {
        const kind = wardrobeKind(text)
        if (kind.kind === 'code') {
          const known = kind.read.ok ? await items.then((list) => new Set(list.map((it) => it.id)), () => null) : null
          const load = codeLoad(kind.read, known)
          if (!load.ok) {
            set({ error: load.error, importing: false })
            return false
          }
          const stats = await engine.setWardrobe(load.ids)
          set({
            owned: load.ids,
            source: 'nikkibase',
            decoded: { ...stats, unresolved: 0 },
            importing: false,
            outfit: null,
            ideal: null,
            notice: load.notice,
          })
          if (version) noteSave(await saveWardrobe({ version, ids: load.ids, source: 'nikkibase', savedAt: Date.now() }), true)
          return true
        }
        const source: WardrobeSource = kind.kind
        const result = await (source === 'sel' ? engine.selections(text) : engine.decode(text))
        set({ owned: result.ids, source, decoded: result, importing: false, outfit: null, ideal: null })
        if (version) noteSave(await saveWardrobe({ version, ids: result.ids, source, savedAt: Date.now() }), true)
        return true
      } catch (e) {
        console.warn(e)
        set({ error: importError(e), importing: false })
        return false
      }
    },
    [set, version],
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
