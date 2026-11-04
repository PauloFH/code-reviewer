import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readStorage, writeStorage, removeStorage, readSessionFlag, writeSessionFlag } from '../src/brand/storage.ts'

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

test('flag de sessão: grava, lê e não lança com sessionStorage bloqueado', () => {
  const m = new Map<string, string>()
  Object.defineProperty(globalThis, 'sessionStorage', { configurable: true, writable: true, value: {
    getItem: (k: string) => m.get(k) ?? null, setItem: (k: string, v: string) => void m.set(k, v),
  } })
  assert.equal(readSessionFlag('abertura'), false)
  writeSessionFlag('abertura')
  assert.equal(readSessionFlag('abertura'), true)

  Object.defineProperty(globalThis, 'sessionStorage', { configurable: true, get() { throw new Error('bloqueado') } })
  assert.equal(readSessionFlag('abertura'), false)
  assert.doesNotThrow(() => writeSessionFlag('abertura'))
})
