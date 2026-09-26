import { test } from 'node:test'
import assert from 'node:assert/strict'
import type { WorthFilter, WorthRanking, WorthRow, WorthSettings, WorthSuit, WorthVersion } from '../src/engine.ts'
import { crashError } from '../src/recycle.ts'
import { NO_SKILLS, worthSettings, worthSkillsText, type SkillSettings } from '../src/skills.ts'
import { variantStages, worthSkip, worthVersions, type IdealTable, type Stage } from '../src/stages.ts'
import {
  ALL_MODES, NO_SOURCE, OWNED_KEY, PAST_NOTE, RESTARTED, SCORE_F, UNLOCK_NOTE, askRanking, chipText, detailsLabel, hideLabel, ownsAnyPart,
  filterMode, filterSuits, gainText, groupPieces, groupTail, hashIds, hardToGet, howToGet, improvesParts, improvesText, itemMeta,
  neededLine, noSession, nothingText, openRows, openTarget, pctText, pieceList, piecesText, rankedName, rankingNote, rankingStatus,
  recipeText, rowKey, rowName, scoreFNote, stageLabel, suitPieces, unlockRanking, unlockText, worthFilter, worthKey, worthRunner,
  worthSuits, type AcquireTable, type WorthApi,
  MORE_WAIT, askSteps, checkingText, rankSteps, rankingText, waitPercent, type Stepped,
} from '../src/worth.ts'

const W = [1, 2, 3, 4, 5]
const A = [0, 1, 2, 3, 4]

const stages: Stage[] = [
  { name: 'Beach Party', mode: 'Arena', weights: [10, 15, 20, 20, 20], attrs: [1, 3, 5, 6, 9] },
  {
    name: '6-9',
    mode: 'Story',
    weights: [9, 9, 9, 9, 9],
    attrs: [5, 6, 7, 8, 9],
    tags: { 3: 1500 },
    rules: { require: [[40080], [50091, 50092]] },
    variants: { maiden: { weights: W, attrs: A, tags: { 7: 800 } } },
  },
  { name: '1-1', mode: 'Story', weights: W, attrs: A, tags: {} },
  { name: '20-7', mode: 'Commission', weights: W, attrs: [5, 6, 7, 8, 9] },
  { name: 'Missing', mode: 'Commission', weights: W, attrs: A },
]

const auto = (score: number) => ({ score, items: [], charmSmile: 0, smile: 1 })

const table: IdealTable = {
  'Arena/Beach Party': { score: 1000, items: [], auto: auto(1300) },
  'Story/6-9': { score: 2000, items: [], auto: auto(2600), variants: { maiden: { score: 900, items: [], auto: auto(1100) } } },
  'Story/1-1': { score: 500, items: [], auto: auto(640) },
  'Commission/20-7': { score: 700, items: [], auto: auto(910) },
}

const on = (charming: number, smile: number, manual = false): SkillSettings => ({ on: true, manual, levels: { charming, smile } })

test('every stage version gets its own entry, Story first, with the Maiden variant keyed like placements', () => {
  const versions = worthVersions(stages, table, 'none')
  assert.deepEqual(
    versions.map((v) => v.key),
    ['Story/6-9', 'Story/6-9#maiden', 'Story/1-1', 'Commission/20-7', 'Arena/Beach Party'],
  )
  const [princess, maiden, plain] = versions
  assert.deepEqual(princess, {
    key: 'Story/6-9', mode: 'Story', weights: [9, 9, 9, 9, 9], attrs: [5, 6, 7, 8, 9], ideal: 2000,
    tags: { 3: 1500 }, require: [[40080], [50091, 50092]],
  })
  assert.deepEqual(maiden, {
    key: 'Story/6-9#maiden', mode: 'Story', weights: W, attrs: A, ideal: 900,
    tags: { 7: 800 }, require: [[40080], [50091, 50092]],
  })
  assert.equal(plain.tags, undefined, 'an empty tag table is left out')
  assert.equal(plain.require, undefined)
})

test('with skills on the ranking divides by the automatic best possible score', () => {
  const ideals = worthVersions(stages, table, 'max').map((v) => v.ideal)
  assert.deepEqual(ideals, [2600, 1100, 640, 910, 1300])
})

test('a stage the best possible table lacks, or scores at zero, is left out', () => {
  const broken: IdealTable = { ...table, 'Story/1-1': { score: 0, items: [] } }
  const keys = worthVersions(stages, broken, 'none').map((v) => v.key)
  assert.ok(!keys.includes('Story/1-1'))
  assert.ok(!keys.includes('Commission/Missing'))
})

test('Maiden skips the Princess numbers of stages that differ, and Princess skips the Maiden ones', () => {
  assert.deepEqual(worthSkip(stages, 'Maiden'), ['Story/6-9'])
  assert.deepEqual(worthSkip(stages, 'Princess'), ['Story/6-9#maiden'])
  assert.deepEqual([...variantStages(stages)], ['Story/6-9'])
})

test('the filter names a mode, a slot or a place, and the skipped versions only when they matter', () => {
  assert.deepEqual(worthFilter(ALL_MODES, {}, []), {})
  assert.deepEqual(worthFilter(ALL_MODES, {}, ['Story/6-9']), { skip: ['Story/6-9'] })
  assert.deepEqual(worthFilter('Story', { slots: [3] }, ['Story/6-9']), { modes: ['Story'], slots: [3], skip: ['Story/6-9'] })
  assert.deepEqual(worthFilter('Commission', { slots: [0] }, ['Story/6-9']), { modes: ['Commission'], slots: [0] })
  assert.deepEqual(worthFilter('Arena', { places: [14] }, []), { modes: ['Arena'], places: [14] })
  assert.deepEqual(worthFilter(ALL_MODES, {}, ['Story/6-9'], true), { skip: ['Story/6-9'], suits: true })
  assert.deepEqual(worthFilter('Commission', { places: [9] }, [], true), { modes: ['Commission'], places: [9], suits: true })
  assert.deepEqual(worthFilter('Arena', {}, [], false), { modes: ['Arena'] })
})

test('rows are labelled from the filter they were ranked for, not the one being ranked now', () => {
  assert.equal(filterMode(worthFilter(ALL_MODES, {}, ['Story/6-9'])), ALL_MODES)
  assert.equal(filterMode(worthFilter('Story', { slots: [3] }, [])), 'Story')
  assert.equal(filterSuits(worthFilter('Commission', {}, [], true)), true)
  assert.equal(filterSuits(worthFilter('Commission', {}, [])), false)
})

test('the ranking status says what is being ranked and never promises a time', () => {
  assert.equal(rankingStatus(true), 'Ranking suits…')
  assert.equal(rankingStatus(false), 'Ranking items…')
})

