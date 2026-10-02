import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  CODE_EMPTY,
  CODE_INCOMPLETE,
  CODE_TEXT,
  BACKUP_EMPTY,
  FORGET_SAVE,
  IMPORT_CARDS,
  KEEP_CURRENT,
  ENGINE_DOWN,
  ENGINE_FAILED,
  ENGINE_WAIT,
  LIST_FAILED,
  LIST_WAIT,
  NOT_A_CODE,
  NO_FILE,
  READ_FAILED,
  SAVE_FAILED,
  SCORES_WAIT,
  TAGLINE,
  afterSave,
  backupLoad,
  codeLoad,
  droppedNotice,
  dropText,
  engineReason,
  importError,
  importingText,
  loadedLabel,
  restorePlan,
  manyUnscored,
  sourceLine,
  stripNotes,
  unscored,
  updatedWardrobe,
} from '../../src/wardrobe/wardrobeText.ts'

const NOT_A_FILE = "That isn't a wardrobe file. Pick the file called clothes_date, a selections file saved from Nikki Calc, or a wbak file."
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

test('pasted text with no code in it is told it is not a code, not asked for a file', () => {
  for (const raw of [
    'wardrobe: not valid base64: illegal base64 data at input byte 5',
    'wardrobe: does not start with a Lua table',
    'wardrobe: file is 12 bytes, too short to hold a table',
    'wardrobe: not a selections file; no "@SEL" header',
  ]) {
    assert.equal(importError(new Error(raw), true), NOT_A_CODE, raw)
  }
  assert.equal(importError(new Error('wardrobe: selections file lists no items'), true), EMPTY)
  assert.equal(importError(new Error('wardrobe: keystream too short'), true), UNREADABLE)
})

test('an error the list does not know is shown as it is', () => {
  const crash = 'The engine crashed and was restarted — import your wardrobe again.'
  assert.equal(importError(new Error(crash)), crash)
  assert.equal(importError('plain text'), 'plain text')
})

test('a wardrobe saved under older item data is kept and saved again, whatever its source, with its first save time', () => {
  const at = Date.UTC(2026, 9, 2, 9)
  const entry = (source: string, version = '2026-09-26') => ({ version, ids: [10001, 20001], source, savedAt: at })
  for (const source of ['sel', 'manual', 'clothes_date', 'nikkibase', 'wbak'] as const) {
    assert.deepEqual(restorePlan({ status: 'stale', entry: entry(source) }), {
      action: 'load',
      ids: [10001, 20001],
      source,
      savedAt: at,
      resave: true,
    })
    assert.deepEqual(restorePlan({ status: 'ok', entry: { ...entry(source, '2026-09-29'), source } }), {
      action: 'load',
      ids: [10001, 20001],
      source,
      savedAt: at,
      resave: false,
    })
  }
  for (const savedAt of [0, -1, Number.NaN, undefined]) {
    const plan = restorePlan({ status: 'ok', entry: { version: 'v', ids: [10001], source: 'sel', savedAt: savedAt as number } })
    assert.equal(plan.action === 'load' && plan.savedAt, null, String(savedAt))
  }
})

test('a saved wardrobe that cannot be understood is still cleared with its notice', () => {
  const entry = (source: string) => ({ version: '2026-09-26', ids: [1], source, savedAt: 0 })
  const cleared = { action: 'clear', notice: "Your saved wardrobe couldn't be read, so it was cleared. Import it again." }
  assert.deepEqual(restorePlan({ status: 'unrecognised', entry: entry('csv') }), cleared)
  assert.deepEqual(restorePlan({ status: 'stale', entry: entry('csv') }), cleared)
  assert.deepEqual(restorePlan({ status: 'unreadable' }), { action: 'warn', notice: READ_FAILED })
  assert.deepEqual(restorePlan({ status: 'none' }), { action: 'none' })
})

test('after a data update only the items still in the data are kept, and a notice counts the rest', () => {
  const known = new Set([10001, 20001, 30001])
  assert.deepEqual(updatedWardrobe([10001, 20001, 30001], known), { ids: [10001, 20001, 30001], notice: null })
  assert.deepEqual(updatedWardrobe([10001, 99999, 20001], known), {
    ids: [10001, 20001],
    notice: "NikkiBase's item data was updated. 1 of your items isn't in it any more.",
  })
  assert.deepEqual(updatedWardrobe([10001, 99997, 99998, 99999], known), {
    ids: [10001],
    notice: "NikkiBase's item data was updated. 3 of your items aren't in it any more.",
  })
  assert.equal(droppedNotice(0), null)
  assert.equal(droppedNotice(1234), "NikkiBase's item data was updated. 1,234 of your items aren't in it any more.")
})

