import { test } from 'node:test'
import assert from 'node:assert/strict'
import { loadIdeals } from '../src/idealLoad.ts'
import type { IdealTable } from '../src/stages.ts'

const URL = '/assets/ideal-abc.json'
const stages: IdealTable = { 'Story/1-1': { score: 119118, items: [[10001, 0]] } }

type Step = { ok: boolean; status?: number; statusText?: string; body?: unknown; badJson?: boolean } | Error

function server(...steps: Step[]) {
  const calls: string[] = []
  const pauses: number[] = []
  const get = async (url: string) => {
    calls.push(url)
    const step = steps[Math.min(calls.length, steps.length) - 1]
    if (step instanceof Error) throw step
    return {
      ok: step.ok,
      status: step.status ?? (step.ok ? 200 : 500),
      statusText: step.statusText ?? '',
      json: async () => {
        if (step.badJson) throw new SyntaxError('Unexpected end of JSON input')
        return step.body
      },
    }
  }
  const pause = async (ms: number) => {
    pauses.push(ms)
  }
  return { get, pause, calls, pauses }
}

const good: Step = { ok: true, body: { version: '2026-09-24', stages } }

test('a good table loads on the first try with no problem and no pause', async () => {
  const s = server(good)
  const got = await loadIdeals(s.get, URL, '2026-09-24', s.pause)
  assert.deepEqual(got, { table: stages, problem: null, fatal: false })
  assert.deepEqual(s.calls, [URL])
  assert.deepEqual(s.pauses, [])
})

test('a network error is retried once after the pause, and the second try wins', async () => {
  const s = server(new TypeError('Failed to fetch'), good)
  const got = await loadIdeals(s.get, URL, '2026-09-24', s.pause, 1000)
  assert.deepEqual(got, { table: stages, problem: null, fatal: false })
  assert.deepEqual(s.calls, [URL, URL])
  assert.deepEqual(s.pauses, [1000])
})

test('two failures give no table and say why, so the engine computes best possible', async () => {
  const s = server({ ok: false, status: 503, statusText: 'Service Unavailable' })
  const got = await loadIdeals(s.get, URL, '2026-09-24', s.pause)
  assert.equal(got.table, null)
  assert.equal(got.fatal, false)
  assert.equal(got.problem, 'HTTP 503 Service Unavailable (after a retry)')
  assert.equal(s.calls.length, 2)
})

test('a truncated file counts as a failure and is retried', async () => {
  const s = server({ ok: true, badJson: true }, good)
  const got = await loadIdeals(s.get, URL, '2026-09-24', s.pause)
  assert.equal(got.table, stages)
  assert.equal(s.calls.length, 2)
  const twice = server({ ok: true, badJson: true })
  const failed = await loadIdeals(twice.get, URL, '2026-09-24', twice.pause)
  assert.match(failed.problem ?? '', /^unreadable JSON: SyntaxError: Unexpected end of JSON input \(after a retry\)$/)
})

test('a table built for another data version is fatal and never retried', async () => {
  const s = server({ ok: true, body: { version: '2026-09-23', stages } })
  const got = await loadIdeals(s.get, URL, '2026-09-24', s.pause)
  assert.deepEqual(got, { table: null, problem: 'built for data 2026-09-23, the app uses 2026-09-24', fatal: true })
  assert.equal(s.calls.length, 1)
  assert.deepEqual(s.pauses, [])
})

test('a file that is not a table is a failure, not a crash', async () => {
  for (const body of [null, 'x', { version: '2026-09-24' }, { version: '2026-09-24', stages: 3 }]) {
    const s = server({ ok: true, body })
    const got = await loadIdeals(s.get, URL, '2026-09-24', s.pause)
    assert.equal(got.table, null)
    assert.equal(got.fatal, false)
    assert.match(got.problem ?? '', /^unreadable JSON: .+ \(after a retry\)$/)
  }
})