test('suits go to the engine by name, with the pieces that have stats, in name order', () => {
  const items = [
    { id: 20854, suit: 'Icewind Warchant', scoreable: true },
    { id: 10001, suit: '', scoreable: true },
    { id: 81770, suit: 'Icewind Warchant', scoreable: true },
    { id: 99001, suit: 'Icewind Warchant', scoreable: false },
    { id: 30423, suit: 'Crimson Feast', scoreable: true },
    { id: 99002, suit: 'Only a Name', scoreable: false },
  ]
  assert.deepEqual(worthSuits(items), [
    { key: 'Crimson Feast', items: [30423] },
    { key: 'Icewind Warchant', items: [20854, 81770] },
  ])
  assert.deepEqual(worthSuits([]), [])
})

test('a suit row is keyed and counted by its suit, and lists each piece by place and rarity', () => {
  const suit: WorthRow = { suit: 'Icewind Warchant', items: [20854, 81770], places: [1, 12], worth: 12, stages: 3, best: { key: 'Story/1-1', points: 5, pct: 1 }, examples: [] }
  assert.equal(rowKey(suit), 'suit:Icewind Warchant')
  assert.equal(rowKey({ items: [40001, 50001] }), '40001+50001')
  assert.equal(piecesText(1), "1 piece you don't own")
  assert.equal(piecesText(1234), "1,234 pieces you don't own")
  const places = Array.from({ length: 13 }, (_, i) => ({ name: i === 1 ? 'Dress' : i === 12 ? 'Earrings' : `Place ${i}` }))
  const items = new Map([
    [20854, { name: 'Battle Song', rarity: 5 }],
    [81770, { name: "Grani's Lightning", rarity: 0 }],
  ])
  assert.deepEqual(suitPieces(suit, items, places), [
    { id: 20854, name: 'Battle Song', place: 'Dress', meta: 'Dress · ★★★★★' },
    { id: 81770, name: "Grani's Lightning", place: 'Earrings', meta: 'Earrings' },
  ])
  assert.deepEqual(suitPieces({ items: [7], places: [99] }, items, places), [{ id: 7, name: '#7', place: '', meta: '' }])
})

test('the notes say what a row is ranked by, and the empty list what a suit row means', () => {
  assert.equal(
    rankingNote(false),
    "Ranked by total gain: the % an item adds on each stage, added up (the bar). A small gain on many stages can rank high. % is of each stage's best possible score. Each row assumes you got the ones above it.",
  )
  assert.equal(
    rankingNote(true),
    "Ranked by total gain if you get every missing piece of a suit (the bar). % is of each stage's best possible score. Each row assumes you got the suits above it. Items not in a suit show with group by suit off.",
  )
  assert.equal(rankingNote(true, true), `${rankingNote(true)} A suit shows if one of its pieces fits the slot. Its gain counts every piece.`)
  assert.equal(rankingNote(false, true), rankingNote(false))
  assert.equal(UNLOCK_NOTE, 'Ranked by stages unlocked per item. Items you can still get come first. Each row assumes you got the ones above it.')
  assert.match(nothingText(true), /^No suit you could complete/)
  assert.match(nothingText(false), /^Nothing you could get/)
})

test('skills go to the engine as automatic placement, whatever the player chose on Best outfit', () => {
  assert.equal(worthSettings(NO_SKILLS), null)
  assert.equal(worthSettings(on(0, 0)), null)
  assert.equal(worthSettings({ ...on(9, 9), on: false }), null)
  assert.deepEqual(worthSettings(on(9, 9)), { auto: true })
  assert.deepEqual(worthSettings(on(9, 9, true)), { auto: true })
  assert.deepEqual(worthSettings(on(4, 6, true)), { auto: true, levels: { charming: 4, smile: 6 } })
})

test('the skills line says what the ranking assumes', () => {
  assert.equal(worthSkillsText(NO_SKILLS), 'Scored without skills. Turn Skills on in Best outfit to include them.')
  assert.equal(worthSkillsText(on(0, 0)), 'Scored without skills: Smile and Charming are both off.')
  assert.equal(worthSkillsText(on(9, 9, true)), 'Scored with Smile 9 and Charming 9.')
  assert.equal(worthSkillsText(on(4, 6)), 'Scored with Smile 6 and Charming 4.')
  assert.equal(worthSkillsText(on(0, 6)), 'Scored with Smile 6.')
  assert.equal(worthSkillsText(on(5, 0)), 'Scored with Charming 5.')
})

test('the cache key changes with the data, the skills and the wardrobe, not with the order of ids', () => {
  assert.equal(hashIds([3, 1, 2]), hashIds([1, 2, 3]))
  assert.notEqual(hashIds([1, 2, 3]), hashIds([1, 2, 4]))
  assert.notEqual(hashIds([1, 2]), hashIds([1, 2, 0]))
  assert.notEqual(hashIds([]), hashIds([0]))
  assert.equal(hashIds([]).split('.')[0], '0')
  const owned = [10001, 20002, 880048]
  const none: WorthSettings = null
  const max: WorthSettings = { auto: true }
  const key = worthKey('2026-09-25', none, owned)
  assert.equal(key, worthKey('2026-09-25', null, [...owned].reverse()))
  assert.notEqual(key, worthKey('2026-09-24', none, owned))
  assert.notEqual(key, worthKey('2026-09-25', max, owned))
  assert.notEqual(worthKey('v', max, owned), worthKey('v', { auto: true, levels: { charming: 4, smile: 6 } }, owned))
  assert.notEqual(key, worthKey('2026-09-25', none, owned.slice(1)))
})

test('stage labels read like the game, naming the difficulty only where the two differ', () => {
  const variants = new Set(['Story/6-9'])
  assert.equal(stageLabel('Commission/20-7', variants), 'Commission 20-7')
  assert.equal(stageLabel('Story/1-1', variants), 'Story 1-1')
  assert.equal(stageLabel('Story/6-9', variants), 'Story 6-9 (Princess)')
  assert.equal(stageLabel('Story/6-9#maiden', variants), 'Story 6-9 (Maiden)')
  assert.equal(stageLabel('Story/II-4-Side 2'), 'Story II-4-Side 2')
  assert.equal(stageLabel('Co-op/Tea Party'), 'Co-op Tea Party')
  assert.equal(stageLabel('Arena/Beach Party'), 'Arena Beach Party')
})

