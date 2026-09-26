import { engine } from './engine'
import idealUrl from './generated/ideal.json?url'
import { loadIdeals } from './idealLoad'
import { parseItems, type ItemPlaces, type Row } from './items'
import type { Place } from './comparison'
import type { IdealTable, Stage } from './stages'
import type { AcquireFile, AcquireTable } from './worth'

export const version = __DATA_VERSION__

function load<T>(name: string): Promise<T> {
  return fetch(`/data/${version}/${name}`).then((r) =>
    r.ok ? (r.json() as Promise<T>) : Promise.reject(new Error(`${name}: ${r.status} ${r.statusText}`)),
  )
}

export const engineReady = engine.init(version)

export const startup = Promise.all([
  load<Stage[]>('stages.json'),
  load<string[]>('tags.json'),
  load<Place[]>('positions.json'),
]).then(([stages, tagNames, places]) => ({ stages, tagNames, places }))

function itemPlaces(): Promise<ItemPlaces | null> {
  return engine.places().catch((e) => {
    console.warn(`item places: ${e instanceof Error ? e.message : e}. Items show their slot instead.`)
    return null
  })
}

export const ideals: Promise<IdealTable | null> = engineReady
  .then(() => loadIdeals((url) => fetch(url), idealUrl, version, (ms) => new Promise((r) => setTimeout(r, ms))))
  .then(({ table, problem, fatal }) => {
    if (problem) (fatal ? console.error : console.warn)(`ideal.json: ${problem}. Best possible is computed live, which is slower.`)
    return table
  })
  .catch(() => null)

export const items = engineReady
  .then(() => ideals)
  .then(() => Promise.all([load<{ items: Row[] }>('items.json'), itemPlaces()]))
  .then(([data, places]) => parseItems(data.items, places))

let acquireLoad: Promise<AcquireTable | null> | null = null

export function loadAcquire(): Promise<AcquireTable | null> {
  acquireLoad ??= load<AcquireFile>('acquire.json')
    .then((file) => {
      if (file.version === version && file.items) return file.items
      console.error(`acquire.json: version ${file.version} does not match ${version}.`)
      return null
    })
    .catch((e) => {
      console.warn(`acquire.json: ${e instanceof Error ? e.message : e}. How to get each item is left out.`)
      acquireLoad = null
      return null
    })
  return acquireLoad
}
