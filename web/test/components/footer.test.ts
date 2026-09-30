import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

type Source = { id: string; name: string; status: string; licence?: string }

const footer = readFileSync(new URL('../../src/components/Footer.tsx', import.meta.url), 'utf8').replace(/\s+/g, ' ')
const sources: Source[] = JSON.parse(readFileSync(new URL('../../../data/sources.json', import.meta.url), 'utf8')).sources
const COUNT = ['no', 'one', 'two', 'three', 'four', 'five', 'six', 'seven']

test('the footer names the licensed and permitted sources the data ships with', () => {
  for (const s of sources.filter((x) => x.status === 'redistributable' || x.status === 'permitted')) {
    assert.ok(footer.includes(s.name), `the footer does not name ${s.name}`)
  }
  assert.ok(footer.includes('CC BY-SA 3.0'))
  assert.ok(footer.includes('(used with permission)'))
})

test('the footer counts the sources with no licence located as the registry does', () => {
  const unlicensed = sources.filter((s) => s.status === 'permission-required').length
  const noun = unlicensed === 1 ? 'source' : 'sources'
  assert.ok(footer.includes(`and ${COUNT[unlicensed]} community ${noun} with no licence located.`), footer)
})