test('Open picks the mode, the stage and, where the two differ, the difficulty', () => {
  const variants = new Set(['Story/6-9'])
  assert.deepEqual(openTarget('Story/6-9#maiden', variants), { mode: 'Story', stage: 'Story/6-9', difficulty: 'Maiden' })
  assert.deepEqual(openTarget('Story/6-9', variants), { mode: 'Story', stage: 'Story/6-9', difficulty: 'Princess' })
  assert.deepEqual(openTarget('Story/1-1', variants), { mode: 'Story', stage: 'Story/1-1', difficulty: null })
  assert.deepEqual(openTarget('Co-op/Tea Party'), { mode: 'Co-op', stage: 'Co-op/Tea Party', difficulty: null })
})

test('percentages keep one decimal, and small ones keep their first digit', () => {
  assert.equal(pctText(1), '1.0')
  assert.equal(pctText(12.345), '12.3')
  assert.equal(pctText(0.4), '0.4')
  assert.equal(pctText(0.1), '0.1')
  assert.equal(pctText(0.096), '0.1')
  assert.equal(pctText(0.04), '0.04')
  assert.equal(pctText(0.00123), '0.001')
  assert.equal(pctText(0), '0')
})

const row = (stages: number, worth: number, key: string, pct: number): WorthRow => ({
  items: [1],
  places: [0],
  worth,
  stages,
  best: { key, points: 412, pct },
  examples: [],
})

test('each row says its gain per stage, on how many stages, and its best stage', () => {
  assert.equal(
    improvesText(row(214, 85.6, 'Commission/20-7', 1), 'Story'),
    '+0.4% on 214 Story stages · best +1.0% on Commission 20-7',
  )
  assert.equal(
    improvesText(row(1214, 121.4, 'Story/6-9#maiden', 2.25), ALL_MODES, new Set(['Story/6-9'])),
    '+0.1% on 1,214 stages · best +2.3% on Story 6-9 (Maiden)',
  )
  assert.equal(improvesText(row(1, 0.8, 'Arena/Beach Party', 0.8), 'Arena'), '+0.8% on Arena Beach Party')
})

test('the stage in a row comes apart from the words, so it can be kept on one line', () => {
  assert.deepEqual(improvesParts(row(1, 0.8, 'Story/1-9', 0.8), 'Story'), ['+0.8% on ', 'Story 1-9', ''])
  const [lead, stage, tail] = improvesParts(row(22, 39.6, 'Story/1-9', 3.4), 'Story')
  assert.equal(lead, '+1.8% on 22 Story stages · best +3.4% on ')
  assert.equal(stage, 'Story 1-9')
  assert.equal(tail, '')
})

test('the needed line counts the stages left out', () => {
  assert.equal(neededLine(1), "1 stage you can't pass yet is left out.")
  assert.equal(neededLine(73), "73 stages you can't pass yet are left out.")
  assert.equal(neededLine(1234), "1,234 stages you can't pass yet are left out.")
})

test('each item is named by its exact place, and a pair says it is worth more together', () => {
  const places = [
    { name: 'Hair', slot: 0 },
    { name: 'Top', slot: 3 },
    { name: 'Bottom', slot: 4 },
    { name: 'Leglets', slot: 5 },
    { name: 'Hosiery', slot: 5 },
    { name: 'Earrings', slot: 8 },
    { name: 'Held, right', slot: 8 },
  ]
  assert.equal(itemMeta({ items: [170004], places: [5] }, places, 5), 'Earrings · ★★★★★')
  assert.equal(itemMeta({ items: [170009], places: [6] }, places, 0), 'Held, right')
  assert.equal(itemMeta({ items: [60001], places: [3] }, places, 3), 'Leglets · ★★★')
  assert.equal(itemMeta({ items: [40001, 50001], places: [1, 2] }, places, 4), 'Top + Bottom · worth more together')
  assert.equal(itemMeta({ items: [1], places: [99] }, places, 0), '')
})

test('how to get: each way, its recipe, which ingredients you own, and whether it may be over', () => {
  const names = new Map([[80951, 'Strawberry Basket - Matcha'], [60285, 'Fluttering - Purple']])
  const lines = howToGet(
    [
      {
        k: 'craft',
        t: 'Craft: 5× Strawberry Basket - Matcha, 5× Fluttering - Purple, 3× Manjusaka',
        from: [[80951, 5], [60285, 5], [80148, 3]],
        recipe: 'Store of Starlight · 204 Starlight Coin',
      },
      { k: 'event', t: 'Circus Night event', past: 1 },
      { k: 'stage', t: 'Story 3-12 (Maiden)', stage: 'Story/3-12', level: 'Maiden' },
      { k: 'stage', t: 'Story 4-1', stage: 'Story/4-1' },
    ],
    new Set([60285]),
    names,
  )
  assert.deepEqual(lines[0], {
    text: 'Craft',
    recipe: 'Store of Starlight · 204 Starlight Coin',
    from: [
      { id: 80951, name: 'Strawberry Basket - Matcha', qty: 5, owned: false },
      { id: 60285, name: 'Fluttering - Purple', qty: 5, owned: true },
      { id: 80148, name: '#80148', qty: 3, owned: false },
    ],
    past: false,
    stage: null,
  })
  assert.deepEqual(lines[1], { text: 'Circus Night event', recipe: null, from: [], past: true, stage: null })
  assert.deepEqual(lines[2].stage, { mode: 'Story', stage: 'Story/3-12', difficulty: 'Maiden' })
  assert.deepEqual(lines[3].stage, { mode: 'Story', stage: 'Story/4-1', difficulty: null })
  assert.deepEqual(howToGet(undefined, new Set(), names), [])
})

test('the owned key line is needed only when a way lists an ingredient you own', () => {
  const names = new Map([[60285, 'Fluttering - Purple']])
  const craft = [{ k: 'craft', t: 'Craft: 5× Fluttering - Purple', from: [[60285, 5]] as [number, number][] }]
  assert.equal(ownsAnyPart(howToGet(craft, new Set([60285]), names)), true)
  assert.equal(ownsAnyPart(howToGet(craft, new Set(), names)), false)
  assert.equal(ownsAnyPart(howToGet([{ k: 'event', t: 'Circus Night event' }], new Set([60285]), names)), false)
})

test('an ingredient a recipe lists twice is shown once, with both amounts added', () => {
  const names = new Map([[80410, 'Festival Atmosphere'], [80323, 'Studded Bracelet']])
  const [line] = howToGet(
    [{ k: 'craft', t: 'Craft: 2× Festival Atmosphere, 2× Studded Bracelet, 3× Studded Bracelet', from: [[80410, 2], [80323, 2], [80323, 3]] }],
    new Set([80323]),
    names,
  )
  assert.deepEqual(line.from, [
    { id: 80410, name: 'Festival Atmosphere', qty: 2, owned: false },
    { id: 80323, name: 'Studded Bracelet', qty: 5, owned: true },
  ])
})

