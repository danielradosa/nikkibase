const CACHE = __CACHE__
const PRECACHE = __PRECACHE__
const PREFIX = 'nikkibase-'
const SLOW_PAGE = 3500

const lasting = (url) =>
  url.pathname.startsWith('/assets/') ||
  /^\/data\/[^/]+\/./.test(url.pathname) ||
  (url.pathname === '/keystream.bin' && url.searchParams.has('v'))

async function cached(request) {
  const names = (await caches.keys()).filter((key) => key.startsWith(PREFIX)).reverse()
  for (const cacheName of names) {
    const hit = await caches.match(request, { cacheName, ignoreVary: true })
    if (hit) return hit
  }
}

async function networkFirst(request) {
  try {
    return await fetch(request)
  } catch (error) {
    const hit = await cached(request)
    if (hit) return hit
    throw error
  }
}

async function openPage(request) {
  const network = fetch(request)
  network.catch(() => {})
  const slow = new Promise((resolve) => setTimeout(resolve, SLOW_PAGE))
  try {
    const answer = await Promise.race([network, slow])
    if (answer) return answer
  } catch {}
  return (await cached(request)) ?? (await cached('/')) ?? network
}

async function cacheFirst(event) {
  const hit = await cached(event.request)
  if (hit) return hit
  const response = await fetch(event.request)
  if (response.status === 200 && !response.headers.get('content-type')?.startsWith('text/html')) {
    const copy = response.clone()
    event.waitUntil(
      caches
        .open(CACHE)
        .then((cache) => cache.put(event.request, copy))
        .catch(() => {}),
    )
  }
  return response
}

self.addEventListener('install', (event) => {
  event.waitUntil(caches.open(CACHE).then((cache) => cache.addAll(PRECACHE)))
})

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((key) => key.startsWith(PREFIX) && key !== CACHE).map((key) => caches.delete(key)))),
  )
})

self.addEventListener('fetch', (event) => {
  const { request } = event
  if (request.method !== 'GET') return
  const url = new URL(request.url)
  if (url.origin !== self.location.origin) return
  if (request.mode === 'navigate') event.respondWith(openPage(request))
  else if (lasting(url)) event.respondWith(cacheFirst(event))
  else event.respondWith(networkFirst(request))
})

const text = (value, fallback) => (typeof value === 'string' && value ? value : fallback)

self.addEventListener('push', (event) => {
  let message = {}
  try {
    message = event.data ? event.data.json() : {}
  } catch {}
  const url = text(message.url, '/')
  event.waitUntil(
    self.registration.showNotification(text(message.title, 'NikkiBase'), {
      body: text(message.body, ''),
      tag: text(message.tag, 'nikkibase-data'),
      icon: '/icons/icon-192.png',
      data: { url: url.startsWith('/') && !url.startsWith('//') ? url : '/' },
    }),
  )
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  const url = new URL(event.notification.data?.url ?? '/', self.location.origin).href
  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((windows) => {
      const open = windows.find((client) => new URL(client.url).origin === self.location.origin)
      return open ? open.focus() : self.clients.openWindow(url)
    }),
  )
})

self.addEventListener('pushsubscriptionchange', (event) => {
  const old = event.oldSubscription
  const renew = async () => {
    const key = old?.options?.applicationServerKey
    const next = event.newSubscription ?? (key ? await self.registration.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: key }) : null)
    if (!next || !old) return
    await fetch('/push/subscribe', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ subscription: next.toJSON(), replaces: old.endpoint }),
    })
  }
  event.waitUntil(renew().catch(() => {}))
})
