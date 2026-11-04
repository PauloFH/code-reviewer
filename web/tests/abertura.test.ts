import { test } from 'node:test'
import assert from 'node:assert/strict'
import { ABERTURA, TRILHO_CENTRO, quadroAbertura, caminhoRamo, deveTocarAbertura, type Medidas } from '../src/brand/abertura.ts'

const med = (largura: number, altura: number, topoLivre = 62): Medidas =>
  ({ largura, altura, topoLivre, logoTopo: { x: 22, y: 19, altura: 24 } })
const TAMANHOS = [[400, 700], [800, 360], [1000, 800], [1280, 800], [1600, 900], [2560, 1440]]

test('fim do fio amarelo cai no centro da linha, na borda do herói, com traço de 3', () => {
  for (const [w, h] of TAMANHOS) {
    const q = quadroAbertura(ABERTURA.FIM, med(w, h))
    assert.ok(Math.abs(q.tronco[8] - (w - TRILHO_CENTRO)) < 1e-6)
    assert.ok(Math.abs(q.tronco[9] - h) < 1e-6)
    assert.ok(Math.abs(q.riscoTronco - 3) < 1e-6)
    assert.ok(Math.abs(q.riscoRamos - 3) < 1e-6)
  }
})

test('ramos e tronco compartilham o ponto de junção em toda a abertura', () => {
  for (let t = 0; t <= ABERTURA.FIM; t += 0.1) {
    const q = quadroAbertura(t, med(1280, 800))
    for (const i of [0, 1]) {
      assert.ok(Math.abs(q.ramoV[18 + i] - q.tronco[i]) < 1e-6, `V em t=${t}`)
      assert.ok(Math.abs(q.ramoG[18 + i] - q.tronco[i]) < 1e-6, `G em t=${t}`)
    }
  }
})

test('fios mantêm a proporção aprovada em janelas altas', () => {
  for (const h of [700, 1400, 2200]) {
    const q = quadroAbertura(ABERTURA.FIM, med(1280, h))
    const larg = q.ramoV[18] - q.ramoV[0], alt = q.ramoG[1] - q.ramoV[1]
    assert.ok(Math.abs(alt / larg - 276 / 220) < 1e-6, `altura ${h}`)
  }
})

test('fios nunca sobem acima de topoLivre nem saem do herói', () => {
  for (const [w, h] of TAMANHOS) {
    const q = quadroAbertura(ABERTURA.FIM, med(w, h, w < 700 ? 300 : 62))
    assert.ok(q.ramoV[1] >= (w < 700 ? 300 : 62))
    assert.ok(q.ramoV[0] >= 0)
    assert.ok(q.ramoG[1] <= h)
  }
})

test('a logo se forma centrada no herói, antes de abrir', () => {
  const q = quadroAbertura(ABERTURA.ABRE, med(1280, 800))
  const centroX = q.resto.x + (866 / 2) * q.resto.escala
  const centroY = q.resto.y + (450 / 2) * q.resto.escala
  assert.ok(Math.abs(centroX - 640) < 1e-6)
  assert.ok(Math.abs(centroY - 400) < 1e-6)
})

test('estado inicial esconde a página e estado final a mostra inteira', () => {
  const a = quadroAbertura(0, med(1280, 800)), z = quadroAbertura(ABERTURA.FIM, med(1280, 800))
  assert.deepEqual([a.topo, a.texto, a.trilho, a.tracoRamos], [0, 0, 0, 0])
  assert.deepEqual([z.topo, z.texto, z.trilho, z.textoY, z.resto.opacidade, z.amarelo], [1, 1, 1, 0, 0, 1])
})

test('tempo além do fim equivale ao fim, como ao voltar de uma aba oculta', () => {
  assert.deepEqual(quadroAbertura(9999, med(1280, 800)), quadroAbertura(ABERTURA.FIM, med(1280, 800)))
})

test('tamanho zero não gera NaN', () => {
  const q = quadroAbertura(3.2, med(0, 0, 0))
  for (const n of [...q.ramoV, ...q.ramoG, ...q.tronco, q.riscoRamos, q.resto.escala]) assert.ok(Number.isFinite(n))
  assert.ok(!caminhoRamo(q.ramoV).includes('NaN'))
})

test('abertura só toca sem movimento reduzido, sem hash e na primeira vez da sessão', () => {
  const base = { reduzMovimento: false, jaTocou: false, temHash: false }
  assert.equal(deveTocarAbertura(base), true)
  assert.equal(deveTocarAbertura({ ...base, reduzMovimento: true }), false)
  assert.equal(deveTocarAbertura({ ...base, jaTocou: true }), false)
  assert.equal(deveTocarAbertura({ ...base, temHash: true }), false)
})
