import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  DROP_HINT,
  ENGINE_WAIT,
  LIST_FAILED,
  LIST_WAIT,
  NO_FILE,
  SCORES_WAIT,
  TAGLINE,
  discardNotice,
  dropText,
  importError,
  importingText,
  loadedLabel,
  manyUnscored,
  stripNotes,
  unscored,
} from '../src/wardrobeText.ts'

const NOT_A_FILE = "That isn't a wardrobe file. Pick the file called clothes_date, or a selections file saved from Nikki Calc."
const EMPTY = 'That selections file has no items in it. Save it again from Nikki Calc.'
const UNREADABLE = "NikkiBase couldn't read this clothes_date file. Try again, or tick what you own in the Items tab."

test('an import error says what to do, in plain words', () => {
  for (const raw of [
    'wardrobe: not valid base64: illegal base64 data at input byte 4',
    'wardrobe: does not start with a Lua table',
    'wardrobe: file is 12 bytes, too short to hold a table',
    'wardrobe: not a selections file; no "@SEL" header',
  ]) {
    assert.equal(importError(new Error(raw)), NOT_A_FILE, raw)
  }
  assert.equal(importError(new Error('wardrobe: selections file lists no items')), EMPTY)
  for (const raw of [
    'wardrobe: record at byte 812 is ambiguous; keystream gap too wide',
    'wardrobe: no valid record at byte 40',
    'keystream not loaded',
    'keystream.bin: 404',
    'wardrobe: keystream too short',
  ]) {
    assert.equal(importError(new Error(raw)), UNREADABLE, raw)
  }
})

test('an error the list does not know is shown as it is', () => {
  const crash = 'The engine crashed and was restarted — import your wardrobe again.'
  assert.equal(importError(new Error(crash)), crash)
  assert.equal(importError('plain text'), 'plain text')
})

test('a discarded saved wardrobe is explained by how it was made', () => {
  const entry = (source: string) => ({ version: '2026-09-26', ids: [1], source, savedAt: 0 })
  assert.equal(
    discardNotice({ status: 'stale', entry: entry('manual') }),
    "NikkiBase's item data was updated, so your ticked items were cleared. Sorry! Tick them again in the Items tab.",
  )
  for (const source of ['sel', 'clothes_date']) {
    assert.equal(
      discardNotice({ status: 'stale', entry: entry(source) }),
      "NikkiBase's item data was updated, so your saved wardrobe was cleared. Import your file again.",
    )
  }
  assert.equal(discardNotice({ status: 'unrecognised', entry: entry('csv') }), "Your saved wardrobe couldn't be read, so it was cleared. Import it again.")
  assert.equal(discardNotice({ status: 'none' }), null)
  assert.equal(discardNotice({ status: 'ok', entry: { version: 'v', ids: [1], source: 'sel', savedAt: 0 } }), null)
})

test('a full warning is kept only when more than 2% of the wardrobe has no stats', () => {
  assert.equal(unscored(null), 0)
  assert.equal(unscored({ items: 3823, known: 3819, unresolved: 0 }), 4)
  assert.equal(manyUnscored(null), false)
  assert.equal(manyUnscored({ items: 100, known: 98, unresolved: 0 }), false)
  assert.equal(manyUnscored({ items: 100, known: 97, unresolved: 0 }), true)
  assert.equal(manyUnscored({ items: 0, known: 0, unresolved: 0 }), false)
})

test('the drop zone asks for the file in words that fit the device', () => {
  assert.equal(dropText(true), 'Tap to choose your clothes_date file')
  assert.equal(dropText(false), 'Drop or choose your clothes_date file')
  assert.equal(DROP_HINT, 'Or a Nikki Calc selections file. It stays on your device.')
  assert.equal(TAGLINE, 'your wardrobe stays on this device')
})

test('the no-file hint points at the Items tab in two short lines', () => {
  assert.equal(`${NO_FILE.before}${NO_FILE.link}${NO_FILE.after}`, 'No file? Tick what you own in the Items tab.')
  assert.equal(NO_FILE.link, 'Items tab')
  assert.equal(NO_FILE.saved, "It's saved in this browser.")
})

test('the loaded strip counts the wardrobe', () => {
  assert.equal(loadedLabel(3823), 'Wardrobe loaded: 3,823 items')
  assert.equal(loadedLabel(1), 'Wardrobe loaded: 1 item')
})

test('the loaded strip names each problem in a few words', () => {
  assert.deepEqual(stripNotes(null), [])
  assert.deepEqual(stripNotes({ items: 3823, known: 3823, unresolved: 0 }), [])
  assert.deepEqual(stripNotes({ items: 3823, known: 3819, unresolved: 0 }), ['4 have no stats'])
  assert.deepEqual(stripNotes({ items: 3823, known: 3822, unresolved: 0 }), ['1 has no stats'])
  assert.deepEqual(stripNotes({ items: 3823, known: 3819, unresolved: 12 }), ['4 have no stats', "12 couldn't be read"])
})

test('the loaded strip leaves the no-stats count to the warning when more than 2% have no stats', () => {
  assert.deepEqual(stripNotes({ items: 100, known: 97, unresolved: 0 }), [])
  assert.deepEqual(stripNotes({ items: 20000, known: 18765, unresolved: 1234 }), ["1,234 couldn't be read"])
})

test('while a wardrobe is read, the drop zone says so, and says when the engine has to start first', () => {
  assert.equal(importingText(true), 'Reading your wardrobe…')
  assert.equal(importingText(false), 'Starting the engine, then reading your wardrobe…')
  assert.equal(ENGINE_WAIT, 'Engine still loading · you can drop your file now')
})

test('the lists that are still loading say which one, and never promise a time', () => {
  assert.equal(LIST_WAIT, 'Loading the item list…')
  assert.equal(SCORES_WAIT, 'Loading the best possible scores…')
})

test('a failed item list says so instead of loading for ever', () => {
  assert.equal(LIST_FAILED, "The item list couldn't be loaded.")
})