const flush = () => new Promise((resolve) => setTimeout(resolve, 0))

const rankedRow = (i: number): WorthRow => ({
  items: [i],
  places: [0],
  worth: 1000 - i,
  stages: 1,
  best: { key: 'Story/1-1', points: 1000 - i, pct: 1 },
  examples: [],
})

function fakeEngine(total: number, ranked?: { rows: number; held?: boolean }) {
  let session = 100
  let live: number | null = null
  let done = 0
  let broken: Error | null = null
  let crash: Error | null = null
  const calls: string[] = []
  const held: (() => void)[] = []
  const ranking: WorthRanking = { rows: [], needed: [] }
  const later = <T,>(work: () => T) =>
    new Promise<T>((resolve, reject) =>
      held.push(() => {
        try {
          resolve(work())
        } catch (e) {
          reject(e)
        }
      }),
    )
  const api: WorthApi = {
    start: (list: WorthVersion[], settings: WorthSettings, suits: WorthSuit[]) => {
      calls.push(`start ${list.length} ${JSON.stringify(settings)}${suits.length ? ` ${suits.map((s) => s.key).join(',')}` : ''}`)
      return later(() => {
        live = ++session
        done = 0
        return { session: live, total }
      })
    },
    run: (id: number, count: number) => {
      calls.push(`run ${id} ${count}`)
      return later(() => {
        if (broken) throw broken
        if (id !== live) throw new Error('no session')
        done = Math.min(total, done + count)
        return { done, total }
      })
    },
    rank: async (id: number, filter: WorthFilter, limit: number) => {
      calls.push(`rank ${id} ${JSON.stringify(filter)} ${limit}`)
      if (crash) {
        const e = crash
        crash = null
        live = null
        throw e
      }
      if (id !== live) throw new Error('no session')
      if (!ranked) return ranking
      const rows = () => ({ rows: Array.from({ length: Math.min(limit, ranked.rows) }, (_, i) => rankedRow(i)), needed: [] })
      return ranked.held ? later(rows) : rows()
    },
  }
  return {
    api,
    calls,
    count: (kind: string) => calls.filter((c) => c.startsWith(kind)).length,
    end: () => (live = null),
    fail: (e: Error) => (broken = e),
    crash: (e: Error) => (crash = e),
    release: async () => {
      held.shift()?.()
      await flush()
    },
    drive: async (runner: ReturnType<typeof worthRunner>, phase: string) => {
      for (let i = 0; i < 200 && runner.get().phase !== phase; i++) {
        held.shift()?.()
        await flush()
      }
      assert.equal(runner.get().phase, phase)
    },
  }
}

const versions = (n: number) => () =>
  Array.from({ length: n }, (_, i) => ({ key: `Story/${i}`, mode: 'Story', weights: W, attrs: A, ideal: 1 }))
const request = (key: string, n = 5) => ({ key, settings: null, versions: versions(n) })

test('the runner checks stages in chunks and reports progress in order', async () => {
  const fake = fakeEngine(70)
  const runner = worthRunner(fake.api, 32)
  const seen: number[] = []
  runner.subscribe(() => seen.push(runner.get().done))
  runner.ensure(request('k1', 70))
  assert.equal(runner.get().phase, 'running')
  assert.equal(runner.get().total, 70)
  await fake.drive(runner, 'done')
  assert.deepEqual(fake.calls, ['start 70 null', 'run 101 32', 'run 101 32', 'run 101 32'])
  assert.deepEqual(seen.filter((d, i) => d !== seen[i - 1]), [0, 32, 64, 70])
  runner.ensure(request('k1', 70))
  assert.equal(fake.calls.length, 4, 'the same wardrobe and settings reuse the session')
})

test('the suits of a request go to the engine when the session starts, and again after a restart', async () => {
  const fake = fakeEngine(10)
  const runner = worthRunner(fake.api, 32)
  let built = 0
  const suits = () => {
    built++
    return [{ key: 'A', items: [1, 2] }, { key: 'B', items: [3] }]
  }
  runner.ensure({ ...request('k1', 10), suits })
  await fake.drive(runner, 'done')
  assert.deepEqual(fake.calls.filter((c) => c.startsWith('start')), ['start 10 null A,B'])
  fake.end()
  await runner.rank({ suits: true }, 50).catch(() => undefined)
  await fake.drive(runner, 'done')
  assert.deepEqual(fake.calls.filter((c) => c.startsWith('start')), ['start 10 null A,B', 'start 10 null A,B'])
  assert.equal(built, 2)
  runner.ensure(request('k2', 10))
  await fake.drive(runner, 'done')
  assert.equal(fake.calls.at(-2), 'start 10 null', 'a request without suits sends none')
})

test('rankings are asked once per filter and limit, and a new key starts over', async () => {
  const fake = fakeEngine(10)
  const runner = worthRunner(fake.api, 32)
  runner.ensure(request('k1', 10))
  await fake.drive(runner, 'done')
  const a = runner.rank({ modes: ['Story'] }, 50)
  assert.equal(runner.rank({ modes: ['Story'] }, 50), a)
  await a
  await runner.rank({ modes: ['Story'] }, 200)
  await runner.rank({}, 50)
  assert.equal(fake.count('rank'), 3)
  runner.ensure(request('k2', 10))
  assert.equal(runner.get().key, 'k2')
  await fake.drive(runner, 'done')
  assert.equal(fake.count('start'), 2)
  await runner.rank({ modes: ['Story'] }, 50)
  assert.equal(fake.count('rank'), 4)
})

test('Stop waits for the chunk in flight, keeps what it checked, and Continue carries on', async () => {
  const fake = fakeEngine(100)
  const runner = worthRunner(fake.api, 32)
  runner.ensure(request('k1', 100))
  await fake.release()
  assert.equal(fake.count('run'), 1)
  runner.stop()
  assert.equal(runner.get().phase, 'stopping', 'the chunk in flight still counts')
  await fake.release()
  assert.equal(runner.get().phase, 'stopped')
  await fake.release()
  assert.equal(runner.get().done, 32)
  assert.equal(fake.count('run'), 1, 'nothing runs while stopped')
  await runner.rank({}, 50)
  runner.resume()
  runner.resume()
  await fake.drive(runner, 'done')
  assert.equal(runner.get().done, 100)
  assert.equal(fake.count('start'), 1)
  assert.equal(fake.count('run'), 4, 'one loop, no chunk twice')
})

