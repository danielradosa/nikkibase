import { createHash } from 'node:crypto'

const SIBLINGS = ['.br', '.gz']
const TOKENS = ['__CACHE__', '__PRECACHE__']

export function keystreamUrl(bytes) {
  if (!bytes) return '/keystream.bin'
  return `/keystream.bin?v=${createHash('sha256').update(bytes).digest('hex').slice(0, 16)}`
}

export function precacheList(files, { dataVersion, keystream }) {
  const present = new Set(files.map((file) => file.path))
  const entries = []
  for (const { path, size } of files) {
    const parts = path.split('/')
    if (path === 'sw.js' || parts.some((part) => part.startsWith('.'))) continue
    if (SIBLINGS.some((suffix) => path.endsWith(suffix) && present.has(path.slice(0, -suffix.length)))) continue
    if (parts[0] === 'data' && parts.length > 2 && parts[1] !== dataVersion) continue
    let url = '/' + parts.map(encodeURIComponent).join('/')
    if (path === 'index.html') url = '/'
    if (path === 'keystream.bin') url = keystream
    entries.push({ url, size })
  }
  return entries.sort((a, b) => (a.url < b.url ? -1 : a.url > b.url ? 1 : 0))
}

export function cacheName(entries, template) {
  const hash = createHash('sha256').update(template)
  for (const { url, size } of entries) hash.update(`\n${url} ${size}`)
  return `nikkibase-${hash.digest('hex').slice(0, 12)}`
}

export function renderWorker(template, cache, urls) {
  for (const token of TOKENS) {
    if (template.split(token).length !== 2) throw new Error(`sw: the template must use ${token} exactly once`)
  }
  const values = { __CACHE__: JSON.stringify(cache), __PRECACHE__: JSON.stringify(urls) }
  return template.replace(/__CACHE__|__PRECACHE__/g, (token) => values[token])
}
