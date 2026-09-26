import { test } from 'node:test'
import assert from 'node:assert/strict'
import { tick, tickState, toggled, type Stats, type TickDeps } from '../src/ticks.ts'

type Reply = { resolve: (stats: { items: number; known: number }) => void; reject: (e: Error) => void }

function fake(start: number[]) {
  let owned = start
  const sent: number[][] = []
  const saved: number[][] = []
  const settled: Stats[] = []
  const failed: { previous: number[]; error: string; ticked: boolean }[] = []
  const replies: Reply[] = []
  const deps: TickDeps = {
    owned: () => owned,
    show: (next) => {
      owned = next
    },
    send: (ids) => {
      sent.push(ids)
      return new Promise((resolve, reject) => replies.push({ resolve, reject }))
    },
    settle: (stats) => {
      settled.push(stats)
    },
    save: async (ids) => {
      saved.push(ids)
    },
    fail: (previous, e, ticked) => {
      owned = previous
      failed.push({ previous, error: String(e), ticked })
    },
  }
  const state = tickState()
  return { deps, state, sent, saved, settled, failed, replies, owned: () => owned }
}

test('a tick adds the id in order, and a second tick on it takes it off', () => {
  assert.deepEqual(toggled([10, 30], 20), [10, 20, 30])
  assert.deepEqual(toggled([10, 20, 30], 20), [10, 30])
  assert.deepEqual(toggled([], 5), [5])
})

test('a second tick while the engine still has the first keeps both', async () => {
  const f = fake([10])
  const first = tick(30, f.deps, f.state)
  const second = tick(20, f.deps, f.state)
  assert.deepEqual(f.owned(), [10, 20, 30], 'both boxes are ticked at once')
  assert.deepEqual(f.sent, [[10, 30], [10, 20, 30]])
  f.replies[0].resolve({ items: 2, known: 2 })
  f.replies[1].resolve({ items: 3, known: 3 })
  await Promise.all([first, second])
  assert.deepEqual(f.owned(), [10, 20, 30])
  assert.deepEqual(f.saved, [[10, 20, 30]], 'only the newest list is saved')
  assert.deepEqual(f.settled, [{ items: 3, known: 3, unresolved: 0 }])
})

test('a tick the engine refuses is taken back and the error is shown', async () => {
  const f = fake([10])
  const done = tick(20, f.deps, f.state)
  assert.deepEqual(f.owned(), [10, 20])
  f.replies[0].reject(new Error('The engine crashed and was restarted — try again.'))
  await done
  assert.deepEqual(f.owned(), [10])
  assert.deepEqual(f.failed, [{ previous: [10], error: 'Error: The engine crashed and was restarted — try again.', ticked: false }])
  assert.deepEqual(f.saved, [])
})

test('a refused tick does not undo a newer one', async () => {
  const f = fake([])
  const first = tick(1, f.deps, f.state)
  const second = tick(2, f.deps, f.state)
  f.replies[0].reject(new Error('boom'))
  f.replies[1].resolve({ items: 2, known: 2 })
  await Promise.all([first, second])
  assert.deepEqual(f.owned(), [1, 2])
  assert.deepEqual(f.failed, [])
  assert.deepEqual(f.saved, [[1, 2]])
})

test('two refused ticks go back to the list the engine still has', async () => {
  const f = fake([10])
  const first = tick(30, f.deps, f.state)
  const second = tick(20, f.deps, f.state)
  f.replies[0].reject(new Error('boom'))
  f.replies[1].reject(new Error('boom'))
  await Promise.all([first, second])
  assert.deepEqual(f.owned(), [10])
  assert.deepEqual(f.failed, [{ previous: [10], error: 'Error: boom', ticked: false }])
  assert.deepEqual(f.saved, [])
})

test('a refused tick after an accepted one keeps and saves the accepted list', async () => {
  const f = fake([10])
  const first = tick(30, f.deps, f.state)
  const second = tick(20, f.deps, f.state)
  f.replies[0].resolve({ items: 2, known: 2 })
  await first
  f.replies[1].reject(new Error('boom'))
  await second
  assert.deepEqual(f.owned(), [10, 30])
  assert.deepEqual(f.failed, [{ previous: [10, 30], error: 'Error: boom', ticked: true }])
  assert.deepEqual(f.saved, [[10, 30]])
  assert.deepEqual(f.settled, [{ items: 2, known: 2, unresolved: 0 }])
})

test('a new tick after the engine settles starts from the shown list', async () => {
  const f = fake([10])
  const first = tick(30, f.deps, f.state)
  f.replies[0].resolve({ items: 2, known: 2 })
  await first
  const second = tick(20, f.deps, f.state)
  f.replies[1].reject(new Error('boom'))
  await second
  assert.deepEqual(f.owned(), [10, 30])
  assert.deepEqual(f.failed, [{ previous: [10, 30], error: 'Error: boom', ticked: false }])
  assert.deepEqual(f.saved, [[10, 30]])
})
