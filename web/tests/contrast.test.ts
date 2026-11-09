import { test } from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const css = readFileSync(join(import.meta.dirname, '../src/assets/tokens.css'), 'utf8')

type RGBA = [number, number, number, number]

function block(selector: string): Record<string, string> {
  const start = css.indexOf(`${selector} {`)
  assert.ok(start >= 0, `bloco ${selector} não encontrado`)
  const body = css.slice(css.indexOf('{', start) + 1, css.indexOf('}', start))
  const vars: Record<string, string> = {}
  for (const m of body.matchAll(/(--[\w-]+)\s*:\s*([^;]+);/g)) vars[m[1]] = m[2].trim()
  return vars
}

function parse(value: string): RGBA {
  const hex = value.match(/^#([0-9a-f]{6})$/i)
  if (hex) {
    const n = parseInt(hex[1], 16)
    return [(n >> 16) & 255, (n >> 8) & 255, n & 255, 1]
  }
  const rgba = value.match(/^rgba?\(([^)]+)\)$/)
  assert.ok(rgba, `cor não reconhecida: ${value}`)
  const p = rgba[1].split(',').map((s) => parseFloat(s))
  return [p[0], p[1], p[2], p[3] ?? 1]
}

function over(fg: RGBA, bg: RGBA): RGBA {
  const a = fg[3]
  return [0, 1, 2].map((i) => fg[i] * a + bg[i] * (1 - a)).concat(1) as RGBA
}

function luminance([r, g, b]: RGBA): number {
  const [R, G, B] = [r, g, b].map((c) => {
    const s = c / 255
    return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
  })
  return 0.2126 * R + 0.7152 * G + 0.0722 * B
}

function ratio(a: RGBA, b: RGBA): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x)
  return (hi + 0.05) / (lo + 0.05)
}

const dark = block(':root')
const themes: Record<string, Record<string, string>> = {
  escuro: dark,
  claro: { ...dark, ...block('[data-theme="light"]') },
}

const fgs = ['--text', '--text-2', '--muted', '--muted-2', '--ok', '--danger', '--warn', '--accent']
const bgs = ['--bg', '--bg-2', '--panel', '--panel-solid', '--panel-2']

for (const [nome, vars] of Object.entries(themes)) {
  const get = (k: string): RGBA => {
    assert.ok(vars[k], `token ${k} ausente no tema ${nome}`)
    const ref = vars[k].match(/^var\((--[\w-]+)\)$/)
    return ref ? get(ref[1]) : parse(vars[k])
  }
  const base = get('--bg')

  test(`contraste AA no tema ${nome}: texto e semânticas sobre fundos`, () => {
    for (const bg of bgs) {
      const fundo = over(get(bg), base)
      for (const fg of fgs) {
        const r = ratio(over(get(fg), fundo), fundo)
        assert.ok(r >= 4.5, `${fg} sobre ${bg} no tema ${nome}: ${r.toFixed(2)} < 4.5`)
      }
    }
  })

  // Componentes de UI (WCAG 1.4.11): borda de campo e anel de foco precisam de 3:1.
  test(`contraste AA no tema ${nome}: borda de campo e foco sobre fundos`, () => {
    for (const bg of bgs) {
      const fundo = over(get(bg), base)
      for (const ui of ['--field-border', '--focus']) {
        const r = ratio(get(ui), fundo)
        assert.ok(r >= 3, `${ui} sobre ${bg} no tema ${nome}: ${r.toFixed(2)} < 3`)
      }
    }
  })

  // .pill.ok/.warn/.danger: texto na cor sobre a própria cor a 8% (color-mix) por cima do fundo.
  test(`contraste AA no tema ${nome}: pills de situação`, () => {
    for (const bg of bgs) {
      const fundo = over(get(bg), base)
      for (const fg of ['--ok', '--warn', '--danger']) {
        const cor = get(fg)
        const pill = over([cor[0], cor[1], cor[2], 0.08], fundo)
        const r = ratio(cor, pill)
        assert.ok(r >= 4.5, `pill ${fg} sobre ${bg} no tema ${nome}: ${r.toFixed(2)} < 4.5`)
      }
    }
  })

  test(`contraste AA no tema ${nome}: --ink sobre marca-texto`, () => {
    for (const hl of ['--hl-1', '--hl-2']) {
      const r = ratio(get('--ink'), get(hl))
      assert.ok(r >= 4.5, `--ink sobre ${hl} no tema ${nome}: ${r.toFixed(2)} < 4.5`)
    }
  })
}
