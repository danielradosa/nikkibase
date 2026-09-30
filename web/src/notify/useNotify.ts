import { useCallback, useEffect, useState } from 'react'
import { TOPICS_KEY, isApple, keyBytes, parseTopics, sameKey, supportOf, type Status, type Support, type Topic } from './notify'

const READY_MS = 10000

function savedTopics(): Topic[] {
  try {
    return parseTopics(localStorage.getItem(TOPICS_KEY))
  } catch {
    return []
  }
}

function keepTopics(topics: Topic[]) {
  try {
    if (topics.length) localStorage.setItem(TOPICS_KEY, JSON.stringify(topics))
    else localStorage.removeItem(TOPICS_KEY)
  } catch {
  }
}

function environment(): Support {
  const nav = navigator as Navigator & { standalone?: boolean }
  return supportOf({
    serviceWorker: 'serviceWorker' in nav,
    pushManager: 'PushManager' in window,
    notification: 'Notification' in window,
    apple: isApple(nav.userAgent, nav.platform, nav.maxTouchPoints ?? 0),
    standalone: window.matchMedia?.('(display-mode: standalone)').matches || nav.standalone === true,
  })
}

async function post(path: string, body: unknown) {
  const res = await fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })
  if (!res.ok) throw new Error(`${path} answered ${res.status}`)
}

function registration(): Promise<ServiceWorkerRegistration> {
  return Promise.race([
    navigator.serviceWorker.ready,
    new Promise<never>((_, reject) => setTimeout(() => reject(new Error('no service worker')), READY_MS)),
  ])
}

export type Notify = {
  available: boolean
  topics: Topic[]
  status: Status
  pending: Topic | null
  apply: (next: Topic[], changed: Topic) => void
}

export function useNotify(): Notify {
  const [support] = useState(environment)
  const [key, setKey] = useState<Uint8Array | null>(null)
  const [topics, setTopics] = useState<Topic[]>(savedTopics)
  const [pending, setPending] = useState<Topic | null>(null)
  const [failed, setFailed] = useState(false)
  const [denied, setDenied] = useState(() => support === 'yes' && Notification.permission === 'denied')

  useEffect(() => {
    if (support === 'no') return
    let live = true
    fetch('/push/key')
      .then((res) => (res.ok && res.headers.get('content-type')?.startsWith('application/json') ? res.json() : null))
      .then((body) => {
        if (live && body && typeof body.key === 'string') setKey(keyBytes(body.key))
      })
      .catch(() => {})
    return () => {
      live = false
    }
  }, [support])

  useEffect(() => {
    if (!key || support !== 'yes') return
    let live = true
    registration()
      .then((reg) => reg.pushManager.getSubscription())
      .then(async (sub) => {
        if (!live) return
        const saved = savedTopics()
        if (sub && saved.length === 0) {
          await post('/push/unsubscribe', { endpoint: sub.endpoint }).catch(() => {})
          await sub.unsubscribe()
        }
        if ((!sub || Notification.permission !== 'granted') && saved.length) {
          keepTopics([])
          setTopics([])
        }
      })
      .catch(() => {})
    return () => {
      live = false
    }
  }, [key, support])

  const apply = useCallback(
    (next: Topic[], changed: Topic) => {
      if (!key || support !== 'yes' || pending) return
      setPending(changed)
      setFailed(false)
      const run = async () => {
        if (next.length && Notification.permission !== 'granted') {
          const answer = await Notification.requestPermission()
          if (answer !== 'granted') {
            setDenied(answer === 'denied')
            return
          }
        }
        setDenied(false)
        const reg = await registration()
        let sub = await reg.pushManager.getSubscription()
        if (next.length === 0) {
          if (sub) {
            await post('/push/unsubscribe', { endpoint: sub.endpoint }).catch(() => {})
            await sub.unsubscribe()
          }
          keepTopics([])
          setTopics([])
          return
        }
        if (sub && !sameKey(sub.options.applicationServerKey, key)) {
          await sub.unsubscribe()
          sub = null
        }
        if (!sub) sub = await reg.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: key as BufferSource })
        await post('/push/subscribe', { subscription: sub.toJSON(), topics: next })
        keepTopics(next)
        setTopics(next)
      }
      run()
        .catch((error) => {
          console.warn('NikkiBase notifications:', error)
          setFailed(true)
        })
        .finally(() => setPending(null))
    },
    [key, support, pending],
  )

  const status: Status =
    support === 'install'
      ? { kind: 'install' }
      : pending
        ? { kind: 'busy' }
        : failed
          ? { kind: 'failed', offline: !navigator.onLine }
          : denied
            ? { kind: 'denied' }
            : topics.length
              ? { kind: 'on', topics }
              : { kind: 'off' }

  return { available: key !== null && support !== 'no', topics, status, pending, apply }
}
