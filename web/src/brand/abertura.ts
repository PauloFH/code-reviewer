// Geometria e tempo da abertura da landing: função pura de (t, medidas) para o quadro a desenhar.
export const ABERTURA = { FIM: 4.2, ABRE: 2.7, DUR: 1 } as const
export const TRILHO_CENTRO = 31.5

export interface Medidas {
  largura: number
  altura: number
  topoLivre: number // y a partir do qual os fios podem ser desenhados
  logoTopo: { x: number; y: number; altura: number } // relativo ao herói
}

export interface Quadro {
  ramoV: number[] // M + 3 cúbicas: 20 números
  ramoG: number[] // idem
  tronco: number[] // M + 1 cúbica + L: 10 números
  riscoRamos: number
  riscoTronco: number
  amarelo: number // 0..1: quanto o tronco já é amarelo
  tracoRamos: number // 0..1: quanto dos ramos já foi desenhado
  tracoTronco: number // 0..1
  resto: { x: number; y: number; escala: number; opacidade: number }
  topo: number // opacidade do topo
  texto: number // opacidade do texto
  textoY: number // deslocamento em px
  trilho: number // 0..1: revelação da linha das seções
}

// Ramos e tronco da logo (coordenadas do PNG 866x450) nascem no mesmo ponto (721,224),
// sob o nó de merge, para nunca abrir vão ao se abrirem.
const LOGO_V = [135, 215, 190, 195, 240, 110, 336, 58, 400, 25, 500, 15, 585, 75, 640, 115, 675, 195, 721, 224]
const LOGO_G = [135, 235, 190, 255, 240, 340, 336, 386, 400, 425, 510, 430, 590, 370, 640, 330, 675, 255, 721, 224]
const LOGO_T = [721, 224, 760, 224, 810, 224, 845, 224, 845, 224]

export const progresso = (t: number, de: number, ate: number) => Math.min(1, Math.max(0, (t - de) / (ate - de)))
const suave = (x: number) => (x < 0.5 ? 4 * x * x * x : 1 - Math.pow(-2 * x + 2, 3) / 2)
const mix = (a: number, b: number, k: number) => a + (b - a) * k

// de Casteljau: divide a cúbica em t, para o fio final ter o mesmo número de pontos do ramo
function divide(p: number[], t: number): [number[], number[]] {
  const m = (i: number, j: number) => [mix(p[i], p[j], t), mix(p[i + 1], p[j + 1], t)]
  const a = m(0, 2), b = m(2, 4), c = m(4, 6)
  const ab = [mix(a[0], b[0], t), mix(a[1], b[1], t)], bc = [mix(b[0], c[0], t), mix(b[1], c[1], t)]
  const q = [mix(ab[0], bc[0], t), mix(ab[1], bc[1], t)]
  return [[p[0], p[1], ...a, ...ab, ...q], [...q, ...bc, ...c, p[6], p[7]]]
}

function emTres(p: number[]): number[] {
  const [um, r] = divide(p, 1 / 3), [dois, tres] = divide(r, 0.5)
  return [...um, ...dois.slice(2), ...tres.slice(2)]
}

export function quadroAbertura(tempo: number, m: Medidas): Quadro {
  const { FIM, ABRE, DUR } = ABERTURA
  const t = Math.min(tempo, FIM)
  const { largura: W, altura: H, topoLivre: T, logoTopo } = m

  const estreita = W < 700
  const s = Math.min((W * (estreita ? 0.86 : 0.6)) / 866, (H * 0.62) / 450)
  const lx = (W - 866 * s) / 2, ly = (H - 450 * s) / 2
  const px = (v: number[]) => v.map((n, i) => (i % 2 ? ly + n * s : lx + n * s))

  // fio final preso à direita; railX é o centro da linha do trilho
  const livre = Math.max(0, H - T)
  const e = Math.min((W * (estreita ? 0.72 : 0.4)) / 340, livre / 330)
  const oy = T + (livre - 330 * e) / 2
  const railX = W - TRILHO_CENTRO, x0 = railX - 308.5 * e, mx = x0 + 220 * e
  const fimV = emTres([x0, oy + 24 * e, x0 + 120 * e, oy + 24 * e, x0 + 130 * e, oy + 140 * e, mx, oy + 160 * e])
  const fimG = emTres([x0, oy + 300 * e, x0 + 120 * e, oy + 300 * e, x0 + 130 * e, oy + 180 * e, mx, oy + 160 * e])
  const fimT = [mx, oy + 160 * e, mx + 60 * e, oy + 160 * e, railX, oy + 180 * e, railX, oy + 230 * e, railX, H]

  const k = suave(progresso(t, ABRE, ABRE + DUR))
  const entre = (a: number[], b: number[]) => a.map((n, i) => mix(n, b[i], k))

  const texto = progresso(t, ABRE + 0.55, ABRE + 1.15)
  return {
    ramoV: entre(px(LOGO_V), fimV),
    ramoG: entre(px(LOGO_G), fimG),
    tronco: entre(px(LOGO_T), fimT),
    riscoRamos: mix(36 * s, 3, k),
    riscoTronco: mix(16 * s, 3, k),
    amarelo: Math.min(1, k * 2.5), // fica amarelo logo no começo, para não parecer outro fio
    tracoRamos: progresso(t, 0.5, 1.5),
    tracoTronco: progresso(t, 1.65, 1.95),
    // o resto do símbolo encolhe até a logo do topo, que aparece no lugar dele
    resto: {
      x: mix(lx, logoTopo.x, k),
      y: mix(ly, logoTopo.y, k),
      escala: mix(s, logoTopo.altura / 450, k),
      opacidade: 1 - progresso(t, ABRE + 0.35, ABRE + 0.95),
    },
    topo: progresso(t, ABRE + 0.45, ABRE + 1),
    texto,
    textoY: (1 - suave(texto)) * 14,
    trilho: progresso(t, ABRE + DUR - 0.05, ABRE + DUR + 0.45),
  }
}

export const caminhoRamo = (v: number[]) => `M${v[0]} ${v[1]}C${v.slice(2, 8)}C${v.slice(8, 14)}C${v.slice(14, 20)}`
export const caminhoTronco = (v: number[]) => `M${v[0]} ${v[1]}C${v.slice(2, 8)}L${v[8]} ${v[9]}`

export function deveTocarAbertura(o: { reduzMovimento: boolean; jaTocou: boolean; temHash: boolean }): boolean {
  return !o.reduzMovimento && !o.jaTocou && !o.temHash
}
