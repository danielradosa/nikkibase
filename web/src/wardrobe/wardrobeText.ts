import type { LoadResult, WardrobeSource } from './storage'
import type { CodeRead } from './wardrobeFile'

const NOT_A_FILE = "That isn't a wardrobe file. Pick the file called clothes_date, or a selections file saved from Nikki Calc."
const EMPTY = 'That selections file has no items in it. Save it again from Nikki Calc.'
const UNREADABLE = "NikkiBase couldn't read this clothes_date file. Try again, or tick what you own in the Items tab."

export function importError(e: unknown): string {
  const text = e instanceof Error ? e.message : String(e)
  if (text.includes('lists no items')) return EMPTY
  if (/keystream|ambiguous|no valid record/.test(text)) return UNREADABLE
  if (/not valid base64|Lua table|too short|not a selections file/.test(text)) return NOT_A_FILE
  return text
}

const CLEARED = "Your saved wardrobe couldn't be read, so it was cleared. Import it again."

const knownSource = (source: string): source is WardrobeSource =>
  source === 'sel' || source === 'manual' || source === 'clothes_date' || source === 'nikkibase'

export type Restore =
  | { action: 'none' }
  | { action: 'load'; ids: number[]; source: WardrobeSource; resave: boolean }
  | { action: 'clear'; notice: string }
  | { action: 'warn'; notice: string }

export function restorePlan(result: LoadResult): Restore {
  if (result.status === 'ok') return { action: 'load', ids: result.entry.ids, source: result.entry.source, resave: false }
  if (result.status === 'stale' && knownSource(result.entry.source)) {
    return { action: 'load', ids: result.entry.ids, source: result.entry.source, resave: true }
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

export function codeLoad(read: CodeRead, known: ReadonlySet<number> | null): CodeLoad {
  if (!read.ok) return { ok: false, error: read.problem === 'incomplete' ? CODE_INCOMPLETE : NOT_A_CODE }
  const kept = known ? updatedWardrobe(read.ids, known) : { ids: read.ids, notice: null }
  return kept.ids.length ? { ok: true, ...kept } : { ok: false, error: CODE_EMPTY }
}

export const CODE_TEXT = {
  save: 'Save a copy',
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

export const DROP_HINT = 'Or a Nikki Calc selections file, or a NikkiBase wardrobe file. It stays on your device.'

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
  return phone ? 'Tap to choose your clothes_date file' : 'Drop or choose your clothes_date file'
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
