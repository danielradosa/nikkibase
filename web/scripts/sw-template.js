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