test('Stop while the session starts leaves nothing checked, and Continue starts checking', async () => {
  const fake = fakeEngine(40)
  const runner = worthRunner(fake.api, 32)
  runner.ensure(request('k1', 40))
  runner.stop()
  assert.equal(runner.get().phase, 'stopping')
  await fake.release()
  assert.equal(runner.get().phase, 'stopped')
  assert.equal(runner.get().done, 0)
  assert.equal(fake.count('run'), 0)
  runner.resume()
  await fake.drive(runner, 'done')
  assert.equal(fake.count('start'), 1)
  assert.equal(fake.count('run'), 2)
})

test('Stop on the last chunk still ends done', async () => {
  const fake = fakeEngine(20)
  const runner = worthRunner(fake.api, 32)
  runner.ensure(request('k1', 20))
  await fake.release()
  runner.stop()
  await fake.release()
  assert.equal(runner.get().phase, 'done')
  assert.equal(runner.get().done, 20)
})

test('Continue right after Stop does not start a second loop', async () => {
  const fake = fakeEngine(100)
  const runner = worthRunner(fake.api, 32)
  runner.ensure(request('k1', 100))
  await fake.release()
  runner.stop()
  runner.resume()
  await fake.drive(runner, 'done')
  assert.equal(fake.count('run'), 4)
})

test('a replaced engine answers "no session", and the runner starts again by itself', async () => {
  const fake = fakeEngine(64)
  const runner = worthRunner(fake.api, 32)
  runner.ensure(request('k1', 64))
  await fake.release()
  fake.end()
  await fake.release()
  assert.equal(fake.count('start'), 2)
  assert.equal(runner.get().phase, 'running')
  assert.equal(runner.get().done, 0)
  await fake.drive(runner, 'done')
  assert.equal(runner.get().key, 'k1')

  fake.end()
  await assert.rejects(runner.rank({}, 50), /no session/)
  await flush()
  assert.equal(runner.get().phase, 'running')
  await fake.drive(runner, 'done')
  assert.equal(fake.count('start'), 3)
  assert.deepEqual(await runner.rank({}, 50), { rows: [], needed: [] })
})

test('the runner gives up after a few restarts in a row, and Try again starts over', async () => {
  const fake = fakeEngine(64)
  const runner = worthRunner(fake.api, 32, 2)
  fake.fail(new Error('no session'))
  runner.ensure(request('k1', 64))
  await fake.drive(runner, 'failed')
  assert.equal(runner.get().error, RESTARTED)
  assert.equal(RESTARTED, 'Ranking stopped unexpectedly. Try again.')
  assert.equal(fake.count('start'), 3)
  runner.retry()
  assert.equal(runner.get().phase, 'running')
  assert.ok(noSession(new Error('no session')))
  assert.ok(!noSession(new Error('The engine crashed and was restarted — try again.')))
})

test('any other engine error stops the run and is shown as it is', async () => {
  const fake = fakeEngine(64)
  const runner = worthRunner(fake.api, 32)
  fake.fail(new Error('The engine crashed and was restarted — try again.'))
  runner.ensure(request('k1', 64))
  await fake.drive(runner, 'failed')
  assert.equal(runner.get().error, 'The engine crashed and was restarted — try again.')
  assert.equal(fake.count('start'), 1)
  runner.ensure(request('k1', 64))
  assert.equal(runner.get().phase, 'running', 'opening the tab again retries')
})

test('an engine crash while ranking fails the run with a Try again, which checks the stages and ranks again', async () => {
  const fake = fakeEngine(64)
  const runner = worthRunner(fake.api, 32)
  runner.ensure(request('k1', 64))
  await fake.drive(runner, 'done')
  const crashed = crashError('worthRank', 'RuntimeError: unreachable')
  fake.crash(crashed)
  await assert.rejects(runner.rank({ modes: ['Story'] }, 50), crashed)
  assert.equal(runner.get().phase, 'failed')
  assert.equal(runner.get().error, 'The engine crashed and was restarted — try again.')
  runner.retry()
  assert.equal(runner.get().phase, 'running')
  await fake.drive(runner, 'done')
  assert.deepEqual(await runner.rank({ modes: ['Story'] }, 50), { rows: [], needed: [] })
  assert.equal(fake.count('start'), 2)
})

test('a hidden tab asks for no ranking, even when its filter changes, and asks once it is shown', async () => {
  const fake = fakeEngine(10)
  const runner = worthRunner(fake.api, 32)
  runner.ensure(request('k1', 10))
  await fake.drive(runner, 'done')
  const settled: string[] = []
  const settle = (ranking: WorthRanking | null, error: string | null) => settled.push(error ?? `${ranking?.rows.length} rows`)
  const maiden = worthFilter(ALL_MODES, {}, worthSkip(stages, 'Maiden'))
  const princess = worthFilter(ALL_MODES, {}, worthSkip(stages, 'Princess'))

  const stop = askRanking(runner, runner.get(), true, maiden, 50, settle)
  await flush()
  assert.equal(fake.count('rank'), 1)
  assert.deepEqual(settled, ['0 rows'])
  stop?.()

  assert.equal(askRanking(runner, runner.get(), false, princess, 50, settle), undefined)
  await flush()
  assert.equal(fake.count('rank'), 1, 'nothing is ranked while the tab is hidden')

  askRanking(runner, runner.get(), true, princess, 50, settle)
  await flush()
  assert.equal(fake.count('rank'), 2)
  assert.deepEqual(settled, ['0 rows', '0 rows'])
})

test('a ranking that lands after the tab was left is dropped, and one that is not ready asks nothing', async () => {
  const fake = fakeEngine(40)
  const runner = worthRunner(fake.api, 32)
  const settled: unknown[] = []
  const settle = (ranking: WorthRanking | null) => settled.push(ranking)
  runner.ensure(request('k1', 40))
  await fake.release()
  await fake.release()
  assert.equal(runner.get().phase, 'running')
  assert.equal(askRanking(runner, runner.get(), true, {}, 50, settle), undefined)
  await fake.drive(runner, 'done')
  const stop = askRanking(runner, runner.get(), true, {}, 50, settle)
  stop?.()
  await flush()
  assert.equal(fake.count('rank'), 1)
  assert.deepEqual(settled, [])
})

test('a reset drops the session, and results of the old run are ignored', async () => {
  const fake = fakeEngine(64)
  const runner = worthRunner(fake.api, 32)
  runner.ensure(request('k1', 64))
  runner.reset()
  assert.deepEqual(runner.get(), { key: null, run: 2, phase: 'idle', done: 0, total: 0, error: null })
  await fake.release()
  await fake.release()
  assert.equal(runner.get().phase, 'idle')
  assert.equal(fake.count('run'), 0)
  await assert.rejects(runner.rank({}, 50), /not ready/)
})

