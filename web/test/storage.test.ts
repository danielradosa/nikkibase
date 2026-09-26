import { test } from 'node:test'
import assert from 'node:assert/strict'
import { classify } from '../src/storage.ts'

const V = '2026-09-22'
const record = (source: string, version = V) => ({ version, ids: [10001, 20001], source, savedAt: 0 })

test('current sources load as they are', () => {
  for (const source of ['sel', 'manual', 'clothes_date']) {
    const r = classify(record(source), V)
    assert.equal(r.status, 'ok')
    assert.equal(r.status === 'ok' && r.entry.source, source)
  }
})

test('an unknown source is reported, not guessed at', () => {
  assert.equal(classify(record('something-else'), V).status, 'unrecognised')
})

test('another catalogue version is stale whatever its source', () => {
  assert.equal(classify(record('clothes_date', '2026-07-29'), V).status, 'stale')
  assert.equal(classify(record('sel', '2026-07-29'), V).status, 'stale')
})

test('empty or malformed records are nothing', () => {
  for (const raw of [null, undefined, 'x', {}, { version: V, ids: [], source: 'sel' }]) {
    assert.equal(classify(raw, V).status, 'none')
  }
})
