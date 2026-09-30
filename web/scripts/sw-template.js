const CACHE = __CACHE__
const PRECACHE = __PRECACHE__
const PREFIX = 'nikkibase-'

const lasting = (url) =>
  url.pathname.startsWith('/assets/') ||
  /^\/data\/[^/]+\/./.test(url.pathname) ||
  (url.pathname === '/keystream.bin' && url.searchParams.has('v'))

const cached = (request) => caches.open(CACHE).then((cache) => cache.match(request, { ignoreVary: true }))

async function networkFirst(request, fallback) {
  try {
    return await fetch(request)
  } catch (error) {
    const hit = (await cached(request)) ?? (fallback && (await cached(fallback)))
    if (hit) return hit
    throw error
  }
}

async function cacheFirst(event) {
  const hit = await cached(event.request)
  if (hit) return hit
  const response = await fetch(event.request)
  if (response.status === 200) {
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
  if (request.mode === 'navigate') event.respondWith(networkFirst(request, '/'))
  else if (lasting(url)) event.respondWith(cacheFirst(event))
  else event.respondWith(networkFirst(request))
})
