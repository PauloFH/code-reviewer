import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readStorage, writeStorage, removeStorage } from '../src/brand/storage.ts'

function stub(impl: unknown) {
  Object.defineProperty(globalThis, 'localStorage', { value: impl, configurable: true, writable: true })
}

test('storage: fluxo normal lê, grava e remove', () => {
  const m = new Map<string, string>()
  stub({
    getItem: (k: string) => m.get(k) ?? null,
    setItem: (k: string, v: string) => void m.set(k, v),
    removeItem: (k: string) => void m.delete(k),
  })
  assert.equal(readStorage('jwt'), null)
  writeStorage('jwt', 'abc')
  assert.equal(readStorage('jwt'), 'abc')
  removeStorage('jwt')
  assert.equal(readStorage('jwt'), null)
})

test('storage: bloqueado não lança; leitura volta null', () => {
  const boom = () => {
    throw new DOMException('bloqueado', 'SecurityError')
  }
  stub({ getItem: boom, setItem: boom, removeItem: boom })
  assert.equal(readStorage('jwt'), null)
  assert.doesNotThrow(() => writeStorage('jwt', 'x'))
  assert.doesNotThrow(() => removeStorage('jwt'))
})
