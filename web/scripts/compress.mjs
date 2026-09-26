import { existsSync, readFileSync, readdirSync, writeFileSync } from 'node:fs'
import { extname, join } from 'node:path'
import { brotliCompressSync, constants, gzipSync } from 'node:zlib'

const COMPRESSIBLE = new Set(['.js', '.css', '.json', '.wasm', '.bin', '.txt', '.svg'])
const MIN_BYTES = 1024
const MIN_SAVING = 0.1

if (!existsSync('dist')) throw new Error('compress: run after vite build, from web/')

function* files(dir) {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const path = join(dir, entry.name)
    if (entry.isDirectory()) yield* files(path)
    else yield path
  }
}

const brotli = (body) =>
  brotliCompressSync(body, {
    params: {
      [constants.BROTLI_PARAM_QUALITY]: constants.BROTLI_MAX_QUALITY,
      [constants.BROTLI_PARAM_SIZE_HINT]: body.length,
    },
  })

const gzip = (body) => gzipSync(body, { level: 9 })

let count = 0
let before = 0
let after = 0
for (const path of [...files('dist')]) {
  if (!COMPRESSIBLE.has(extname(path))) continue
  const body = readFileSync(path)
  if (body.length < MIN_BYTES) continue
  let smallest = body.length
  for (const [suffix, encode] of [['.br', brotli], ['.gz', gzip]]) {
    const encoded = encode(body)
    if (encoded.length > body.length * (1 - MIN_SAVING)) continue
    writeFileSync(path + suffix, encoded)
    smallest = Math.min(smallest, encoded.length)
  }
  if (smallest === body.length) continue
  count++
  before += body.length
  after += smallest
}

const mb = (n) => `${(n / 1048576).toFixed(1)} MB`
console.log(`compress: ${count} files, ${mb(before)} -> ${mb(after)}`)
