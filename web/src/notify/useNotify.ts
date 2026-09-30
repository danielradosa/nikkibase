import { useCallback, useEffect, useState } from 'react'
import { SYNC_KEY, TOPICS_KEY, isApple, keyBytes, needsSync, parseSync, parseTopics, sameKey, supportOf, type Status, type Support, type Topic } from './notify'

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

function keepSync(endpoint: string | null) {
  try {
    if (endpoint) localStorage.setItem(SYNC_KEY, JSON.stringify({ endpoint, at: Date.now() }))
    else localStorage.removeItem(SYNC_KEY)
  } catch {
  }
}

function savedSync() {
  try {
    return parseSync(localStorage.getItem(SYNC_KEY))
  } catch {
    return null
  }
}

async function post(path: string, body: unknown): Promise<number> {
  const res = await fetch(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })
  return res.status
}

async function register(reg: ServiceWorkerRegistration, key: Uint8Array, topics: Topic[]) {
  const fresh = () => reg.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: key as BufferSource })
  let sub = await reg.pushManager.getSubscription()
  let replaces = ''
  if (sub && !sameKey(sub.options.applicationServerKey, key)) {
    replaces = sub.endpoint
    await sub.unsubscribe()
    sub = null
  }
  if (!sub) sub = await fresh()
  let status = await post('/push/subscribe', { subscription: sub.toJSON(), topics, replaces })
  if (status === 400) {
    replaces = sub.endpoint
    await sub.unsubscribe()
    sub = await fresh()
    status = await post('/push/subscribe', { subscription: sub.toJSON(), topics, replaces })
  }
  if (status !== 204) throw new Error(`/push/subscribe answered ${status}`)
  keepSync(sub.endpoint)
}

async function unregister(reg: ServiceWorkerRegistration) {
  const sub = await reg.pushManager.getSubscription()
  if (sub) {
    await post('/push/unsubscribe', { endpoint: sub.endpoint }).catch(() => 0)
    await sub.unsubscribe()
  }
  keepSync(null)
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
    const sync = async () => {
      const reg = await registration()
      const sub = await reg.pushManager.getSubscription()
      const saved = savedTopics()
      if (!live) return
      if (saved.length === 0) {
        if (sub) await unregister(reg)
        return
      }
      if (Notification.permission !== 'granted') {
        keepTopics([])
        setTopics([])
        return
      }
      if (!sub || !sameKey(sub.options.applicationServerKey, key) || needsSync(savedSync(), sub.endpoint, Date.now())) {
        await register(reg, key, saved)
      }
    }
    sync().catch((error) => console.warn('NikkiBase notifications:', error))
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
        if (next.length === 0) await unregister(reg)
        else await register(reg, key, next)
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