test('a new key while a chunk is in flight leaves the old answer unused', async () => {
  const fake = fakeEngine(64)
  const runner = worthRunner(fake.api, 32)
  runner.ensure(request('k1', 64))
  await fake.release()
  runner.ensure(request('k2', 64))
  await fake.release()
  assert.equal(runner.get().key, 'k2')
  assert.equal(runner.get().done, 0)
  await fake.drive(runner, 'done')
  assert.equal(runner.get().done, 64)
})

test('what unlocks the most stages for each item comes first, and each row assumes the ones above it', () => {
  const rows = unlockRanking([
    { key: 'Story/4-12', missing: [[10], [20]] },
    { key: 'Story/4-12#maiden', missing: [[10], [20]] },
    { key: 'Story/5-1', missing: [[30]] },
    { key: 'Story/6-2', missing: [[10], [20], [40]] },
    { key: 'Story/7-3', missing: [[50], [60]] },
    { key: 'Story/8-1', missing: [[70], [80]] },
    { key: 'Story/8-2', missing: [[70], [80]] },
    { key: 'Story/8-3', missing: [[70], [80]] },
  ])
  assert.deepEqual(rows, [
    { items: [70, 80], stages: ['Story/8-1', 'Story/8-2', 'Story/8-3'] },
    { items: [10, 20], stages: ['Story/4-12', 'Story/4-12#maiden'] },
    { items: [30], stages: ['Story/5-1'] },
    { items: [40], stages: ['Story/6-2'] },
    { items: [50, 60], stages: ['Story/7-3'] },
  ])
})

test('a set with a choice takes the item that other stages need too, and a smaller step comes before a bigger one that adds nothing', () => {
  const rows = unlockRanking([
    { key: 'Story/1-1', missing: [[7, 8]] },
    { key: 'Story/1-2', missing: [[8], [9]] },
  ])
  assert.deepEqual(rows, [
    { items: [8], stages: ['Story/1-1'] },
    { items: [9], stages: ['Story/1-2'] },
  ])
  assert.deepEqual(unlockRanking([{ key: 'Story/1-1', missing: [[7, 8]] }]), [{ items: [7], stages: ['Story/1-1'] }])
  assert.deepEqual(unlockRanking([]), [])
})

test('an unlock row names its stage, or counts them', () => {
  assert.equal(unlockText({ items: [1], stages: ['Story/4-12#maiden'] }, new Set(['Story/4-12'])), 'Unlocks Story 4-12 (Maiden)')
  assert.equal(unlockText({ items: [1, 2], stages: ['Story/4-12', 'Story/4-12#maiden'] }), 'Unlocks 2 stages')
})

test('stages you can unlock with items on offer now come before those needing an item that may be gone', () => {
  const needed = [
    { key: 'Story/1-1', missing: [[1]] },
    { key: 'Story/1-2', missing: [[2], [3]] },
  ]
  assert.deepEqual(unlockRanking(needed).map((r) => r.items), [[1], [2, 3]])
  assert.deepEqual(unlockRanking(needed, (id) => id === 1).map((r) => r.items), [[2, 3], [1]])
  assert.ok(hardToGet(undefined))
  assert.ok(hardToGet([]))
  assert.ok(hardToGet([{ k: 'event', t: 'Circus Night event', past: 1 }]))
  assert.ok(!hardToGet([{ k: 'event', t: 'Circus Night event', past: 1 }, { k: 'shop', t: 'Clothing Store · 3,000 Gold' }]))
})

test('a row is named by its suit or by its items, and on phones carries its rank', () => {
  const names = new Map([[40001, 'Cloud Blouse'], [50001, 'Cloud Skirt']])
  assert.equal(rowName({ suit: 'Icewind Warchant', items: [20854] }, names), 'Icewind Warchant')
  assert.equal(rowName({ items: [40001, 50001] }, names), 'Cloud Blouse + Cloud Skirt')
  assert.equal(rowName({ items: [7] }, names), '#7')
  assert.equal(rankedName(1, 'Cloud Blouse'), '1. Cloud Blouse')
  assert.equal(detailsLabel('Cloud Blouse'), 'Show details for Cloud Blouse')
  assert.equal(hideLabel('Cloud Blouse'), 'Hide details for Cloud Blouse')
})

test('on phones one row is open at a time, and on bigger screens rows open side by side', () => {
  assert.deepEqual(openRows([], 'a', true, true), ['a'])
  assert.deepEqual(openRows(['a'], 'b', true, true), ['b'])
  assert.deepEqual(openRows(['a'], 'b', true, false), ['a', 'b'])
  assert.deepEqual(openRows(['a', 'b'], 'b', true, false), ['a', 'b'])
  assert.deepEqual(openRows(['a', 'b'], 'a', false, false), ['b'])
  assert.deepEqual(openRows(['b'], 'b', false, true), [])
})

test('a stage line gives the gain in points and percent, and says when the stage may score F', () => {
  assert.equal(gainText({ points: 7061, pct: 4.2 }), '+7,061 (+4.2%)')
  assert.equal(gainText({ points: 1, pct: 0.004 }), '+1 (+0.004%)')
  assert.equal(gainText({ points: 412, pct: 1.02 }, true), '+412 (+1.0%) · may score F')
  assert.equal(SCORE_F, 'may score F')
  assert.equal(
    scoreFNote(true),
    "may score F: some items score F on this stage and NikkiBase doesn't check it, so the real gain may be smaller.",
  )
  assert.equal(scoreFNote(false), "may score F: some items score F on this stage and NikkiBase doesn't check it.")
})

test('a way with ingredients starts with its verb and any cost, and each ingredient reads as a quantity chip', () => {
  const names = new Map([[10004, 'Sporty Teenager'], [10064, 'Gourds'], [10065, 'Gourd Bell']])
  const [customize, evolve, craft] = howToGet(
    [
      {
        k: 'customize',
        t: 'Customize: Sporty Teenager + 1 Sunny Orange + 3 Material',
        from: [[10004, 1]],
        cost: [[1, 'Sunny Orange'], [3, 'Material']],
      },
      { k: 'evolve', t: 'Evolve: 4× Gourd Bell + 2,400 Gold', from: [[10065, 4]], cost: [[2400, 'Gold']] },
      { k: 'craft', t: 'Craft: 10× Gourds', from: [[10064, 10]], recipe: 'Available by default' },
    ],
    new Set([10004, 10064]),
    names,
  )
  assert.equal(customize.text, 'Customize · 1 Sunny Orange + 3 Material')
  assert.equal(evolve.text, 'Evolve · 2,400 Gold')
  assert.equal(craft.text, 'Craft')
  assert.equal(chipText(customize.from[0]), '1× Sporty Teenager ✓')
  assert.equal(chipText(evolve.from[0]), '4× Gourd Bell')
  assert.equal(chipText(craft.from[0]), '10× Gourds ✓')
  assert.equal(howToGet([{ k: 'craft', t: 'Crafting' }], new Set(), names)[0].text, 'Crafting')
  assert.equal(OWNED_KEY, '✓ = in your wardrobe (you may need more copies)')
})

