import type { LoadResult } from './storage'

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

export function discardNotice(result: LoadResult): string | null {
  if (result.status === 'stale') {
    return result.entry.source === 'manual'
      ? "NikkiBase's item data was updated, so your ticked items were cleared. Sorry! Tick them again in the Items tab."
      : "NikkiBase's item data was updated, so your saved wardrobe was cleared. Import your file again."
  }
  if (result.status === 'unrecognised') return "Your saved wardrobe couldn't be read, so it was cleared. Import it again."
  return null
}

export type Decoded = { items: number; known: number; unresolved: number }

export const UNSCORED_ALERT = 0.02

export const unscored = (d: Decoded | null) => (d ? d.items - d.known : 0)

export const manyUnscored = (d: Decoded | null) => !!d && d.items > 0 && unscored(d) / d.items > UNSCORED_ALERT

export const TAGLINE = 'your wardrobe stays on this device'

export const DROP_HINT = 'Or a Nikki Calc selections file. It stays on your device.'

export const NO_FILE = {
  before: 'No file? Tick what you own in the ',
  link: 'Items tab',
  after: '.',
  saved: "It's saved in this browser.",
}

export function dropText(phone: boolean): string {
  return phone ? 'Tap to choose your clothes_date file' : 'Drop or choose your clothes_date file'
}

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
