import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { runInNewContext } from 'node:vm'
import { cacheName, keystreamUrl, precacheList, renderWorker } from '../../scripts/precache.mjs'

const template = readFileSync(new URL('../../scripts/sw-template.js', import.meta.url), 'utf8')
const KEYSTREAM = '/keystream.bin?v=2c70e12b7a0646f9'
const ORIGIN = 'https://nikkibase.test'

const dist = (
  [
    ['index.html', 900],
    ['assets/index-abc.js', 5000],
    ['assets/index-abc.js.br', 1200],
    ['assets/index-abc.js.gz', 1500],
    ['assets/nikkibase-def.wasm', 90000],
    ['assets/nikkibase-def.wasm.br', 30000],
    ['assets/worker-ghi.js', 3000],
    ['data/index.json', 20],
    ['data/cloud/items.json', 2600],
    ['data/cloud/items.json.gz', 400],
    ['data/cloud/stages.json', 66],
    ['data/lilith/items.json', 2601],
    ['data/lilith/stages.json', 66],
    ['keystream.bin', 70],
    ['keystream.bin.br', 60],
    ['credits/wiki-contributors.json', 10],
    ['favicon.ico', 1400],
    ['manifest.webmanifest', 600],
    ['icons/icon-192.png', 6800],
    ['notes.gz', 30],
    ['sw.js', 4000],
    ['sw.js.br', 1000],
    ['.DS_Store', 6000],
    ['data/.DS_Store', 6000],
  ] as [string, number][]
).map(([path, size]) => ({ path, size }))

const entries = precacheList(dist, { dataVersion: 'cloud', keystream: KEYSTREAM })
const urls = entries.map((entry: { url: string }) => entry.url)

test('the keystream URL is its sha256 cut to 16 hex digits, like the page asks for it', () => {
  assert.equal(keystreamUrl(Buffer.from('key')), KEYSTREAM)
  assert.equal(keystreamUrl(null), '/keystream.bin')
})

test('every built file is kept once, the page as / and the keystream at its versioned URL', () => {
  assert.deepEqual(urls, [
    '/',
    '/assets/index-abc.js',
    '/assets/nikkibase-def.wasm',
    '/assets/worker-ghi.js',
    '/credits/wiki-contributors.json',
    '/data/cloud/items.json',
    '/data/cloud/stages.json',
    '/data/index.json',
    '/favicon.ico',
    '/icons/icon-192.png',
    KEYSTREAM,
    '/manifest.webmanifest',
    '/notes.gz',
  ])
})

test('compressed copies, older data, the worker itself and hidden files are left out', () => {
  for (const url of urls) {
    assert.doesNotMatch(url, /\.br$|\.js\.gz$|\.json\.gz$|lilith|\/sw\.js|DS_Store|index\.html/)
  }
})

test('each kept file carries its size', () => {
  assert.equal(entries.find((entry: { url: string }) => entry.url === '/assets/nikkibase-def.wasm').size, 90000)
  assert.equal(entries.find((entry: { url: string }) => entry.url === '/').size, 900)
})

test('the cache name changes with the files, their sizes and the worker, not with listing order', () => {
  const name = cacheName(entries, template)
  assert.match(name, /^nikkibase-[0-9a-f]{12}$/)
  assert.equal(cacheName(precacheList([...dist].reverse(), { dataVersion: 'cloud', keystream: KEYSTREAM }), template), name)
  const grown = dist.map((file) => (file.path === 'favicon.ico' ? { ...file, size: 1401 } : file))
  assert.notEqual(cacheName(precacheList(grown, { dataVersion: 'cloud', keystream: KEYSTREAM }), template), name)
  const renamed = dist.map((file) => (file.path === 'assets/index-abc.js' ? { ...file, path: 'assets/index-xyz.js' } : file))
  assert.notEqual(cacheName(precacheList(renamed, { dataVersion: 'cloud', keystream: KEYSTREAM }), template), name)
  assert.notEqual(cacheName(precacheList(dist, { dataVersion: 'lilith', keystream: KEYSTREAM }), template), name)
  assert.notEqual(cacheName(precacheList(dist, { dataVersion: 'cloud', keystream: '/keystream.bin?v=1' }), template), name)
  assert.notEqual(cacheName(entries, template + '\n'), name)
})

test('the worker gets its cache name and list, and a template without them is refused', () => {
  const code = renderWorker(template, 'nikkibase-abc', ['/', '/a$&.js'])
  assert.match(code, /const CACHE = "nikkibase-abc"/)
  assert.match(code, /const PRECACHE = \["\/","\/a\$&\.js"\]/)
  assert.doesNotMatch(code, /__CACHE__|__PRECACHE__/)
  assert.throws(() => renderWorker('const CACHE = __CACHE__', 'x', []), /__PRECACHE__/)
})

