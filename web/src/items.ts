import type { Place } from './comparison'

export const SLOTS = [
  'hair', 'dress', 'coat', 'top', 'bottom', 'hosiery', 'shoes', 'makeup', 'accessory', 'spirit',
]
export const ATTRS = [
  'Gorgeous', 'Simple', 'Elegant', 'Lively', 'Mature', 'Cute', 'Sexy', 'Pure', 'Warm', 'Cool',
]

export type Item = {
  id: number
  name: string
  slot: number
  place: number
  attrs: number[]
  grades: string[]
  search: string
  scoreable: boolean
  rarity: number
  suit: string
}

export type Row = [number, string, number, ...number[]]
export type ItemPlaces = { ids: number[]; places: number[] }

export function parseItems(rows: Row[], places: ItemPlaces | null = null): Item[] {
  const placeOf = new Map<number, number>()
  places?.ids.forEach((id, i) => placeOf.set(id, places.places[i]))
  return rows.map((r) => {
    const grades = r.slice(8, 13) as unknown as string[]
    return {
      id: r[0] as number,
      name: r[1] as string,
      slot: r[2] as number,
      place: placeOf.get(r[0] as number) ?? -1,
      attrs: r.slice(3, 8) as number[],
      grades,
      search: (r[1] as string).toLowerCase(),
      scoreable: grades.some(Boolean),
      rarity: (r[13] as number) ?? 0,
      suit: (r[14] as unknown as string) ?? '',
    }
  })
}

export const ANY_SLOT = 'any'

type Choice = { value: string; label: string }
export type SlotOption = Choice | { label: string; options: Choice[] }

export function slotOptions(places: readonly Place[]): SlotOption[] {
  const options: SlotOption[] = [{ value: ANY_SLOT, label: 'any slot' }]
  SLOTS.forEach((slot, s) => {
    const own = places.flatMap((p, place) => (p.slot === s ? [{ value: `p${place}`, label: p.name.toLowerCase() }] : []))
    if (own.length < 2) options.push({ value: `s${s}`, label: slot })
    else options.push({ label: slot, options: [{ value: `s${s}`, label: `any ${slot}` }, ...own] })
  })
  return options
}

export function slotChoice(choice: string): { slots?: number[]; places?: number[] } {
  const n = Number(choice.slice(1))
  if (!Number.isInteger(n) || n < 0 || choice.length < 2) return {}
  if (choice[0] === 's') return { slots: [n] }
  if (choice[0] === 'p') return { places: [n] }
  return {}
}

export function inChoice(choice: string, it: Pick<Item, 'slot' | 'place'>): boolean {
  const { slots, places } = slotChoice(choice)
  return (!slots || slots.includes(it.slot)) && (!places || places.includes(it.place))
}

export function placeName(it: Pick<Item, 'slot' | 'place'>, places: readonly Place[]): string {
  const place = places[it.place]
  if (place) return place.name
  const slot = SLOTS[it.slot] ?? `slot ${it.slot}`
  return slot.charAt(0).toUpperCase() + slot.slice(1)
}

export function itemsTabLabel(count: number | null, phone: boolean): string {
  return count === null || phone ? 'Items' : `Items (${count.toLocaleString('en-US')})`
}

export const ITEM_STEP = 50

export type ItemPage = { filters: string; shown: number }

export function itemPage(page: ItemPage, filters: string): ItemPage {
  return page.filters === filters ? page : { filters, shown: ITEM_STEP }
}

export function gradesLine(it: Pick<Item, 'attrs' | 'grades'>): string {
  return it.grades.flatMap((grade, pair) => (grade ? [`${ATTRS[it.attrs[pair]]}\u00a0${grade}`] : [])).join('\u00a0· ')
}

export const ownLabel = (name: string) => `Own ${name}`

export function tickHint(shown: boolean, manual: boolean, onItems: boolean): boolean {
  return onItems ? shown : manual
}

export function moreItemsText(total: number, shown: number): string | null {
  const left = total - shown
  return left > 0 ? `Show ${Math.min(ITEM_STEP, left).toLocaleString('en-US')} more` : null
}