test('a full warning is kept only when more than 2% of the wardrobe has no stats', () => {
  assert.equal(unscored(null), 0)
  assert.equal(unscored({ items: 3823, known: 3819, unresolved: 0 }), 4)
  assert.equal(manyUnscored(null), false)
  assert.equal(manyUnscored({ items: 100, known: 98, unresolved: 0 }), false)
  assert.equal(manyUnscored({ items: 100, known: 97, unresolved: 0 }), true)
  assert.equal(manyUnscored({ items: 0, known: 0, unresolved: 0 }), false)
})

test('each import card asks for its file in words that fit the device', () => {
  assert.equal(dropText(true), 'Tap to choose')
  assert.equal(dropText(false), 'Drop or choose')
  assert.deepEqual(
    IMPORT_CARDS.map((card) => [card.title, card.file]),
    [
      ['From the game', 'Your clothes_date file'],
      ['From Nikki Calc', 'A selections file or a wbak file'],
      ['From NikkiBase', 'Your NikkiBase wardrobe file'],
    ],
  )
  for (const card of IMPORT_CARDS) assert.doesNotMatch(`${card.title} ${card.file} ${card.hint}`, /info/i)
  assert.equal(KEEP_CURRENT, 'Keep this wardrobe')
  assert.equal(FORGET_SAVE, 'Save NikkiBase file first')
  assert.equal(TAGLINE, 'your wardrobe stays on this device')
})

test('the loaded strip says where the wardrobe came from and when', () => {
  const now = new Date(2026, 9, 2, 12)
  assert.deepEqual(sourceLine('clothes_date', new Date(2026, 9, 2, 9).getTime(), now), ['from clothes_date', '2 Oct'])
  assert.deepEqual(sourceLine('sel', null, now), ['from Nikki Calc'])
  assert.deepEqual(sourceLine('wbak', new Date(2025, 11, 31).getTime(), now), ['from a wbak file', '31 Dec 2025'])
  assert.deepEqual(sourceLine('nikkibase', null, now), ['from NikkiBase'])
  assert.deepEqual(sourceLine('manual', null, now), ['ticked by hand'])
  assert.deepEqual(sourceLine(null, null, now), [])
})

test('a wbak file loads the items NikkiBase knows and counts the rest', () => {
  const known = new Set([10022, 81001])
  assert.deepEqual(backupLoad({ ok: true, ids: [10022, 81001], unreadable: 0 }, known), { ok: true, ids: [10022, 81001], notice: null })
  assert.deepEqual(backupLoad({ ok: true, ids: [10022, 99999], unreadable: 2 }, known), {
    ok: true,
    ids: [10022],
    notice: "3 items in that wbak file aren't in NikkiBase's item data.",
  })
  assert.deepEqual(backupLoad({ ok: true, ids: [10022, 99999], unreadable: 0 }, known), {
    ok: true,
    ids: [10022],
    notice: "1 item in that wbak file isn't in NikkiBase's item data.",
  })
  assert.deepEqual(backupLoad({ ok: true, ids: [10022, 99999], unreadable: 0 }, null), { ok: true, ids: [10022, 99999], notice: null })
})