test('the worker never takes over open pages early', () => {
  assert.doesNotMatch(template, /skipWaiting|clients\.claim/)
})

type Req = { url: string; method: string; mode: string }

const req = (path: string, mode = 'cors', method = 'GET'): Req => ({ url: new URL(path, ORIGIN).href, method, mode })

type Body = string | { body: string; type: string }

function worker(site: Record<string, Body>, kept = urls, name = 'nikkibase-new') {
  const stores = new Map<string, Map<string, Response>>()
  const key = (r: string | Req) => new URL(typeof r === 'string' ? r : r.url, ORIGIN).href
  const net = { online: true, calls: [] as string[] }
  const fetch = async (r: string | Req) => {
    const url = new URL(key(r))
    net.calls.push(url.pathname + url.search)
    if (!net.online) throw new TypeError('Failed to fetch')
    const body = site[url.pathname + url.search]
    if (body === undefined) return new Response('missing', { status: 404 })
    return typeof body === 'string' ? new Response(body) : new Response(body.body, { headers: { 'content-type': body.type } })
  }
  const open = async (name: string) => {
    if (!stores.has(name)) stores.set(name, new Map())
    const store = stores.get(name)!
    return {
      match: async (r: string | Req) => store.get(key(r))?.clone(),
      put: async (r: string | Req, response: Response) => {
        store.set(key(r), response)
      },
      addAll: async (list: string[]) => {
        const got = await Promise.all(list.map(async (url) => [key(url), await fetch(url)] as const))
        if (got.some(([, response]) => !response.ok)) throw new TypeError('addAll: a request failed')
        for (const [url, response] of got) store.set(url, response)
      },
    }
  }
  const match = async (r: string | Req, options: { cacheName: string }) => stores.get(options.cacheName)?.get(key(r))?.clone()
  const caches = { open, match, keys: async () => [...stores.keys()], delete: async (name: string) => stores.delete(name) }
  const listeners = new Map<string, (event: object) => void>()
  const self = { location: new URL('/sw.js', ORIGIN), addEventListener: (type: string, fn: (event: object) => void) => listeners.set(type, fn) }
  runInNewContext(renderWorker(template, name, kept), { self, caches, fetch, URL, Response })

  const lifecycle = (type: string) => {
    let done: Promise<unknown> = Promise.resolve()
    listeners.get(type)!({ waitUntil: (p: Promise<unknown>) => (done = p) })
    return done
  }
  const get = async (r: Req) => {
    let responded: Promise<Response> | null = null
    const waits: Promise<unknown>[] = []
    listeners.get('fetch')!({ request: r, respondWith: (p: Promise<Response>) => (responded = p), waitUntil: (p: Promise<unknown>) => waits.push(p) })
    if (!responded) return null
    const response = await responded
    await Promise.all(waits)
    return response
  }
  const text = async (r: Req) => (await get(r))!.text()
  return { stores, net, open, install: () => lifecycle('install'), activate: () => lifecycle('activate'), get, text }
}

const site = () => Object.fromEntries(urls.map((url: string) => [url, `v1 ${url}`]))

test('installing keeps every listed file in the new cache', async () => {
  const sw = worker(site())
  await sw.install()
  assert.deepEqual([...sw.stores.keys()], ['nikkibase-new'])
  assert.equal(sw.stores.get('nikkibase-new')!.size, urls.length)
})

test('one file that cannot be fetched fails the whole install', async () => {
  const files = site()
  delete files['/data/cloud/items.json']
  const sw = worker(files)
  await assert.rejects(sw.install())
  assert.equal(sw.stores.get('nikkibase-new')!.size, 0)
})

test('activating drops older NikkiBase caches and nothing else', async () => {
  const sw = worker(site())
  await sw.open('nikkibase-old')
  await sw.open('someone-else')
  await sw.install()
  await sw.activate()
  assert.deepEqual([...sw.stores.keys()].sort(), ['nikkibase-new', 'someone-else'])
})

test('opening a page asks the network first and falls back to the kept page offline', async () => {
  const files = site()
  const sw = worker(files)
  await sw.install()
  files['/'] = 'v2 shell'
  files['/stages'] = 'v2 shell'
  assert.equal(await sw.text(req('/stages', 'navigate')), 'v2 shell')
  sw.net.online = false
  assert.equal(await sw.text(req('/', 'navigate')), 'v1 /')
  assert.equal(await sw.text(req('/stages', 'navigate')), 'v1 /')
  assert.equal(await sw.text(req('/privacy.txt?x=1', 'navigate')), 'v1 /')
})

