import { useEffect, useRef, type RefObject } from 'react'
import { HOLD_MS, blocksSwipe, nearEdge, swipeAxis, swipePull, swipeSpeed, swipeStep, tabAfter, track, type Axis, type Sample } from './swipe'

const SETTLE_MS = 260
const HOLDER = '.nb-content > .ant-tabs > .ant-tabs-content-holder'
const CHROME = '.ant-layout-header, .ant-tabs-nav'

type Options = {
  keys: readonly string[]
  tab: string
  enabled: boolean
  onSwipe: (key: string) => void
}

type Gesture = { x: number; y: number; t: number; holder: HTMLElement; axis: Axis | null; samples: Sample[] }

function selecting(): boolean {
  const s = document.getSelection()
  return !!s && !s.isCollapsed
}

export function useTabSwipe(area: RefObject<HTMLElement>, { keys, tab, enabled, onSwipe }: Options) {
  const latest = useRef({ keys, tab, onSwipe })
  latest.current = { keys, tab, onSwipe }

  useEffect(() => {
    const root = area.current
    if (!enabled || !root) return
    let g: Gesture | null = null
    let settling: { holder: HTMLElement; timer: ReturnType<typeof setTimeout> } | null = null

    const settle = (holder: HTMLElement, state: 'back' | 'next' | 'prev') => {
      holder.style.removeProperty('--nb-pull')
      holder.dataset.swipe = state
      const timer = setTimeout(() => {
        delete holder.dataset.swipe
        settling = null
      }, SETTLE_MS)
      settling = { holder, timer }
    }

    const stopSettling = () => {
      if (!settling) return
      clearTimeout(settling.timer)
      delete settling.holder.dataset.swipe
      settling = null
    }

    const drop = () => {
      if (g?.axis === 'x') settle(g.holder, 'back')
      g = null
    }

    const onStart = (e: TouchEvent) => {
      if (g) {
        drop()
        return
      }
      if (e.touches.length !== 1) return
      const target = e.target instanceof Element ? e.target : null
      const holder = root.querySelector<HTMLElement>(HOLDER)
      if (!target || !holder || target.closest(CHROME)) return
      const t = e.touches[0]
      if (nearEdge(t.clientX, window.innerWidth)) return
      if ((window.visualViewport?.scale ?? 1) > 1.01) return
      if (selecting()) return
      if (blocksSwipe(target, root, document.activeElement, (el) => getComputedStyle(el).overflowX)) return
      g = { x: t.clientX, y: t.clientY, t: e.timeStamp, holder, axis: null, samples: [{ x: t.clientX, t: e.timeStamp }] }
    }

    const onMove = (e: TouchEvent) => {
      if (!g) return
      if (e.touches.length !== 1) return drop()
      const t = e.touches[0]
      const dx = t.clientX - g.x
      g.samples = track(g.samples, { x: t.clientX, t: e.timeStamp })
      if (!g.axis) {
        if (e.timeStamp - g.t > HOLD_MS) return drop()
        g.axis = swipeAxis(dx, t.clientY - g.y)
        if (!g.axis) return
        if (g.axis === 'y') return drop()
        stopSettling()
        g.holder.dataset.swipe = 'drag'
      }
      const { keys, tab } = latest.current
      const open = tabAfter(keys, tab, dx < 0 ? 1 : -1) !== null
      g.holder.style.setProperty('--nb-pull', `${swipePull(dx, open)}px`)
    }

    const onEnd = (e: TouchEvent) => {
      if (!g || g.axis !== 'x') {
        g = null
        return
      }
      const { holder } = g
      const t = e.changedTouches[0]
      const step = selecting() || !t ? 0 : swipeStep(t.clientX - g.x, swipeSpeed(track(g.samples, { x: t.clientX, t: e.timeStamp })), window.innerWidth)
      g = null
      const { keys, tab, onSwipe } = latest.current
      const next = tabAfter(keys, tab, step)
      if (!next) return settle(holder, 'back')
      settle(holder, step > 0 ? 'next' : 'prev')
      onSwipe(next)
    }

    root.addEventListener('touchstart', onStart, { passive: true })
    root.addEventListener('touchmove', onMove, { passive: true })
    root.addEventListener('touchend', onEnd, { passive: true })
    root.addEventListener('touchcancel', drop, { passive: true })
    return () => {
      root.removeEventListener('touchstart', onStart)
      root.removeEventListener('touchmove', onMove)
      root.removeEventListener('touchend', onEnd)
      root.removeEventListener('touchcancel', drop)
      if (g) {
        g.holder.style.removeProperty('--nb-pull')
        delete g.holder.dataset.swipe
      }
      stopSettling()
    }
  }, [area, enabled])
}