test('a wbak file with nothing to load, or that is not one, leaves the wardrobe alone', () => {
  assert.deepEqual(backupLoad({ ok: true, ids: [], unreadable: 4 }, null), { ok: false, error: BACKUP_EMPTY })
  assert.deepEqual(backupLoad({ ok: true, ids: [99999], unreadable: 0 }, new Set([10001])), { ok: false, error: BACKUP_EMPTY })
  assert.deepEqual(backupLoad({ ok: false }, null), { ok: false, error: NOT_A_FILE })
  assert.equal(BACKUP_EMPTY, 'That wbak file has no items NikkiBase knows.')
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

test('when the engine cannot start, the page says so plainly and says what to do', () => {
  assert.equal(ENGINE_FAILED, "The scoring engine couldn't start. Reload the page to try again.")
  assert.equal(ENGINE_DOWN.import, "Engine couldn't start · files can't be read")
  assert.equal(ENGINE_DOWN.outfit, "The engine couldn't start, so no outfit can be found.")
  assert.equal(ENGINE_DOWN.worth, "The engine couldn't start, so nothing can be ranked.")
})

test('the reason the engine gives drops the repeated Error prefixes', () => {
  assert.equal(engineReason(new Error('Error: items.bin: 404 Not Found')), 'items.bin: 404 Not Found')
  assert.equal(engineReason('Error: Error: WebAssembly.instantiate(): out of memory'), 'WebAssembly.instantiate(): out of memory')
  assert.equal(engineReason(new Error('the engine stopped')), 'the engine stopped')
  assert.equal(engineReason('Error: '), '')
  assert.equal(engineReason(new TypeError('Failed to fetch')), 'TypeError: Failed to fetch')
})

test('a failed save says the wardrobe will be gone, once for each run of failures', () => {
  assert.equal(SAVE_FAILED, "Couldn't save your wardrobe in this browser. It will be gone when you reload.")
  assert.equal(NO_FILE.unsaved, 'It will be gone when you reload.')
  assert.deepEqual(afterSave(false, true), { failing: false, notice: null })
  assert.deepEqual(afterSave(false, false), { failing: true, notice: SAVE_FAILED })
  assert.deepEqual(afterSave(true, false), { failing: true, notice: null })
  assert.deepEqual(afterSave(true, true), { failing: false, notice: null })
  let failing = false
  const shown: (string | null)[] = []
  for (const ok of [true, false, false, false, true, false]) {
    const next = afterSave(failing, ok)
    failing = next.failing
    shown.push(next.notice)
  }
  assert.deepEqual(shown, [null, SAVE_FAILED, null, null, null, SAVE_FAILED])
})

test('a storage that cannot be read says so', () => {
  assert.equal(READ_FAILED, "Couldn't read this browser's storage, so no saved wardrobe was loaded.")
})

test('a bad wardrobe code says plainly what is wrong with it', () => {
  assert.equal(CODE_INCOMPLETE, 'That code is incomplete. Copy it again, whole.')
  assert.equal(NOT_A_CODE, "That's not a NikkiBase wardrobe code.")
  assert.equal(CODE_EMPTY, 'That code has no items NikkiBase knows.')
  assert.deepEqual(codeLoad({ ok: false, problem: 'incomplete' }, null), { ok: false, error: CODE_INCOMPLETE })
  assert.deepEqual(codeLoad({ ok: false, problem: 'invalid' }, null), { ok: false, error: NOT_A_CODE })
})

test('a wardrobe code keeps the items the data knows and says how many of the rest it left out', () => {
  const known = new Set([10001, 20001, 30001])
  assert.deepEqual(codeLoad({ ok: true, ids: [10001, 20001] }, known), { ok: true, ids: [10001, 20001], notice: null })
  assert.deepEqual(codeLoad({ ok: true, ids: [10001, 20001, 99998, 99999] }, known), {
    ok: true,
    ids: [10001, 20001],
    notice: "2 items in that code aren't in NikkiBase's item data.",
  })
  assert.deepEqual(codeLoad({ ok: true, ids: [10001, 99999] }, known), {
    ok: true,
    ids: [10001],
    notice: "1 item in that code isn't in NikkiBase's item data.",
  })
  assert.deepEqual(codeLoad({ ok: true, ids: [10001, 99999] }, null), { ok: true, ids: [10001, 99999], notice: null })
})

test('a wardrobe code with nothing to load leaves the wardrobe alone', () => {
  assert.deepEqual(codeLoad({ ok: true, ids: [] }, null), { ok: false, error: CODE_EMPTY })
  assert.deepEqual(codeLoad({ ok: true, ids: [99998, 99999] }, new Set([10001])), { ok: false, error: CODE_EMPTY })
})

test('the save, copy, Nikki Calc and paste controls use short plain words', () => {
  assert.deepEqual(CODE_TEXT, {
    save: 'Save NikkiBase file',
    calc: 'Save for Nikki Calc',
    copy: 'Copy code',
    copied: 'Copied',
    paste: 'Paste a code',
    field: 'NikkiBase wardrobe code',
    placeholder: 'Paste your NikkiBase wardrobe code',
    load: 'Load',
    blocked: "Copying didn't work. Copy the code below by hand.",
  })
})
