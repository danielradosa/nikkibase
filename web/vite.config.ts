import { createHash } from 'node:crypto'
import { existsSync, readFileSync } from 'node:fs'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

function dataVersion(): string {
  const index = new URL('./public/data/index.json', import.meta.url)
  try {
    return JSON.parse(readFileSync(index, 'utf8')).version
  } catch (err) {
    throw new Error(`no data bundle at ${index.pathname}: build one with cmd/bundle (see SOURCES.md): ${err}`)
  }
}

function keystreamUrl(): string {
  const key = new URL('./public/keystream.bin', import.meta.url)
  if (!existsSync(key)) return '/keystream.bin'
  const hash = createHash('sha256').update(readFileSync(key)).digest('hex').slice(0, 16)
  return `/keystream.bin?v=${hash}`
}

export default defineConfig({
  plugins: [react()],
  worker: { format: 'es' },
  define: {
    __DATA_VERSION__: JSON.stringify(dataVersion()),
    __KEYSTREAM_URL__: JSON.stringify(keystreamUrl()),
  },
})
