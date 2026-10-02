import { SITE_HOST } from '../outfit/skills'
import { fileName, wardrobeFile } from './wardrobeFile'

const REVOKE_AFTER = 40000

export function download(text: string, name: string) {
  const url = URL.createObjectURL(new Blob([text], { type: 'text/plain;charset=utf-8' }))
  const link = document.createElement('a')
  link.href = url
  link.download = name
  link.hidden = true
  document.body.append(link)
  link.click()
  link.remove()
  setTimeout(() => URL.revokeObjectURL(url), REVOKE_AFTER)
}

export function saveWardrobeFile(ids: readonly number[]) {
  const now = new Date()
  download(wardrobeFile(ids, now, SITE_HOST), fileName(now))
}
