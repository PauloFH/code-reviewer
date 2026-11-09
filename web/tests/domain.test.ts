import { test } from 'node:test'
import assert from 'node:assert/strict'
import { NAV, ROLES, EVENT_STATUSES, roleCan, statusMark } from '../src/domain.ts'

test('NAV: atalhos e caminhos únicos, atalho é uma letra minúscula', () => {
  const keys = NAV.map((n) => n.key)
  assert.equal(new Set(keys).size, keys.length, `atalho repetido: ${keys}`)
  assert.equal(new Set(NAV.map((n) => n.path)).size, NAV.length)
  for (const k of keys) assert.match(k, /^[a-z]$/)
})

test('NAV: cada página restrita é alcançável por algum papel, e o super alcança todas', () => {
  for (const n of NAV) {
    assert.ok(ROLES.some((r) => roleCan(r.value, n.need)), `${n.path} inalcançável`)
    assert.ok(roleCan('super', n.need), `super sem acesso a ${n.path}`)
  }
})

test('papel desconhecido (usuário não carregado) não acessa página restrita', () => {
  assert.equal(roleCan(undefined, 'canManageResources'), false)
  assert.equal(roleCan('admin', 'canManageUsers'), false)
  assert.equal(roleCan(undefined, undefined), true)
})

test('situação: cada uma tem sinal e rótulo próprios; desconhecida mostra o valor cru', () => {
  const marks = EVENT_STATUSES.map((s) => statusMark(s.value))
  assert.equal(new Set(marks.map((m) => m.sign)).size, marks.length)
  assert.equal(new Set(marks.map((m) => m.label)).size, marks.length)
  for (const s of EVENT_STATUSES) assert.notEqual(statusMark(s.value).label, s.value)
  assert.equal(statusMark('novo').label, 'novo')
})
