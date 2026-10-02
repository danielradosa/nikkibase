import type { LoadResult, WardrobeSource } from './storage'
import type { BackupRead, CodeRead } from './wardrobeFile'

const NOT_A_FILE = "That isn't a wardrobe file. Pick the file called clothes_date, a selections file saved from Nikki Calc, or a wbak file."
const EMPTY = 'That selections file has no items in it. Save it again from Nikki Calc.'
const UNREADABLE = "NikkiBase couldn't read this clothes_date file. Try again, or tick what you own in the Items tab."

export function importError(e: unknown, pasted = false): string {
  const text = e instanceof Error ? e.message : String(e)
  if (text.includes('lists no items')) return EMPTY
  if (/keystream|ambiguous|no valid record/.test(text)) return UNREADABLE
  if (/not valid base64|Lua table|too short|not a selections file/.test(text)) return pasted ? NOT_A_CODE : NOT_A_FILE
  return text
}

const CLEARED = "Your saved wardrobe couldn't be read, so it was cleared. Import it again."

export type Restore =
  | { action: 'none' }
  | { action: 'load'; ids: number[]; source: WardrobeSource; savedAt: number | null; resave: boolean }
  | { action: 'clear'; notice: string }
  | { action: 'warn'; notice: string }

const knownSource = (source: string): source is WardrobeSource =>
  source === 'sel' || source === 'manual' || source === 'clothes_date' || source === 'nikkibase' || source === 'wbak'

const savedTime = (t: unknown): number | null => (typeof t === 'number' && Number.isFinite(t) && t > 0 ? t : null)

export function restorePlan(result: LoadResult): Restore {
  if (result.status === 'ok') {
    return { action: 'load', ids: result.entry.ids, source: result.entry.source, savedAt: savedTime(result.entry.savedAt), resave: false }
  }
  if (result.status === 'stale' && knownSource(result.entry.source)) {
    return { action: 'load', ids: result.entry.ids, source: result.entry.source, savedAt: savedTime(result.entry.savedAt), resave: true }
  }
  if (result.status === 'stale' || result.status === 'unrecognised') return { action: 'clear', notice: CLEARED }
  if (result.status === 'unreadable') return { action: 'warn', notice: READ_FAILED }
  return { action: 'none' }
}

export function droppedNotice(dropped: number): string | null {
  if (dropped <= 0) return null
  const count = dropped.toLocaleString('en-US')
  return `NikkiBase's item data was updated. ${count} of your items ${dropped === 1 ? "isn't" : "aren't"} in it any more.`
}

export function updatedWardrobe(ids: readonly number[], known: ReadonlySet<number>): { ids: number[]; notice: string | null } {
  const kept = ids.filter((id) => known.has(id))
  return { ids: kept, notice: droppedNotice(ids.length - kept.length) }
}

export const CODE_INCOMPLETE = 'That code is incomplete. Copy it again, whole.'

export const NOT_A_CODE = "That's not a NikkiBase wardrobe code."

export const CODE_EMPTY = 'That code has no items NikkiBase knows.'

export type CodeLoad = { ok: true; ids: number[]; notice: string | null } | { ok: false; error: string }

function codeLeftOut(count: number): string | null {
  if (count <= 0) return null
  const one = count === 1
  return `${count.toLocaleString('en-US')} ${one ? 'item' : 'items'} in that code ${one ? "isn't" : "aren't"} in NikkiBase's item data.`
}

export function codeLoad(read: CodeRead, known: ReadonlySet<number> | null): CodeLoad {
  if (!read.ok) return { ok: false, error: read.problem === 'incomplete' ? CODE_INCOMPLETE : NOT_A_CODE }
  const ids = known ? read.ids.filter((id) => known.has(id)) : read.ids
  return ids.length ? { ok: true, ids, notice: codeLeftOut(read.ids.length - ids.length) } : { ok: false, error: CODE_EMPTY }
}

export const BACKUP_EMPTY = 'That wbak file has no items NikkiBase knows.'

export function backupLoad(read: BackupRead, known: ReadonlySet<number> | null): CodeLoad {
  if (!read.ok) return { ok: false, error: NOT_A_FILE }
  const ids = known ? read.ids.filter((id) => known.has(id)) : read.ids
  if (!ids.length) return { ok: false, error: BACKUP_EMPTY }
  const left = read.unreadable + read.ids.length - ids.length
  if (left <= 0) return { ok: true, ids, notice: null }
  const one = left === 1
  return { ok: true, ids, notice: `${left.toLocaleString('en-US')} ${one ? 'item' : 'items'} in that wbak file ${one ? "isn't" : "aren't"} in NikkiBase's item data.` }
}

