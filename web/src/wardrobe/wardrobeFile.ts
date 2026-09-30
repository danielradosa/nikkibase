export const CODE_PREFIX = 'NB1.'
export const MAX_ID = 9_999_999
export const MAX_IDS = 100_000

export type CodeProblem = 'none' | 'incomplete' | 'invalid'
export type CodeRead = { ok: true; ids: number[] } | { ok: false; problem: CodeProblem }

const ALPHABET = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_'
const VALUES = new Int8Array(128).fill(-1)
for (let i = 0; i < ALPHABET.length; i++) VALUES[ALPHABET.charCodeAt(i)] = i
const payloadValue = (code: number) => (code < 128 ? VALUES[code] : -1)
const IGNORED = /[\s\u00ad\u200b-\u200d\u2060]/
const LAST_SHIFT = 21

const incomplete: CodeRead = { ok: false, problem: 'incomplete' }
const invalid: CodeRead = { ok: false, problem: 'invalid' }

function tidy(ids: readonly number[]): number[] {
  const kept = ids.filter((id) => Number.isInteger(id) && id > 0 && id <= MAX_ID)
  return [...new Set(kept)].sort((a, b) => a - b)
}

function varints(sorted: readonly number[]): number[] {
  const bytes: number[] = []
  let previous = 0
  for (const id of sorted) {
    let gap = id - previous
    previous = id
    while (gap >= 0x80) {
      bytes.push((gap & 0x7f) | 0x80)
      gap >>>= 7
    }
    bytes.push(gap)
  }
  return bytes
}

function base64url(bytes: readonly number[]): string {
  const out: string[] = []
  for (let i = 0; i < bytes.length; i += 3) {
    const size = Math.min(3, bytes.length - i)
    const block = (bytes[i] << 16) | ((bytes[i + 1] ?? 0) << 8) | (bytes[i + 2] ?? 0)
    for (let k = 0; k <= size; k++) out.push(ALPHABET[(block >> (18 - 6 * k)) & 63])
  }
  return out.join('')
}

const codeOf = (sorted: readonly number[]) => `${CODE_PREFIX}${sorted.length}.${base64url(varints(sorted))}`

export const wardrobeCode = (ids: readonly number[]): string => codeOf(tidy(ids))

function day(date: Date): string {
  const two = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${two(date.getMonth() + 1)}-${two(date.getDate())}`
}

export const fileName = (date: Date): string => `nikkibase-wardrobe-${day(date)}.txt`

export function wardrobeFile(ids: readonly number[], saved: Date, host: string): string {
  const sorted = tidy(ids)
  const count = sorted.length
  return [
    `NikkiBase wardrobe · ${count.toLocaleString('en-US')} ${count === 1 ? 'item' : 'items'} · saved ${day(saved)}`,
    codeOf(sorted),
    `Drop this file on ${host} or paste the code there to load it.`,
    '',
  ].join('\n')
}

export function readWardrobeCode(text: string): CodeRead {
  const start = text.indexOf(CODE_PREFIX)
  if (start < 0) return { ok: false, problem: 'none' }
  let at = start + CODE_PREFIX.length
  const digits = /^\d*/.exec(text.slice(at, at + 32))![0]
  at += digits.length
  if (text.slice(at).trim() === '') return incomplete
  if (!digits || text[at] !== '.') return invalid
  const count = Number(digits)
  if (count > MAX_IDS) return invalid
  at++

  const ids: number[] = []
  let bits = 0
  let pending = 0
  let gap = 0
  let shift = 0
  let previous = 0
  let wrapped = false
  const broken = () => (wrapped ? incomplete : invalid)
  while (ids.length < count) {
    while (bits < 8) {
      if (at >= text.length) return incomplete
      const char = text[at++]
      const value = payloadValue(char.charCodeAt(0))
      if (value < 0) {
        if (!IGNORED.test(char)) return incomplete
        wrapped = true
        continue
      }
      pending = (pending << 6) | value
      bits += 6
    }
    bits -= 8
    const byte = pending >> bits
    pending &= (1 << bits) - 1
    if (shift > 0 && byte === 0) return broken()
    gap += (byte & 0x7f) * 2 ** shift
    if (byte & 0x80) {
      if (shift === LAST_SHIFT) return broken()
      shift += 7
      continue
    }
    const id = previous + gap
    if (gap === 0 || id > MAX_ID) return broken()
    ids.push(id)
    previous = id
    gap = 0
    shift = 0
  }
  if (pending !== 0 || payloadValue(text.charCodeAt(at)) >= 0) return broken()
  return { ok: true, ids }
}
