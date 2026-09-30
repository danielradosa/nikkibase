import { createHash } from 'node:crypto'
import { existsSync, readFileSync, readdirSync, statSync, writeFileSync } from 'node:fs'
import { join, sep } from 'node:path'
import { cacheName, keystreamUrl, precacheList, renderWorker } from './precache.mjs'

if (!existsSync('dist/index.html')) throw new Error('sw: run after vite build, from web/')

const files = []
for (const rel of readdirSync('dist', { recursive: true })) {
  const info = statSync(join('dist', rel))
  if (!info.isFile()) continue
  const hash = createHash('sha256').update(readFileSync(join('dist', rel))).digest('hex')
  files.push({ path: rel.split(sep).join('/'), size: info.size, hash })
}

const dataVersion = JSON.parse(readFileSync('dist/data/index.json', 'utf8')).version
if (!files.some((file) => file.path.startsWith(`data/${dataVersion}/`))) {
  throw new Error(`sw: dist/data/index.json points at ${dataVersion}, which is not in the build`)
}

const keystream = keystreamUrl(existsSync('dist/keystream.bin') ? readFileSync('dist/keystream.bin') : null)
const scripts = readdirSync('dist/assets').filter((name) => name.endsWith('.js'))
if (!scripts.some((name) => readFileSync(join('dist/assets', name), 'utf8').includes(keystream))) {
  throw new Error(`sw: no built script fetches ${keystream}, so the worker would keep the wrong keystream`)
}

const template = readFileSync(new URL('./sw-template.js', import.meta.url), 'utf8')
const entries = precacheList(files, { dataVersion, keystream })
const cache = cacheName(entries, template)
writeFileSync('dist/sw.js', renderWorker(template, cache, entries.map((entry) => entry.url)))

const bytes = entries.reduce((sum, entry) => sum + entry.size, 0)
console.log(`sw: ${entries.length} files, ${(bytes / 1048576).toFixed(1)} MB kept for offline use in ${cache}`)