test('a recipe says where it comes from, and a way that may be over says it may have ended', () => {
  assert.equal(recipeText('Time Diary'), 'Recipe from Time Diary')
  assert.equal(recipeText('Store of Starlight · 204 Starlight Coin'), 'Recipe from Store of Starlight · 204 Starlight Coin')
  assert.equal(recipeText('Available by default'), 'Recipe: unlocked from the start')
  assert.equal(PAST_NOTE, 'may have ended')
  assert.equal(NO_SOURCE, 'Source unknown')
})

test('suit pieces that share one plain way are listed together under it, and pieces with a way of their own keep their block', () => {
  const piece = (id: number, name: string, place: string) => ({ id, name, place, meta: place })
  const locks = piece(1, 'Neon Locks', 'Hair')
  const aspiration = piece(2, 'Bright Aspiration', 'Makeup')
  const gown = piece(3, 'Star Gown', 'Dress')
  const shoes = piece(4, 'Moon Shoes', 'Shoes')
  const ring = piece(5, 'Glow Ring', '')
  const veil = piece(6, 'Pale Veil', 'Veil')
  const hat = piece(7, 'Old Hat', 'Hat')
  const bow = piece(8, 'Lost Bow', 'Hair Ornament')
  const gloves = piece(9, 'Maiden Gloves', 'Gloves')
  const table: AcquireTable = {
    '1': [{ k: 'event', t: 'Limited event', past: 1 }],
    '2': [{ k: 'event', t: 'Limited event', past: 1 }],
    '3': [{ k: 'craft', t: 'Craft: 2× Star Dress', from: [[30, 2]] }],
    '4': [{ k: 'suit', t: "Styling Gift Box for completing Momo's Star Power" }],
    '5': [{ k: 'event', t: 'Limited event', past: 1 }],
    '6': [{ k: 'event', t: 'Limited event' }],
    '7': [{ k: 'store', t: 'Clothes Store · 1,250 Gold' }, { k: 'pavilion', t: 'Pavilion of Mystery' }],
    '9': [{ k: 'stage', t: 'Story 3-12 (Maiden)', stage: 'Story/3-12', level: 'Maiden' }],
  }
  assert.deepEqual(groupPieces([locks, aspiration, gown, shoes, ring, veil, hat, bow, gloves], table), [
    { key: '1|Limited event', text: 'Limited event', past: true, pieces: [locks, aspiration, ring] },
    gown,
    {
      key: "0|Styling Gift Box for completing Momo's Star Power",
      text: "Styling Gift Box for completing Momo's Star Power",
      past: false,
      pieces: [shoes],
    },
    { key: '0|Limited event', text: 'Limited event', past: false, pieces: [veil] },
    hat,
    { key: '0|Source unknown', text: 'Source unknown', past: false, pieces: [bow] },
    gloves,
  ])
  assert.equal(pieceList([locks, aspiration, ring]), 'Neon Locks (Hair), Bright Aspiration (Makeup), Glow Ring')
  assert.equal(pieceList([{ name: 'Shadow of Nightmare (Dress)', place: 'Dress' }]), 'Shadow of Nightmare (Dress)')
  assert.equal(pieceList([{ name: 'Moon Veil (Hair Ornament)', place: 'Hair ornament' }]), 'Moon Veil (Hair Ornament)')
  assert.equal(pieceList([{ name: 'Dress Up (Hair)', place: 'Dress' }]), 'Dress Up (Hair) (Dress)')
  assert.equal(pieceList([{ name: 'Fragrance in Snow (Handheld)', place: 'Held, right' }]), 'Fragrance in Snow (Handheld)')
  assert.equal(pieceList([{ name: 'Rose Dream (Leglet)', place: 'Leglets' }]), 'Rose Dream (Leglet)')
  assert.equal(pieceList([{ name: 'Fleeting Time (Head Ornament)', place: 'Hair ornament' }]), 'Fleeting Time (Head Ornament)')
  assert.equal(pieceList([{ name: 'Ice Cuff (Bracelet)', place: 'Left hand' }]), 'Ice Cuff (Bracelet)')
  assert.equal(pieceList([{ name: 'Classic (Handheld)', place: 'Hair' }]), 'Classic (Handheld) (Hair)')
  assert.equal(pieceList([{ name: 'Odd (constructor)', place: 'Hair' }]), 'Odd (constructor) (Hair)')
  assert.equal(groupTail({ past: true, pieces: [locks, aspiration, ring] }), '· may have ended (3)')
  assert.equal(groupTail({ past: false, pieces: [shoes] }), '(1)')
})

test('the stage check counts stages as step 1 of 2, with a bar that never rounds up', () => {
  assert.equal(checkingText(544, 605), 'Step 1 of 2 · Checking stages 544 of 605')
  assert.equal(checkingText(1234, 2181), 'Step 1 of 2 · Checking stages 1,234 of 2,181')
  assert.equal(waitPercent(544, 605), 89)
  assert.equal(waitPercent(604, 605), 99)
  assert.equal(waitPercent(605, 605), 100)
  assert.equal(waitPercent(0, 0), 0)
})

test('the first ranking of a run is step 2 of 2, later ones name what is being ranked', () => {
  assert.equal(rankingText({ suits: false, mode: 'Story', first: true }), 'Step 2 of 2 · Ranking items')
  assert.equal(rankingText({ suits: true, mode: ALL_MODES, got: 20, of: 50, first: true }), 'Step 2 of 2 · Ranking suits · 20 of 50')
  assert.equal(rankingText({ suits: true, mode: ALL_MODES, got: 0, of: 50, first: true }), 'Step 2 of 2 · Ranking suits')
  assert.equal(rankingText({ suits: false, mode: 'Story', first: false }), 'Ranking Story items')
  assert.equal(rankingText({ suits: true, mode: ALL_MODES, got: 20, of: 50, first: false }), 'Ranking suits · 20 of 50')
  assert.equal(
    rankingText({ suits: true, mode: 'Commission', where: 'Hair', got: 30, of: 50, first: false }),
    'Ranking Commission suits for Hair · 30 of 50',
  )
  assert.equal(rankingText({ suits: false, mode: ALL_MODES, where: 'Hair', first: false }), 'Ranking items for Hair')
  assert.equal(MORE_WAIT, 'Ranking 50 more…')
})