export const CODE_TEXT = {
  save: 'Save NikkiBase file',
  calc: 'Save for Nikki Calc',
  copy: 'Copy code',
  copied: 'Copied',
  paste: 'Paste a code',
  field: 'NikkiBase wardrobe code',
  placeholder: 'Paste your NikkiBase wardrobe code',
  load: 'Load',
  blocked: "Copying didn't work. Copy the code below by hand.",
}

export type Decoded = { items: number; known: number; unresolved: number }

export const UNSCORED_ALERT = 0.02

export const unscored = (d: Decoded | null) => (d ? d.items - d.known : 0)

export const manyUnscored = (d: Decoded | null) => !!d && d.items > 0 && unscored(d) / d.items > UNSCORED_ALERT

export const TAGLINE = 'your wardrobe stays on this device'

export type ImportCard = { key: 'game' | 'calc' | 'nikkibase'; title: string; file: string; hint: string }

export const IMPORT_CARDS: readonly ImportCard[] = [
  { key: 'game', title: 'From the game', file: 'Your clothes_date file', hint: "It's in the game's usrdat folder. Take the largest one." },
  { key: 'calc', title: 'From Nikki Calc', file: 'A selections file or a wbak file', hint: "Make one with Generate file on Nikki Calc's Manual page." },
  { key: 'nikkibase', title: 'From NikkiBase', file: 'Your NikkiBase wardrobe file', hint: 'Saved here before, on this or another device.' },
]

export const KEEP_CURRENT = 'Keep this wardrobe'

const SOURCE_LABEL: Readonly<Record<WardrobeSource, string>> = {
  clothes_date: 'from clothes_date',
  sel: 'from Nikki Calc',
  wbak: 'from a wbak file',
  nikkibase: 'from NikkiBase',
  manual: 'ticked by hand',
}

export function sourceLine(source: WardrobeSource | null, savedAt: number | null, now = new Date()): string[] {
  const parts: string[] = []
  if (source) parts.push(SOURCE_LABEL[source])
  if (savedAt) {
    const when = new Date(savedAt)
    const sameYear = when.getFullYear() === now.getFullYear()
    parts.push(when.toLocaleDateString('en-GB', sameYear ? { day: 'numeric', month: 'short' } : { day: 'numeric', month: 'short', year: 'numeric' }))
  }
  return parts
}

export const NO_FILE = {
  before: 'No file? Tick what you own in the ',
  link: 'Items tab',
  after: '.',
  saved: "It's saved in this browser.",
  unsaved: 'It will be gone when you reload.',
}

export const SAVE_FAILED = "Couldn't save your wardrobe in this browser. It will be gone when you reload."

export const READ_FAILED = "Couldn't read this browser's storage, so no saved wardrobe was loaded."

export function afterSave(failing: boolean, ok: boolean): { failing: boolean; notice: string | null } {
  if (ok) return { failing: false, notice: null }
  return { failing: true, notice: failing ? null : SAVE_FAILED }
}

export function dropText(phone: boolean): string {
  return phone ? 'Tap to choose' : 'Drop or choose'
}

export function importingText(engineReady: boolean): string {
  return engineReady ? 'Reading your wardrobe…' : 'Starting the engine, then reading your wardrobe…'
}

export const ENGINE_WAIT = 'Engine still loading · you can drop your file now'

export const ENGINE_FAILED = "The scoring engine couldn't start. Reload the page to try again."

export const ENGINE_DOWN = {
  import: "Engine couldn't start · files can't be read",
  outfit: "The engine couldn't start, so no outfit can be found.",
  worth: "The engine couldn't start, so nothing can be ranked.",
}

export function engineReason(e: unknown): string {
  return String(e).replace(/^(?:\s*Error:\s*)+/, '').trim()
}

export const LIST_WAIT = 'Loading the item list…'

export const LIST_FAILED = "The item list couldn't be loaded."

export const SCORES_WAIT = 'Loading the best possible scores…'

export function loadedLabel(count: number): string {
  return `Wardrobe loaded: ${count.toLocaleString('en-US')} ${count === 1 ? 'item' : 'items'}`
}

export function stripNotes(d: Decoded | null): string[] {
  if (!d) return []
  const notes: string[] = []
  const missing = unscored(d)
  if (missing > 0 && !manyUnscored(d)) notes.push(`${missing.toLocaleString('en-US')} ${missing === 1 ? 'has' : 'have'} no stats`)
  if (d.unresolved > 0) notes.push(`${d.unresolved.toLocaleString('en-US')} couldn't be read`)
  return notes
}
