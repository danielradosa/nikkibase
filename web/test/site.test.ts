import { test } from 'node:test'
import assert from 'node:assert/strict'
import { SITE_HOST } from '../src/site.ts'

test('the copied outfit ends on the site address', () => {
  assert.equal(SITE_HOST, 'nikkibase.up.railway.app')
})
