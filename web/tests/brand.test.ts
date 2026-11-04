import { test } from 'node:test'
import assert from 'node:assert/strict'
import { statusMark } from '../src/brand/status.ts'
import { formatUsd, formatCount, formatRelative, formatRole } from '../src/brand/format.ts'
import { NAV_SHORTCUTS, isEditableTarget } from '../src/brand/shortcuts.ts'

test('statusMark: sinal, rótulo e tom por situação', () => {
  assert.deepEqual(statusMark('success'), { sign: '+', label: 'publicada', tone: 'ok' })
  assert.deepEqual(statusMark('error'), { sign: '−', label: 'erro', tone: 'danger' })
  assert.deepEqual(statusMark('skipped'), { sign: '~', label: 'ignorada', tone: 'warn' })
  assert.deepEqual(statusMark('novo'), { sign: '~', label: 'novo', tone: 'warn' })
})

test('formatUsd: vírgula decimal, 2 casas, até 5 abaixo de 0,01', () => {
  assert.equal(formatUsd(38.17), 'US$ 38,17')
  assert.equal(formatUsd(0), 'US$ 0,00')
  assert.equal(formatUsd(0.00042), 'US$ 0,00042')
  assert.equal(formatUsd(Number.NaN), 'US$ 0,00')
  assert.equal(formatUsd(Infinity), 'US$ 0,00')
})

test('formatCount: milhar, mil e mi', () => {
  assert.equal(formatCount(1284), '1.284')
  assert.equal(formatCount(12400), '12,4 mil')
  assert.equal(formatCount(4200000), '4,2 mi')
  assert.equal(formatCount(0), '0')
  assert.equal(formatCount(Number.NaN), '0')
})

test('formatRelative: agora, minutos, horas e dias', () => {
  const now = new Date('2026-11-04T12:00:00Z')
  assert.equal(formatRelative('2026-11-04T11:59:40Z', now), 'agora')
  assert.equal(formatRelative('2026-11-04T11:46:00Z', now), 'há 14 min')
  assert.equal(formatRelative('2026-11-04T09:00:00Z', now), 'há 3 h')
  assert.equal(formatRelative('2026-11-01T12:00:00Z', now), 'há 3 dias')
  assert.equal(formatRelative('2026-11-03T11:00:00Z', now), 'há 1 dia')
  assert.equal(formatRelative('2026-11-04T12:05:00Z', now), 'agora')
  assert.equal(formatRelative('lixo', now), '')
})

test('atalhos de navegação e alvos editáveis', () => {
  assert.deepEqual(NAV_SHORTCUTS, { p: '/dashboard', r: '/events', c: '/clients', o: '/prompts', m: '/ai-configs', f: '/review-profiles', u: '/users' })
  assert.equal(isEditableTarget({ tagName: 'INPUT' }), true)
  assert.equal(isEditableTarget({ tagName: 'SELECT' }), true)
  assert.equal(isEditableTarget({ tagName: 'TEXTAREA' }), true)
  assert.equal(isEditableTarget({ tagName: 'DIV', isContentEditable: true }), true)
  assert.equal(isEditableTarget({ tagName: 'DIV', isContentEditable: false }), false)
  assert.equal(isEditableTarget(null), false)
})

test('formatRole: só `padrao` ganha acento; os demais ficam como vêm', () => {
  assert.equal(formatRole('padrao'), 'padrão')
  assert.equal(formatRole('super'), 'super')
  assert.equal(formatRole('gestor'), 'gestor')
})