test('offline, a page opens from the newest complete copy, never from an older one', async () => {
  const sw = worker(site(), urls, 'nikkibase-a')
  await sw.install()
  const newer = await sw.open('nikkibase-b')
  await newer.put('/', new Response('v2 /'))
  await newer.put('/assets/index-v2.js', new Response('v2 js'))
  await newer.put('/data/newer/items.json', new Response('v2 data'))
  await newer.put('/manifest.webmanifest', new Response('v2 manifest'))
  await sw.open('nikkibase-c')
  const other = await sw.open('someone-else')
  await other.put('/', new Response('not ours'))
  sw.net.online = false
  assert.equal(await sw.text(req('/', 'navigate')), 'v2 /')
  assert.equal(await sw.text(req('/stages', 'navigate')), 'v2 /')
  assert.equal(await sw.text(req('/assets/index-v2.js')), 'v2 js')
  assert.equal(await sw.text(req('/data/newer/items.json')), 'v2 data')
  assert.equal(await sw.text(req('/manifest.webmanifest')), 'v2 manifest')
  assert.equal(await sw.text(req('/assets/index-abc.js')), 'v1 /assets/index-abc.js')
})

test('a newer copy is used for built files online too, without asking the network', async () => {
  const sw = worker(site(), urls, 'nikkibase-a')
  await sw.install()
  const newer = await sw.open('nikkibase-b')
  await newer.put('/assets/index-v2.js', new Response('v2 js'))
  sw.net.calls.length = 0
  assert.equal(await sw.text(req('/assets/index-v2.js')), 'v2 js')
  assert.deepEqual(sw.net.calls, [])
  assert.equal(sw.stores.get('nikkibase-a')!.has(new URL('/assets/index-v2.js', ORIGIN).href), false)
})

test('built files, data and the keystream come from the cache even online', async () => {
  const files = site()
  const sw = worker(files)
  await sw.install()
  sw.net.calls.length = 0
  for (const path of ['/assets/index-abc.js', '/assets/nikkibase-def.wasm', '/data/cloud/items.json', KEYSTREAM]) {
    files[path] = 'changed'
    assert.equal(await sw.text(req(path)), `v1 ${path}`)
  }
  assert.deepEqual(sw.net.calls, [])
})

test('a lasting file that is not kept yet is fetched once and then kept', async () => {
  const files = { ...site(), '/data/lilith/items.json': 'old data', '/keystream.bin?v=old': 'old key' }
  const sw = worker(files)
  await sw.install()
  assert.equal(await sw.text(req('/data/lilith/items.json')), 'old data')
  assert.equal(await sw.text(req('/keystream.bin?v=old')), 'old key')
  assert.equal((await sw.get(req('/assets/missing.js')))!.status, 404)
  sw.net.online = false
  assert.equal(await sw.text(req('/data/lilith/items.json')), 'old data')
  assert.equal(await sw.text(req('/keystream.bin?v=old')), 'old key')
  await assert.rejects(sw.get(req('/assets/missing.js')))
})

test('a page sent in place of a built file, data or the keystream is passed on but never kept', async () => {
  const page = { body: '<!doctype html><title>NikkiBase</title>', type: 'text/html; charset=utf-8' }
  const paths = ['/data/lilith/tags.json', '/assets/index-gone.js', '/keystream.bin?v=gone']
  const sw = worker({ ...site(), ...Object.fromEntries(paths.map((path) => [path, page])) })
  await sw.install()
  for (const path of paths) assert.equal(await sw.text(req(path)), page.body)
  sw.net.online = false
  for (const path of paths) await assert.rejects(sw.get(req(path)), path)
})

test('everything else asks the network first and uses the kept copy offline', async () => {
  const files = site()
  const sw = worker(files)
  await sw.install()
  files['/data/index.json'] = 'v2 index'
  files['/keystream.bin'] = 'plain key'
  assert.equal(await sw.text(req('/data/index.json')), 'v2 index')
  assert.equal(await sw.text(req('/keystream.bin')), 'plain key')
  sw.net.online = false
  assert.equal(await sw.text(req('/data/index.json')), 'v1 /data/index.json')
  assert.equal(await sw.text(req('/manifest.webmanifest')), 'v1 /manifest.webmanifest')
  await assert.rejects(sw.get(req('/keystream.bin')))
  await assert.rejects(sw.get(req('/nothing.txt')))
})

test('other sites and anything but GET are left to the browser', async () => {
  const sw = worker(site())
  await sw.install()
  assert.equal(await sw.get({ url: 'https://elsewhere.test/assets/x.js', method: 'GET', mode: 'cors' }), null)
  assert.equal(await sw.get(req('/', 'navigate', 'POST')), null)
  assert.equal(await sw.get(req('/assets/index-abc.js', 'cors', 'HEAD')), null)
})
