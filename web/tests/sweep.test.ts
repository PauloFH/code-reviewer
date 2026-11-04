// Varredura final da identidade Mergeye: resíduos da marca antiga, cores soltas e textos em inglês.
import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readdirSync, readFileSync } from 'node:fs'
import { join, relative, sep } from 'node:path'

const SRC = new URL('../src/', import.meta.url).pathname

const BANIDOS = ['Geist', 'Code Reviewer', 'v 2.4', '818cf8', '6366f1', '4f46e5']
// Cor literal: #hex que não seja entidade HTML (&#8226;) e rgba( com número; rgba(${...}) montado de token passa.
const COR_HEX = /(?<![&\w])#[0-9a-fA-F]{3,8}\b/
const COR_RGBA = /rgba?\(\s*\d/
const RGBA_PRETO = /^rgba\(\s*0,\s*0,\s*0\s*[,)]/
// Textos do glossário que não podem aparecer como conteúdo de tag.
const INGLES = [
  'Sign in', 'Username', 'Password', 'Total Reviews', 'Reviews (30d)', 'Total Tokens', 'Total Cost',
  'Cost (30d)', 'all time', 'Live feed', 'live', 'Disconnected', 'Waiting for reviews…', 'Daily cost',
  'By client', 'share of total cost', 'Time', 'Client', 'Status', 'Cost', 'Duration', 'Cancel',
  'Watch Username', 'Loading stats…', 'No data yet', 'Dashboard', 'Events', 'Clients', 'Users',
  'Workspace', 'Overview',
]
const escapa = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
const TAG_INGLES = new RegExp(`>\\s*(${INGLES.map(escapa).join('|')})\\s*<`)

function achados(arquivo: string, texto: string): string[] {
  const out: string[] = []
  const ehVue = arquivo.endsWith('.vue')
  texto.split('\n').forEach((linha, i) => {
    const onde = `${arquivo}:${i + 1}`
    for (const b of BANIDOS) if (linha.includes(b)) out.push(`${onde} termo banido "${b}"`)
    // símbolo animado: AppIcon só desenha ícones estáticos de um traço
    const podeSvg = arquivo.endsWith(`${sep}AppIcon.vue`) || arquivo.endsWith(`${sep}LandingAbertura.vue`)
    if (linha.includes('<svg') && !podeSvg) out.push(`${onde} <svg fora de AppIcon.vue e LandingAbertura.vue`)
    if (ehVue) {
      if (COR_HEX.test(linha)) out.push(`${onde} cor hexadecimal`)
      for (const m of linha.matchAll(new RegExp(COR_RGBA, 'g'))) {
        if (!RGBA_PRETO.test(linha.slice(m.index))) out.push(`${onde} rgba( que não é preto`)
      }
      const m = TAG_INGLES.exec(linha)
      if (m) out.push(`${onde} texto em inglês ">${m[1]}<"`)
    }
  })
  return out
}

function arquivosDeSrc(): string[] {
  return readdirSync(SRC, { recursive: true, encoding: 'utf8' })
    .filter((f) => !f.split(sep).some((p) => p.startsWith('.')))
    .filter((f) => /\.(vue|ts|css)$/.test(f))
    .map((f) => join(SRC, f))
}

test('src está livre de resíduos da marca antiga, cores soltas e textos em inglês', () => {
  const todos = arquivosDeSrc().flatMap((f) => achados(relative(SRC, f), readFileSync(f, 'utf8')))
  assert.deepEqual(todos, [])
})

test('detectores pegam o que devem e deixam passar o legítimo', () => {
  assert.equal(achados('X.vue', '<b>Sign in</b>').length, 1)
  assert.equal(achados('X.vue', '<p>Total Reviews</p>').length, 1)
  assert.equal(achados('X.vue', 'color: #818cf8;').length, 2) // hex e termo banido
  assert.equal(achados('X.vue', 'color: #abc;').length, 1)
  assert.equal(achados('X.vue', 'background: rgba(10, 20, 30, .5)').length, 1)
  assert.equal(achados('X.ts', "font: 'Geist'").length, 1)
  assert.equal(achados('X.ts', '<svg></svg>').length, 1)
  assert.equal(achados('X.vue', '<svg></svg>').length, 1)
  assert.equal(achados(`a${sep}AppIcon.vue`, '<svg></svg>').length, 0)
  assert.equal(achados(`components${sep}LandingAbertura.vue`, '<svg></svg>').length, 0)
  assert.equal(achados('X.vue', 'box-shadow: 0 1px rgba(0, 0, 0, .3)').length, 0)
  assert.equal(achados('X.vue', '&#8226; Lista').length, 0)
  assert.equal(achados('X.vue', 'return `rgba(${r}, ${g}, ${b}, ${a})`').length, 0)
  assert.equal(achados('X.vue', '<span>Entrar</span>').length, 0)
  assert.equal(achados('tokens.css', '--ink: #0b0d12;').length, 0)
})
