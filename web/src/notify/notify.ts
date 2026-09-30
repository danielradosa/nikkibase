export const TOPICS = ['items', 'stages', 'fixes'] as const
export type Topic = (typeof TOPICS)[number]

export const TOPIC_LABELS: Record<Topic, string> = {
  items: 'New items',
  stages: 'New stages',
  fixes: 'Fixes to the data',
}

export const TOPICS_KEY = 'nikkibase.notifyTopics'
export const SYNC_KEY = 'nikkibase.notifySync'
export const SYNC_MS = 24 * 60 * 60 * 1000

export type Support = 'yes' | 'install' | 'no'

export type Env = {
  serviceWorker: boolean
  pushManager: boolean
  notification: boolean
  apple: boolean
  standalone: boolean
}

export function supportOf(env: Env): Support {
  if (env.serviceWorker && env.pushManager && env.notification) return 'yes'
  if (env.apple && !env.standalone) return 'install'
  return 'no'
}

export function isApple(userAgent: string, platform: string, touchPoints: number): boolean {
  return /iPad|iPhone|iPod/.test(userAgent) || (platform === 'MacIntel' && touchPoints > 1)
}

export function cleanTopics(list: readonly unknown[]): Topic[] {
  return TOPICS.filter((t) => list.includes(t))
}

export function parseTopics(raw: string | null): Topic[] {
  if (!raw) return []
  try {
    const list = JSON.parse(raw)
    return Array.isArray(list) ? cleanTopics(list) : []
  } catch {
    return []
  }
}

export function withTopic(topics: readonly Topic[], topic: Topic, on: boolean): Topic[] {
  return cleanTopics(on ? [...topics, topic] : topics.filter((t) => t !== topic))
}

export type Sync = { endpoint: string; at: number }

export function parseSync(raw: string | null): Sync | null {
  if (!raw) return null
  try {
    const v = JSON.parse(raw)
    return v && typeof v.endpoint === 'string' && typeof v.at === 'number' ? { endpoint: v.endpoint, at: v.at } : null
  } catch {
    return null
  }
}

export function needsSync(last: Sync | null, endpoint: string, now: number): boolean {
  return !last || last.endpoint !== endpoint || now - last.at >= SYNC_MS || now < last.at
}

export function keyBytes(base64url: string): Uint8Array {
  const b64 = base64url.replace(/-/g, '+').replace(/_/g, '/')
  const padded = b64 + '='.repeat((4 - (b64.length % 4)) % 4)
  const bin = atob(padded)
  const out = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i)
  return out
}

export function sameKey(a: ArrayBuffer | null | undefined, b: Uint8Array): boolean {
  if (!a) return false
  const x = new Uint8Array(a)
  return x.length === b.length && x.every((v, i) => v === b[i])
}

export type Status =
  | { kind: 'off' }
  | { kind: 'on'; topics: Topic[] }
  | { kind: 'busy' }
  | { kind: 'denied' }
  | { kind: 'install' }
  | { kind: 'failed'; offline: boolean }

export const NOTIFY_INTRO = 'Get a notification when NikkiBase gets new data.'
export const NOTIFY_PRIVACY =
  "Your browser's push service delivers them. NikkiBase keeps only the push address your browser gives it and these choices. Turning everything off deletes them."

export function statusText(s: Status): string {
  switch (s.kind) {
    case 'off':
      return 'Off. Pick what you want to hear about.'
    case 'on':
      return s.topics.length === TOPICS.length ? 'On for everything.' : `On for ${s.topics.map((t) => TOPIC_LABELS[t].toLowerCase()).join(' and ')}.`
    case 'busy':
      return 'Saving…'
    case 'denied':
      return "Notifications are blocked for NikkiBase in this browser's settings. Allow them there, then try again."
    case 'install':
      return 'On iPhone and iPad, add NikkiBase to your Home Screen first: tap Share, then Add to Home Screen. Open it from there to turn notifications on.'
    case 'failed':
      return s.offline ? "You're offline. Try again when you're connected." : "Couldn't save that. Try again in a moment."
  }
}
