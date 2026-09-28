import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const web = new URL('../', import.meta.url)
const html = readFileSync(new URL('index.html', web), 'utf8')
const file = (name: string) => readFileSync(new URL(`public/${name}`, web))

test('the page links the tab icon, its fallback and the home-screen icon', () => {
  assert.match(html, /<link rel="icon" href="\/favicon\.ico" sizes="32x32" \/>/)
  assert.match(html, /<link rel="icon" href="\/favicon\.svg" type="image\/svg\+xml" \/>/)
  assert.match(html, /<link rel="apple-touch-icon" href="\/apple-touch-icon\.png" \/>/)
})

test('favicon.svg is a square vector with its colours on every stop, no script and nothing from elsewhere', () => {
  const svg = file('favicon.svg').toString('utf8')
  assert.match(svg, /^<svg xmlns="http:\/\/www\.w3\.org\/2000\/svg" viewBox="(-?\d+) \1 (\d+) \2"/)
  assert.doesNotMatch(svg, /<stop(?![^>]*stop-color=)/)
  assert.doesNotMatch(svg, /<script|<image|href="(?!#)/)
})

test('favicon.ico holds one 32 by 32 PNG', () => {
  const ico = file('favicon.ico')
  assert.deepEqual([...ico.subarray(0, 8)], [0, 0, 1, 0, 1, 0, 32, 32])
  assert.equal(ico.readUInt32LE(14), ico.length - 22)
  assert.equal(ico.readUInt32LE(18), 22)
  assert.equal(ico.subarray(22, 30).toString('latin1'), '\x89PNG\r\n\x1a\n')
})

test('apple-touch-icon.png is 180 by 180 and has no transparency', () => {
  const png = file('apple-touch-icon.png')
  assert.equal(png.subarray(0, 8).toString('latin1'), '\x89PNG\r\n\x1a\n')
  assert.equal(png.readUInt32BE(16), 180)
  assert.equal(png.readUInt32BE(20), 180)
  assert.ok([2, 3].includes(png[25]))
  assert.equal(png.indexOf('tRNS'), -1)
})
