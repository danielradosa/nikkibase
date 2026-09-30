import { test } from 'node:test'
import assert from 'node:assert/strict'
import { TOPICS, cleanTopics, isApple, keyBytes, parseTopics, sameKey, statusText, supportOf, withTopic } from '../../src/notify/notify.ts'

const full = { serviceWorker: true, pushManager: true, notification: true, apple: false, standalone: false }

test('notifications need a service worker, push and the notification API', () => {
  assert.equal(supportOf(full), 'yes')
  assert.equal(supportOf({ ...full, pushManager: false }), 'no')
  assert.equal(supportOf({ ...full, notification: false }), 'no')
  assert.equal(supportOf({ ...full, serviceWorker: false }), 'no')
})

test('on iPhone and iPad in the browser, the answer is to install the app first', () => {
  const safari = { serviceWorker: true, pushManager: false, notification: false, apple: true, standalone: false }
  assert.equal(supportOf(safari), 'install')
  assert.equal(supportOf({ ...safari, standalone: true }), 'no')
  assert.equal(supportOf({ ...safari, standalone: true, pushManager: true, notification: true }), 'yes')
})

test('Apple phones and tablets are recognised, including iPads that say they are Macs', () => {
  assert.equal(isApple('Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X)', 'iPhone', 5), true)
  assert.equal(isApple('Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)', 'MacIntel', 5), true)
  assert.equal(isApple('Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)', 'MacIntel', 0), false)
  assert.equal(isApple('Mozilla/5.0 (Linux; Android 14)', 'Linux armv8l', 5), false)
})

test('saved topics are read in a fixed order and anything unknown is dropped', () => {
  assert.deepEqual(parseTopics('["fixes","items","ads",3]'), ['items', 'fixes'])
  assert.deepEqual(parseTopics(null), [])
  assert.deepEqual(parseTopics('{'), [])
  assert.deepEqual(parseTopics('{"items":true}'), [])
  assert.deepEqual(cleanTopics(['stages', 'stages']), ['stages'])
})

test('switching a topic keeps the order and never doubles it', () => {
  assert.deepEqual(withTopic([], 'fixes', true), ['fixes'])
  assert.deepEqual(withTopic(['fixes'], 'items', true), ['items', 'fixes'])
  assert.deepEqual(withTopic(['items', 'fixes'], 'items', true), ['items', 'fixes'])
  assert.deepEqual(withTopic(['items', 'fixes'], 'items', false), ['fixes'])
  assert.deepEqual(withTopic(['items'], 'stages', false), ['items'])
})

test('the server key is decoded from base64url and compared byte for byte', () => {
  const key = keyBytes('BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4')
  assert.equal(key.length, 65)
  assert.equal(key[0], 4)
  assert.deepEqual([...keyBytes('_-8')], [255, 239])
  assert.equal(sameKey(key.slice().buffer, key), true)
  const other = key.slice()
  other[64] ^= 1
  assert.equal(sameKey(other.buffer, key), false)
  assert.equal(sameKey(key.slice(0, 64).buffer, key), false)
  assert.equal(sameKey(null, key), false)
})

test('the status line says what is on, in the order of the switches', () => {
  assert.equal(statusText({ kind: 'on', topics: [...TOPICS] }), 'On for everything.')
  assert.equal(statusText({ kind: 'on', topics: ['items', 'fixes'] }), 'On for new items and fixes to the data.')
  assert.equal(statusText({ kind: 'on', topics: ['stages'] }), 'On for new stages.')
  assert.match(statusText({ kind: 'denied' }), /blocked/)
  assert.match(statusText({ kind: 'install' }), /Home Screen/)
  assert.match(statusText({ kind: 'failed', offline: true }), /offline/)
  assert.match(statusText({ kind: 'failed', offline: false }), /Try again/)
})
