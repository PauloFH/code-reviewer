import { test } from 'node:test'
import assert from 'node:assert/strict'
import { csvField, toCsv } from '../src/brand/csv.ts'

test('csv: campo com vírgula, aspas ou quebra de linha vai entre aspas', () => {
  assert.equal(csvField('simples'), 'simples')
  assert.equal(csvField('a, b'), '"a, b"')
  assert.equal(csvField('diz "oi"'), '"diz ""oi"""')
  assert.equal(csvField('linha\nnova'), '"linha\nnova"')
  assert.equal(csvField(42), '42')
})

test('csv: texto que viraria fórmula na planilha ganha apóstrofo; número negativo não', () => {
  assert.equal(csvField('=HYPERLINK("x")'), `"'=HYPERLINK(""x"")"`)
  assert.equal(csvField('+1'), "'+1")
  assert.equal(csvField('@sum'), "'@sum")
  assert.equal(csvField(-3), '-3')
})

test('csv: linhas separadas por CRLF, uma coluna por campo', () => {
  assert.equal(toCsv([['a', 'b'], ['x, y', 1]]), 'a,b\r\n"x, y",1')
})
