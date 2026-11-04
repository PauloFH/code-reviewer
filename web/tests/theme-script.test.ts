import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const html = readFileSync(join(import.meta.dirname, '../index.html'), 'utf8')
// Script inline que roda antes do módulo principal (sem src, sem type=module).
const inline = [...html.matchAll(/<script>([\s\S]*?)<\/script>/g)].map((m) => m[1])
assert.equal(inline.length, 1, 'esperado exatamente um script inline em index.html')

function run(localStorage: { getItem(k: string): string | null }) {
  const attrs: Record<string, string> = {}
  const document = {
    documentElement: { setAttribute: (k: string, v: string) => (attrs[k] = v) },
  }
  new Function('document', 'localStorage', inline[0])(document, localStorage)
  return attrs
}

test('tema salvo "light" define data-theme', () => {
  assert.deepEqual(run({ getItem: () => 'light' }), { 'data-theme': 'light' })
})

test('sem valor salvo ou "dark" não define atributo', () => {
  assert.deepEqual(run({ getItem: () => null }), {})
  assert.deepEqual(run({ getItem: () => 'dark' }), {})
})

test('localStorage que lança erro não propaga e não define atributo', () => {
  const attrs = run({
    getItem: () => {
      throw new Error('bloqueado')
    },
  })
  assert.deepEqual(attrs, {})
})
