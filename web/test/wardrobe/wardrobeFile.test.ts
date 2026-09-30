import { test } from 'node:test'
import assert from 'node:assert/strict'
import { MAX_ID, MAX_IDS, fileName, readWardrobeCode, wardrobeCode, wardrobeFile, wardrobeKind } from '../../src/wardrobe/wardrobeFile.ts'

function varints(values: number[]): number[] {
  const bytes: number[] = []
  for (let v of values) {
    while (v >= 0x80) {
      bytes.push((v % 0x80) | 0x80)
      v = Math.floor(v / 0x80)
    }
    bytes.push(v)
  }
  return bytes
}

const payload = (bytes: number[]) => Buffer.from(bytes).toString('base64url')

function seeded(seed: number) {
  let a = seed >>> 0
  return () => {
    a = (a + 0x6d2b79f5) >>> 0
    let t = a
    t = Math.imul(t ^ (t >>> 15), t | 1)
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

function realIds(count: number): number[] {
  const random = seeded(20260930)
  const ranges: [number, number][] = [
    [10001, 99999],
    [180001, 189999],
    [880001, 880180],
  ]
  const ids = new Set([880024, 188198])
  while (ids.size < count) {
    const [low, high] = ranges[Math.floor(random() * ranges.length)]
    ids.add(low + Math.floor(random() * (high - low + 1)))
  }
  return [...ids]
}

const read = (text: string) => readWardrobeCode(text)
const ids = (text: string) => {
  const result = read(text)
  assert.ok(result.ok, `expected a code in ${JSON.stringify(text.slice(0, 80))}, got ${JSON.stringify(result)}`)
  return result.ids
}

test('an empty wardrobe makes an empty code and reads back empty', () => {
  assert.equal(wardrobeCode([]), 'NB1.0.')
  assert.deepEqual(ids('NB1.0.'), [])
})

test('one item makes a short code that reads back', () => {
  assert.equal(wardrobeCode([10001]), 'NB1.1.kU4')
  assert.deepEqual(ids('NB1.1.kU4'), [10001])
})

test('the payload is base64url without padding of the varint gaps between sorted ids', () => {
  const list = [10001, 10002, 10130, 188198, 880024]
  const gaps = list.map((id, i) => id - (i ? list[i - 1] : 0))
  assert.equal(wardrobeCode(list), `NB1.5.${payload(varints(gaps))}`)
  assert.doesNotMatch(wardrobeCode(list), /[=+/]/)
})

test('10,000 ids from the real id range survive the round trip', () => {
  const list = realIds(10000)
  const sorted = [...list].sort((a, b) => a - b)
  const code = wardrobeCode(list)
  assert.match(code, /^NB1\.10000\.[A-Za-z0-9_-]+$/)
  const back = ids(code)
  assert.deepEqual(back, sorted)
  assert.ok(back.includes(880024) && back.includes(188198))
})

test('unsorted ids with duplicates come back sorted and once each', () => {
  const code = wardrobeCode([30001, 10001, 880024, 30001, 188198, 10001])
  assert.equal(code, wardrobeCode([10001, 30001, 188198, 880024]))
  assert.match(code, /^NB1\.4\./)
  assert.deepEqual(ids(code), [10001, 30001, 188198, 880024])
})

test('ids that cannot be item ids are left out of a code', () => {
  assert.equal(wardrobeCode([0, -5, 1.5, 10001, MAX_ID + 1, Number.NaN]), wardrobeCode([10001]))
  assert.deepEqual(ids(wardrobeCode([1, MAX_ID])), [1, MAX_ID])
})

test('the largest wardrobe a code may hold reads back', () => {
  const list = Array.from({ length: MAX_IDS }, (_, i) => i + 1)
  assert.deepEqual(ids(wardrobeCode(list)), list)
})

test('a code is found inside other text', () => {
  const code = wardrobeCode([10001, 20002, 188198, 880024])
  const want = [10001, 20002, 188198, 880024]
  assert.deepEqual(ids(`My wardrobe: ${code} (from NikkiBase)`), want)
  assert.deepEqual(ids(`Try mine!\n${code}\nTell me what you think.`), want)
  assert.deepEqual(ids(`Here it is: ${code}. Have fun`), want)
  assert.deepEqual(ids(`  ${code}  `), want)
})

test('line breaks and spaces a messenger adds inside a code are ignored', () => {
  const list = realIds(800)
  const code = wardrobeCode(list)
  const sorted = [...list].sort((a, b) => a - b)
  const wrapped = code.replace(/(.{37})/g, '$1\n')
  assert.deepEqual(ids(wrapped), sorted)
  assert.deepEqual(ids(code.replace(/(.{50})/g, '$1\r\n')), sorted)
  assert.deepEqual(ids(code.replace(/(.{23})/g, '$1 ')), sorted)
  assert.deepEqual(ids(code.replace(/(.{61})/g, '$1\u200b')), sorted)
  const [head, body] = [code.slice(0, code.indexOf('.', 4) + 1), code.slice(code.indexOf('.', 4) + 1)]
  assert.deepEqual(ids(`${head}\n${body}`), sorted)
  assert.deepEqual(ids(`Wardrobe:\n${wrapped}\nThanks!`), sorted)
})

test('a whole saved file reads back to its ids', () => {
  const list = realIds(1500)
  const file = wardrobeFile(list, new Date(2026, 8, 30, 12), 'nikkibase.up.railway.app')
  assert.deepEqual(ids(file), [...list].sort((a, b) => a - b))
})

test('a code cut short is incomplete', () => {
  const code = wardrobeCode(realIds(400))
  for (const cut of [1, 2, 3, 4, 10, Math.floor(code.length / 2), code.length - 'NB1.400.'.length]) {
    assert.deepEqual(read(code.slice(0, -cut)), { ok: false, problem: 'incomplete' }, `cut ${cut}`)
  }
  for (const head of ['NB1.', 'NB1.4', 'NB1.400', 'NB1.400.', 'NB1.400 ']) {
    assert.deepEqual(read(head), { ok: false, problem: 'incomplete' }, head)
  }
  assert.deepEqual(read(`${code.slice(0, -12)}… see more`), { ok: false, problem: 'incomplete' })
})

test('a code cut short and followed by other text is incomplete', () => {
  const code = wardrobeCode(realIds(400))
  const tails = ['\nDrop this file on example.test or paste the code there to load it.', ' thanks!', '\n\nSee you', '… more']
  for (const cut of [1, 2, 10, 25, 100, Math.floor(code.length / 2)]) {
    for (const tail of tails) {
      assert.deepEqual(read(`${code.slice(0, -cut)}${tail}`), { ok: false, problem: 'incomplete' }, `cut ${cut} + ${JSON.stringify(tail)}`)
    }
  }
})

test('a code whose count is higher than its ids is incomplete', () => {
  const code = wardrobeCode([10001, 20002, 30003])
  assert.deepEqual(read(code.replace('NB1.3.', 'NB1.4.')), { ok: false, problem: 'incomplete' })
  assert.deepEqual(read(code.replace('NB1.3.', 'NB1.3000.')), { ok: false, problem: 'incomplete' })
})

test('a code whose count is lower than its ids, or with text glued on, is not a code', () => {
  const code = wardrobeCode([10001, 20002, 30003])
  assert.deepEqual(read(code.replace('NB1.3.', 'NB1.2.')), { ok: false, problem: 'invalid' })
  assert.deepEqual(read(`${code}thanks`), { ok: false, problem: 'invalid' })
  assert.deepEqual(read(`${code}-`), { ok: false, problem: 'invalid' })
})

test('garbage that starts like a code is not a code', () => {
  const invalid = { ok: false, problem: 'invalid' }
  assert.deepEqual(read('NB1.x.AAAA'), invalid, 'no count')
  assert.deepEqual(read('NB1..kU4'), invalid, 'empty count')
  assert.deepEqual(read('NB1.1,kU4'), invalid, 'no dot after the count')
  assert.deepEqual(read(`NB1.1.${payload([0])}`), invalid, 'id 0')
  assert.deepEqual(read(`NB1.1.${payload(varints([MAX_ID + 1]))}`), invalid, 'id above the range')
  assert.deepEqual(read(`NB1.2.${payload(varints([MAX_ID, 1]))}`), invalid, 'second id above the range')
  assert.deepEqual(read(`NB1.2.${payload([1, 0])}`), invalid, 'the same id twice')
  assert.deepEqual(read(`NB1.1.${payload([0xff, 0xff, 0xff, 0xff, 0x01])}`), invalid, 'varint too long')
  assert.deepEqual(read(`NB1.1.${payload([0x81, 0x00])}`), invalid, 'varint with a zero last byte')
  assert.deepEqual(read('NB1.1.AR'), invalid, 'bits left over after the last id')
  assert.deepEqual(read('NB1.3.____________'), invalid, 'all ones')
  assert.deepEqual(read(`NB1.${MAX_IDS + 1}.${wardrobeCode([10001]).slice(6)}`), invalid, 'too many ids')
  const tooMany = wardrobeCode(Array.from({ length: MAX_IDS + 1 }, (_, i) => i + 1))
  assert.deepEqual(read(tooMany), invalid, 'a real code with too many ids')
  assert.deepEqual(read('NB1.99999999999999999999.AAAA'), invalid, 'a huge count')
  assert.equal(read(`NB1.1.${payload(varints([MAX_ID]))}`).ok, true)
})

test('text without a code says there is none', () => {
  for (const text of ['', '@SEL10001,20002', 'aGVsbG8gd29ybGQ=', 'NikkiBase wardrobe', 'nb1.1.kU4', 'NB2.1.kU4']) {
    assert.deepEqual(read(text), { ok: false, problem: 'none' }, text)
  }
})

test('the first code in the text is the one read', () => {
  const first = wardrobeCode([10001])
  const second = wardrobeCode([20002, 30003])
  assert.deepEqual(ids(`${first}\n${second}`), [10001])
})

test('a saved file names the wardrobe, holds the code and says how to load it', () => {
  const list = [30001, 10001, 880024, 10001]
  const text = wardrobeFile(list, new Date(2026, 8, 30, 23, 59), 'nikkibase.up.railway.app')
  const lines = text.split('\n')
  assert.deepEqual(lines, [
    'NikkiBase wardrobe · 3 items · saved 2026-09-30',
    wardrobeCode(list),
    'Drop this file on nikkibase.up.railway.app or paste the code there to load it.',
    '',
  ])
  const many = wardrobeFile(realIds(3823), new Date(2026, 0, 5), 'example.test').split('\n')
  assert.equal(many[0], 'NikkiBase wardrobe · 3,823 items · saved 2026-01-05')
  assert.equal(many[2], 'Drop this file on example.test or paste the code there to load it.')
  assert.equal(wardrobeFile([10001], new Date(2026, 8, 30), 'x').split('\n')[0], 'NikkiBase wardrobe · 1 item · saved 2026-09-30')
})

test('the file name carries the local date', () => {
  assert.equal(fileName(new Date(2026, 8, 30, 23, 59)), 'nikkibase-wardrobe-2026-09-30.txt')
  assert.equal(fileName(new Date(2026, 0, 5, 0, 1)), 'nikkibase-wardrobe-2026-01-05.txt')
  assert.equal(fileName(new Date(2031, 11, 31, 12)), 'nikkibase-wardrobe-2031-12-31.txt')
})

test('a code is loaded as a code, even inside other text or when it is broken', () => {
  const code = wardrobeCode([10001, 20002])
  for (const text of [code, `Here is mine: ${code} have fun`, wardrobeFile([10001, 20002], new Date(2026, 8, 30), 'x')]) {
    assert.deepEqual(wardrobeKind(text), { kind: 'code', read: { ok: true, ids: [10001, 20002] } }, text)
  }
  assert.deepEqual(wardrobeKind(code.slice(0, 6)), { kind: 'code', read: { ok: false, problem: 'incomplete' } })
  assert.deepEqual(wardrobeKind('NB1.x'), { kind: 'code', read: { ok: false, problem: 'invalid' } })
})

test('a selections file is loaded as one, and anything else goes to the clothes_date reader', () => {
  assert.deepEqual(wardrobeKind('@SEL10001,20002'), { kind: 'sel' })
  assert.deepEqual(wardrobeKind('\n  @SEL10001'), { kind: 'sel' })
  for (const text of ['aGVsbG8gd29ybGQ=', 'aGVs\nbG8g\nd29y', 'hello there', 'nb1.1.kU4', 'x @SEL10001']) {
    assert.deepEqual(wardrobeKind(text), { kind: 'clothes_date' }, text)
  }
})
