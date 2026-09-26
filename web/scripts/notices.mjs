import { execSync } from 'node:child_process'
import { existsSync, readFileSync, readdirSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'

const out = ['Third-party software in NikkiBase', '='.repeat(33), '']

const goroot = execSync('go env GOROOT', { encoding: 'utf8' }).trim()
const goLicence = [join(goroot, 'LICENSE'), join(goroot, '..', 'LICENSE')].find(existsSync)
if (!goLicence) throw new Error(`notices: no Go LICENSE under ${goroot}`)
out.push('Go runtime and wasm_exec.js (BSD-3-Clause)', '-'.repeat(42), readFileSync(goLicence, 'utf8').trim(), '')

const vite = JSON.parse(readFileSync('node_modules/vite/package.json', 'utf8'))
out.push(`vite@${vite.version} module preload helper (${vite.license})`, '-'.repeat(40), readFileSync('node_modules/vite/LICENSE.md', 'utf8').trim(), '')

const paths = execSync('npm ls --omit=dev --all --parseable', { encoding: 'utf8' })
  .split('\n')
  .filter((p) => p.includes('node_modules'))
const seen = new Set()
for (const dir of paths.sort()) {
  const pkg = JSON.parse(readFileSync(join(dir, 'package.json'), 'utf8'))
  const id = `${pkg.name}@${pkg.version}`
  if (seen.has(id)) continue
  seen.add(id)
  const file = readdirSync(dir).find((f) => /^(licen[cs]e|copying)(\.|$)/i.test(f))
  const licence = typeof pkg.license === 'string' ? pkg.license : (pkg.license?.type ?? 'unknown')
  out.push(`${id} (${licence})`, '-'.repeat(id.length + licence.length + 3))
  out.push(file ? readFileSync(join(dir, file), 'utf8').trim() : `No licence file shipped; package.json declares ${licence}.`, '')
}

if (!existsSync('dist')) throw new Error('notices: run after vite build, from web/')
writeFileSync('dist/third-party-notices.txt', out.join('\n') + '\n')
writeFileSync('dist/privacy.txt', readFileSync('../PRIVACY.md', 'utf8'))
console.log(`notices: ${seen.size} packages + the Go runtime + Vite's preload helper -> dist/third-party-notices.txt`)
