import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const html = readFileSync(join(import.meta.dirname, '../index.html'), 'utf8')
const href = html.match(/href="(https:\/\/fonts\.googleapis\.com\/css2\?[^"]+)"/)?.[1]

test('URL do Google Fonts: cada tupla tem todos os eixos declarados', () => {
  assert.ok(href, 'link do Google Fonts não encontrado')
  const families = new URL(href.replaceAll('&amp;', '&')).searchParams.getAll('family')
  assert.ok(families.length >= 2)
  for (const fam of families) {
    const [nome, spec] = fam.split(':')
    if (!spec) continue
    const [eixos, tuplas] = spec.split('@')
    const n = eixos.split(',').length
    for (const tupla of tuplas.split(';')) {
      assert.equal(tupla.split(',').length, n, `${nome}: tupla "${tupla}" não tem os ${n} eixos (${eixos})`)
    }
  }
})
