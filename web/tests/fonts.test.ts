import { test } from 'node:test'
import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import { join } from 'node:path'

const assets = join(import.meta.dirname, '../src/assets')
const css = readFileSync(join(assets, 'fonts.css'), 'utf8')
const html = readFileSync(join(import.meta.dirname, '../index.html'), 'utf8')

test('fontes: servidas pelo próprio app, sem CDN externo no index.html', () => {
  assert.doesNotMatch(html, /fonts\.(googleapis|gstatic)\.com/)
})

test('fontes: toda url() do fonts.css aponta para um woff2 existente', () => {
  const urls = [...css.matchAll(/url\('([^']+)'\)/g)].map((m) => m[1])
  assert.ok(urls.length >= 2)
  for (const u of urls) assert.ok(existsSync(join(assets, u)), `arquivo ausente: ${u}`)
})

test('fontes: as duas famílias cobrem os pesos usados (Bricolage 400–800, DM Mono 400 e 500)', () => {
  const faces = [...css.matchAll(/font-family: '([^']+)';[^}]*?font-weight: ([\d ]+);/g)].map((m) => `${m[1]}:${m[2]}`)
  assert.ok(faces.includes('Bricolage Grotesque:400 800'))
  assert.ok(faces.includes('DM Mono:400'))
  assert.ok(faces.includes('DM Mono:500'))
})