test('suits are ranked ten at a time, items first ten and then the rest', () => {
  assert.deepEqual(rankSteps(true, 0, 50), [10, 20, 30, 40, 50])
  assert.deepEqual(rankSteps(true, 50, 100), [60, 70, 80, 90, 100])
  assert.deepEqual(rankSteps(false, 0, 50), [10, 50])
  assert.deepEqual(rankSteps(false, 50, 200), [200])
})

const stepLog = (settled: string[]) => (step: Stepped) =>
  settled.push(step.error ?? `${step.got} of ${step.of}${step.streaming ? ' +' : ''}`)

test('the steps are asked in order, and each one shows its rows at once', async () => {
  const fake = fakeEngine(10, { rows: 500 })
  const runner = worthRunner(fake.api, 32)
  runner.ensure(request('k1', 10))
  await fake.drive(runner, 'done')
  const settled: string[] = []
  askSteps(runner, runner.get(), true, { suits: true }, rankSteps(true, 0, 50), stepLog(settled))
  for (let i = 0; i < 10; i++) await flush()
  assert.deepEqual(
    fake.calls.filter((c) => c.startsWith('rank')).map((c) => c.split(' ').at(-1)),
    ['10', '20', '30', '40', '50'],
  )
  assert.deepEqual(settled, ['10 of 50 +', '20 of 50 +', '30 of 50 +', '40 of 50 +', '50 of 50'])
})

test('items ask for ten rows and then fifty, and Show more asks for the rest in one go', async () => {
  const fake = fakeEngine(10, { rows: 500 })
  const runner = worthRunner(fake.api, 32)
  runner.ensure(request('k1', 10))
  await fake.drive(runner, 'done')
  const settled: string[] = []
  askSteps(runner, runner.get(), true, {}, rankSteps(false, 0, 50), stepLog(settled))
  for (let i = 0; i < 5; i++) await flush()
  askSteps(runner, runner.get(), true, {}, rankSteps(false, 50, 200), stepLog(settled))
  for (let i = 0; i < 5; i++) await flush()
  assert.deepEqual(fake.calls.filter((c) => c.startsWith('rank')), ['rank 101 {} 10', 'rank 101 {} 50', 'rank 101 {} 200'])
  assert.deepEqual(settled, ['10 of 50 +', '50 of 50', '200 of 200'])
})

test('a short step ends the stream, since there is nothing more to rank', async () => {
  const fake = fakeEngine(10, { rows: 25 })
  const runner = worthRunner(fake.api, 32)
  runner.ensure(request('k1', 10))
  await fake.drive(runner, 'done')
  const settled: string[] = []
  askSteps(runner, runner.get(), true, { suits: true }, rankSteps(true, 0, 50), stepLog(settled))
  for (let i = 0; i < 10; i++) await flush()
  assert.equal(fake.count('rank'), 3)
  assert.deepEqual(settled, ['10 of 50 +', '20 of 50 +', '25 of 50'])
})

test('a new filter or a hidden tab stops after the step in flight, and coming back replays what was ranked', async () => {
  const fake = fakeEngine(10, { rows: 500, held: true })
  const runner = worthRunner(fake.api, 32)
  runner.ensure(request('k1', 10))
  await fake.drive(runner, 'done')
  const settled: string[] = []
  assert.equal(askSteps(runner, runner.get(), false, { suits: true }, rankSteps(true, 0, 50), stepLog(settled)), undefined)
  assert.equal(fake.count('rank'), 0, 'a hidden tab asks nothing')
  const stop = askSteps(runner, runner.get(), true, { suits: true }, rankSteps(true, 0, 50), stepLog(settled))
  await fake.release()
  assert.equal(fake.count('rank'), 2)
  stop?.()
  await fake.release()
  await flush()
  assert.equal(fake.count('rank'), 2, 'no step is sent once the view has changed')
  assert.deepEqual(settled, ['10 of 50 +'])
  askSteps(runner, runner.get(), true, { suits: true }, rankSteps(true, 0, 50), stepLog(settled))
  for (let i = 0; i < 5; i++) await flush()
  assert.equal(fake.count('rank'), 3, 'the first two steps come from the memo')
  await fake.release()
  assert.deepEqual(settled, ['10 of 50 +', '10 of 50 +', '20 of 50 +', '30 of 50 +'])
})

test('a step that fails shows its error and ends the stream', async () => {
  const fake = fakeEngine(10, { rows: 500 })
  const runner = worthRunner(fake.api, 32)
  runner.ensure(request('k1', 10))
  await fake.drive(runner, 'done')
  const settled: string[] = []
  fake.crash(new Error('boom'))
  askSteps(runner, runner.get(), true, {}, rankSteps(false, 0, 50), stepLog(settled))
  for (let i = 0; i < 5; i++) await flush()
  assert.deepEqual(settled, ['boom'])
  assert.equal(fake.count('rank'), 1)
})

test('steps are not asked before the ranking is ready, and a step that lands after the view was left is dropped', async () => {
  const fake = fakeEngine(40, { rows: 500, held: true })
  const runner = worthRunner(fake.api, 32)
  const settled: string[] = []
  runner.ensure(request('k1', 40))
  await fake.release()
  await fake.release()
  assert.equal(runner.get().phase, 'running')
  assert.equal(askSteps(runner, runner.get(), true, {}, [50], stepLog(settled)), undefined)
  await flush()
  assert.equal(fake.count('rank'), 0)
  await fake.drive(runner, 'done')
  const stop = askSteps(runner, runner.get(), true, {}, [50], stepLog(settled))
  stop?.()
  await fake.release()
  await flush()
  assert.equal(fake.count('rank'), 1)
  assert.deepEqual(settled, [])
})

test('a step the engine answers with "no session" settles nothing, and neither does an error after the view was left', async () => {
  const fake = fakeEngine(10, { rows: 500 })
  const runner = worthRunner(fake.api, 32)
  runner.ensure(request('k1', 10))
  await fake.drive(runner, 'done')
  const settled: string[] = []
  fake.crash(new Error('boom'))
  const stop = askSteps(runner, runner.get(), true, {}, [50], stepLog(settled))
  stop?.()
  for (let i = 0; i < 5; i++) await flush()
  assert.deepEqual(settled, [])
  const gone = fakeEngine(10, { rows: 500 })
  const restarted = worthRunner(gone.api, 32)
  restarted.ensure(request('k1', 10))
  await gone.drive(restarted, 'done')
  gone.end()
  askSteps(restarted, restarted.get(), true, {}, [50], stepLog(settled))
  for (let i = 0; i < 5; i++) await flush()
  assert.equal(gone.count('rank'), 1)
  assert.deepEqual(settled, [])
})
