import type { IdealTable } from './stages'

type Reply = { ok: boolean; status: number; statusText: string; json(): Promise<unknown> }

export type IdealLoad = { table: IdealTable | null; problem: string | null; fatal: boolean }

export async function loadIdeals(
  get: (url: string) => Promise<Reply>,
  url: string,
  version: string,
  pause: (ms: number) => Promise<void>,
  retryMs = 1000,
): Promise<IdealLoad> {
  const first = await attempt(get, url, version)
  if (!first.problem || first.fatal) return first
  await pause(retryMs)
  const second = await attempt(get, url, version)
  return second.problem ? { ...second, problem: `${second.problem} (after a retry)` } : second
}

async function attempt(get: (url: string) => Promise<Reply>, url: string, version: string): Promise<IdealLoad> {
  let r: Reply
  try {
    r = await get(url)
  } catch (e) {
    return failed(`network error: ${String(e)}`)
  }
  if (!r.ok) return failed(`HTTP ${r.status} ${r.statusText}`.trim())
  let file: unknown
  try {
    file = await r.json()
  } catch (e) {
    return failed(`unreadable JSON: ${String(e)}`)
  }
  if (!file || typeof file !== 'object') return failed('unreadable JSON: not an object')
  const { version: built, stages } = file as { version?: unknown; stages?: unknown }
  if (built !== version) return { table: null, problem: `built for data ${String(built)}, the app uses ${version}`, fatal: true }
  if (!stages || typeof stages !== 'object') return failed('unreadable JSON: no stages')
  return { table: stages as IdealTable, problem: null, fatal: false }
}

function failed(problem: string): IdealLoad {
  return { table: null, problem, fatal: false }
}
