import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'

const web = new URL('../', import.meta.url)
const html = readFileSync(new URL('index.html', web), 'utf8')
const file = (name: string) => readFileSync(new URL(`public/${name}`, web))

function png(name: string) {
  const body = file(name)
  assert.equal(body.subarray(0, 8).toString('latin1'), '\x89PNG\r\n\x1a\n', `${name} is not a PNG`)
  const chunks: string[] = []
  for (let at = 8; at < body.length; at += 12 + body.readUInt32BE(at)) chunks.push(body.subarray(at + 4, at + 8).toString('latin1'))
  return { width: body.readUInt32BE(16), height: body.readUInt32BE(20), colourType: body[25], chunks }
}

function assertOpaque(name: string) {
  const { colourType, chunks } = png(name)
  assert.ok([2, 3].includes(colourType), `${name} has colour type ${colourType}, which carries alpha`)
  assert.ok(!chunks.includes('tRNS'), `${name} has a transparency chunk`)
}

type Icon = { src: string; sizes: string; type: string; purpose: string }

const manifest = JSON.parse(file('manifest.webmanifest').toString('utf8')) as {
  name: string
  short_name: string
  description: string
  start_url: string
  scope: string
  display: string
  theme_color: string
  background_color: string
  icons: Icon[]
}

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
  const { width, height } = png('apple-touch-icon.png')
  assert.equal(width, 180)
  assert.equal(height, 180)
  assertOpaque('apple-touch-icon.png')
})

test('the page links the app manifest', () => {
  assert.match(html, /<link rel="manifest" href="\/manifest\.webmanifest" \/>/)
})

test('the manifest opens NikkiBase on its own at the root, in the page blush', () => {
  const description = html.match(/<meta\s+name="description"\s+content="([^"]+)"/)?.[1]
  const themeColour = html.match(/<meta name="theme-color" content="([^"]+)"/)?.[1]
  assert.equal(manifest.name, 'NikkiBase')
  assert.equal(manifest.short_name, 'NikkiBase')
  assert.equal(manifest.description, description)
  assert.equal(manifest.start_url, '/')
  assert.equal(manifest.scope, '/')
  assert.equal(manifest.display, 'standalone')
  assert.equal(manifest.theme_color, '#fdeef4')
  assert.equal(manifest.theme_color, themeColour)
  assert.equal(manifest.background_color, '#fdeef4')
})

test('the manifest lists a 192 and a 512 icon and a 512 maskable one', () => {
  assert.deepEqual(
    manifest.icons.map((icon) => `${icon.sizes} ${icon.purpose}`),
    ['192x192 any', '512x512 any', '512x512 maskable'],
  )
})

test('every manifest icon is an opaque PNG of the size it declares', () => {
  for (const icon of manifest.icons) {
    assert.match(icon.src, /^\/icons\/[\w-]+\.png$/)
    assert.equal(icon.type, 'image/png')
    const name = icon.src.slice(1)
    const { width, height } = png(name)
    assert.equal(`${width}x${height}`, icon.sizes, `${name} is ${width}x${height}`)
    assertOpaque(name)
  }
})
