import type { Alternative, Outfit } from './engine'
import type { Ideal } from './stages'

export type Place = { name: string; slot: number }

export type ComparisonRow = {
  key: number
  slot: string
  mine: string | null
  unworn: string
  best: string | null
  same: boolean
  alts: Alternative[]
  moreAlts: boolean
  owned?: number
}

export const CLOSE_CALL = 0.05

export function unwornLabel(pos: number, ownedPlaces: ReadonlySet<number>): string {
  return ownedPlaces.has(pos) ? 'not worn' : 'nothing owned'
}

export function alternativesLabel(worn: boolean, listed: number, more = false): string {
  if (!worn) return '—'
  if (listed === 0) return 'none owned'
  if (more) return `${listed}+ others`
  return listed === 1 ? '1 other' : `${listed} others`
}

export function closeCall(o: { dress: number; separates: number } | null): number | null {
  if (!o || o.dress <= 0 || o.separates <= 0) return null
  const gap = Math.abs(o.dress - o.separates) / Math.max(o.dress, o.separates)
  return gap < CLOSE_CALL ? gap : null
}

export function closeCallText(gap: number): string {
  return `Close call: your best dress and best top + bottom are within ${Math.max(1, Math.round(gap * 100))}%. Either could win in game, so try both.`
}

export function comparisonRows(
  outfit: Outfit | null,
  ideal: Ideal | null,
  names: ReadonlyMap<number, string>,
  places: readonly Place[],
  slots: readonly string[],
): ComparisonRow[] {
  const idealByPlace = new Map<number, number>()
  for (const it of ideal?.items ?? []) if (!idealByPlace.has(it.pos)) idealByPlace.set(it.pos, it.id)
  const held = new Set(outfit?.ownedPlaces)
  const filled = new Set<number>()
  for (const it of outfit?.items ?? []) filled.add(it.pos)
  for (const it of ideal?.items ?? []) filled.add(it.pos)
  return [...filled]
    .sort((a, b) => a - b)
    .map((pos) => {
      const mine = (outfit?.items ?? []).find((it) => it.pos === pos)
      const best = idealByPlace.get(pos)
      const place = places[pos]
      return {
        key: pos,
        slot: place?.name ?? slots[mine?.slot ?? -1] ?? `#${pos}`,
        mine: mine ? (names.get(mine.id) ?? `#${mine.id}`) : null,
        unworn: unwornLabel(pos, held),
        best: best !== undefined ? (names.get(best) ?? `#${best}`) : null,
        same: mine?.id === best,
        alts: mine?.alts ?? [],
        moreAlts: mine?.moreAlts ?? false,
        owned: mine?.id,
      }
    })
}

export function outfitText(
  outfit: Outfit | null,
  ideal: Ideal | null,
  stage: { mode: string; name: string } | null,
  difficulty: string,
  names: ReadonlyMap<number, string>,
  places: readonly Place[],
  slots: readonly string[],
  skills: string,
): string {
  if (!outfit || !stage) return ''
  const level = stage.mode === 'Story' ? ` (${difficulty})` : ''
  const lines = [
    `${stage.mode} ${stage.name}${level} — ${outfit.score.toLocaleString('en-US')} (${outfit.items.length} items)`,
  ]
  for (const it of [...outfit.items].sort((a, b) => a.pos - b.pos)) {
    lines.push(`${places[it.pos]?.name ?? slots[it.slot] ?? `#${it.pos}`}: ${names.get(it.id) ?? `#${it.id}`}`)
  }
  if (ideal) lines.push(`best possible — ${ideal.score.toLocaleString('en-US')}`)
  lines.push(skills)
  return lines.join('\n')
}

export function bestNote(row: Pick<ComparisonRow, 'best' | 'same'>): string {
  if (row.same) return 'best possible'
  return `Best: ${row.best ?? '—'}`
}

export function rowOpens(row: Pick<ComparisonRow, 'same' | 'alts'>, phone: boolean): boolean {
  return row.alts.length > 0 || (phone && !row.same)
}

export type CopyLine = { label: string; name: string }

export function copyLines(row: Pick<ComparisonRow, 'mine' | 'best' | 'same'>): CopyLine[] {
  const lines: CopyLine[] = []
  if (row.mine) lines.push({ label: 'Your best', name: row.mine })
  if (row.best && !row.same) lines.push({ label: 'Best possible', name: row.best })
  return lines
}

export function expandLabel(slot: string): string {
  return `Show alternatives for ${slot}`
}
