import { test } from 'node:test'
import assert from 'node:assert/strict'
import { clsFromShifts, inpFromEvents } from '../src/brand/vitals.ts'

const shift = (startTime: number, value: number, hadRecentInput = false) => ({ startTime, value, hadRecentInput })

test('CLS: soma deslocamentos próximos numa mesma janela', () => {
  assert.ok(Math.abs(clsFromShifts([shift(0, 0.1), shift(500, 0.05)]) - 0.15) < 1e-9)
})

test('CLS: intervalo de 1 s abre nova janela e vale a maior', () => {
  assert.ok(Math.abs(clsFromShifts([shift(0, 0.1), shift(2000, 0.02), shift(2500, 0.03)]) - 0.1) < 1e-9)
})

test('CLS: janela não passa de 5 s mesmo com deslocamentos seguidos', () => {
  const seguidos = Array.from({ length: 12 }, (_, i) => shift(i * 900, 0.01))
  // 0, 900, ..., 4500 cabem na primeira janela (6 entradas); 5400 abre outra
  assert.ok(Math.abs(clsFromShifts(seguidos) - 0.06) < 1e-9)
})

test('CLS: ignora deslocamento após input e lista vazia vale 0', () => {
  assert.equal(clsFromShifts([shift(0, 0.5, true)]), 0)
  assert.equal(clsFromShifts([]), 0)
})

test('INP: pior interação, ignorando eventos sem interactionId', () => {
  assert.equal(inpFromEvents([{ interactionId: 1, duration: 40 }, { interactionId: 2, duration: 120 }, { interactionId: 0, duration: 900 }]), 120)
  assert.equal(inpFromEvents([]), 0)
})
