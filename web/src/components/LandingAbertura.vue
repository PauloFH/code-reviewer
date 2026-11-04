<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import {
  ABERTURA,
  caminhoRamo,
  caminhoTronco,
  deveTocarAbertura,
  progresso,
  quadroAbertura,
} from '@/brand/abertura'
import { readSessionFlag, writeSessionFlag } from '@/brand/storage'

const props = defineProps<{
  raiz: HTMLElement
  heroi: HTMLElement
  topo: HTMLElement
  logoTopo: HTMLElement
  texto: HTMLElement
}>()

const cena = ref<SVGSVGElement>()
const ramoV = ref<SVGPathElement>()
const ramoG = ref<SVGPathElement>()
const tronco = ref<SVGPathElement>()
const resto = ref<SVGGElement>()

const VARIAVEIS = ['--abertura-topo', '--abertura-texto', '--abertura-texto-y', '--abertura-trilho']
const PULAM = ['pointerdown', 'wheel', 'touchstart', 'keydown', 'scroll']

let partes: SVGElement[] = []
let atual: number = ABERTURA.FIM
let inicio: number | null = null
let pedido = 0
let observador: ResizeObserver | undefined

function desenha(t: number) {
  atual = t
  const { raiz, heroi, topo, logoTopo, texto } = props
  const h = heroi.getBoundingClientRect()
  const { width: largura, height: altura } = h
  const l = logoTopo.getBoundingClientRect()
  const q = quadroAbertura(t, {
    largura,
    altura,
    topoLivre: largura >= 700 ? topo.offsetHeight : texto.offsetTop + texto.offsetHeight + 16,
    logoTopo: { x: l.left - h.left, y: l.top - h.top, altura: l.height },
  })

  cena.value!.setAttribute('viewBox', `0 0 ${largura} ${altura}`)
  for (const [el, v] of [[ramoV.value!, q.ramoV], [ramoG.value!, q.ramoG]] as const) {
    el.setAttribute('d', caminhoRamo(v))
    el.setAttribute('stroke-width', String(q.riscoRamos))
    el.setAttribute('stroke-dashoffset', String(1 - q.tracoRamos))
  }
  tronco.value!.setAttribute('d', caminhoTronco(q.tronco))
  tronco.value!.setAttribute('stroke-width', String(q.riscoTronco))
  tronco.value!.setAttribute('stroke-dashoffset', String(1 - q.tracoTronco))
  tronco.value!.style.setProperty('--amarelo', String(q.amarelo))

  const { x, y, escala, opacidade } = q.resto
  resto.value!.setAttribute('transform', `translate(${x} ${y}) scale(${escala})`)
  resto.value!.style.opacity = String(opacidade)
  for (const el of partes) {
    const p = progresso(t, Number(el.dataset.de), Number(el.dataset.ate))
    if (el.classList.contains('pop')) {
      el.style.transform = `scale(${p ? 0.4 + 0.6 * p : 0})`
      el.style.opacity = p ? '1' : '0'
    } else {
      el.setAttribute('stroke-dashoffset', String(1 - p))
    }
  }

  raiz.style.setProperty('--abertura-topo', String(q.topo))
  raiz.style.setProperty('--abertura-texto', String(q.texto))
  raiz.style.setProperty('--abertura-texto-y', `${q.textoY}px`)
  raiz.style.setProperty('--abertura-trilho', String(q.trilho))
}

function para() {
  cancelAnimationFrame(pedido)
  PULAM.forEach((e) => window.removeEventListener(e, pula))
}

function pula() {
  para()
  desenha(ABERTURA.FIM)
}

function passo(agora: number) {
  inicio ??= agora
  const t = (agora - inicio) / 1000
  if (t < ABERTURA.FIM) {
    desenha(t)
    pedido = requestAnimationFrame(passo)
  } else {
    pula()
  }
}

onMounted(() => {
  partes = [...resto.value!.querySelectorAll<SVGElement>('[data-de]')]
  // O texto muda de altura quando a fonte carrega sem o herói mudar, e abaixo de 700 px os fios dependem dele.
  observador = new ResizeObserver(() => desenha(atual))
  observador.observe(props.heroi)
  observador.observe(props.texto)

  const toca = deveTocarAbertura({
    reduzMovimento: matchMedia('(prefers-reduced-motion: reduce)').matches,
    jaTocou: readSessionFlag('abertura-tocou'),
    temHash: !!location.hash,
  })
  if (!toca) {
    desenha(ABERTURA.FIM)
    return
  }
  writeSessionFlag('abertura-tocou')
  desenha(0)
  PULAM.forEach((e) => window.addEventListener(e, pula, { passive: true }))
  pedido = requestAnimationFrame(passo)
})

onBeforeUnmount(() => {
  para()
  observador?.disconnect()
  VARIAVEIS.forEach((v) => props.raiz.style.removeProperty(v))
})
</script>

<template>
  <svg ref="cena" class="cena" aria-hidden="true">
    <defs>
      <linearGradient id="abertura-gradiente-v">
        <stop class="v-1" offset="0" />
        <stop class="v-2" offset=".5" />
        <stop class="meio" offset="1" />
      </linearGradient>
      <linearGradient id="abertura-gradiente-g">
        <stop class="g-1" offset="0" />
        <stop class="g-2" offset=".5" />
        <stop class="meio" offset="1" />
      </linearGradient>
    </defs>
    <path id="ramoV" ref="ramoV" stroke="url(#abertura-gradiente-v)" pathLength="1" stroke-dasharray="1 1" />
    <path id="ramoG" ref="ramoG" stroke="url(#abertura-gradiente-g)" pathLength="1" stroke-dasharray="1 1" />
    <path id="tronco" ref="tronco" stroke-linecap="round" pathLength="1" stroke-dasharray="1 1" />
    <!-- redesenho em SVG do símbolo oficial, nas coordenadas do PNG de 866x450 -->
    <g id="resto" ref="resto">
      <line class="claro" data-de="0" data-ate=".35" x1="22" y1="224" x2="90" y2="224" stroke-width="16" pathLength="1" stroke-dasharray="1 1" />
      <line class="claro" data-de=".5" data-ate="1.1" x1="150" y1="224" x2="322" y2="224" stroke-width="16" pathLength="1" stroke-dasharray="1 1" />
      <line class="claro" data-de="1.35" data-ate="1.7" x1="566" y1="224" x2="680" y2="224" stroke-width="16" pathLength="1" stroke-dasharray="1 1" />
      <circle class="pop claro fundo" data-de=".3" data-ate=".55" cx="117" cy="224" r="29" stroke-width="10" />
      <circle class="pop claro no-v" data-de=".8" data-ate="1.05" cx="336" cy="58" r="29" stroke-width="10" />
      <circle class="pop claro no-g" data-de=".8" data-ate="1.05" cx="336" cy="386" r="29" stroke-width="10" />
      <g class="brilho">
        <rect class="bloco" data-de="1.1" data-ate="1.6" x="351" y="136" width="184" height="180" rx="44" stroke-width="13" pathLength="1" stroke-dasharray="1 1" />
        <polyline class="claro" data-de="1.5" data-ate="1.7" points="418,181 391,204 418,227" stroke-width="10" pathLength="1" stroke-dasharray="1 1" />
        <line class="claro" data-de="1.6" data-ate="1.8" x1="456" y1="171" x2="432" y2="231" stroke-width="10" pathLength="1" stroke-dasharray="1 1" />
        <polyline class="claro" data-de="1.7" data-ate="1.9" points="471,181 498,204 471,227" stroke-width="10" pathLength="1" stroke-dasharray="1 1" />
        <polyline class="check" data-de="1.85" data-ate="2.15" points="414,263 433,285 472,252" stroke-width="10" pathLength="1" stroke-dasharray="1 1" />
      </g>
      <g class="pop" data-de="1.45" data-ate="1.75">
        <circle class="claro fundo" cx="721" cy="224" r="43" stroke-width="10" />
        <path class="claro" d="M696 258 L721 228 V198 M712 207 L721 196 L730 207 M721 228 H750 M741 219 L752 228 L741 237" stroke-width="6" />
      </g>
    </g>
  </svg>
</template>

<style scoped>
.cena {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  overflow: visible;
  pointer-events: none;
}

path,
polyline,
line,
rect {
  fill: none;
}

.v-1 { stop-color: var(--logo-v-1); }
.v-2 { stop-color: var(--logo-v-2); }
.g-1 { stop-color: var(--logo-g-1); }
.g-2 { stop-color: var(--logo-g-2); }
.meio { stop-color: var(--merge-mid); }

#tronco {
  stroke: var(--merge-mid);
  stroke: color-mix(in srgb, var(--merge-mid) calc(var(--amarelo) * 100%), var(--logo-traco));
}

.claro {
  stroke: var(--logo-traco);
  stroke-linecap: round;
  stroke-linejoin: round;
}

.check {
  stroke: var(--logo-check);
  stroke-linecap: round;
  stroke-linejoin: round;
}

.bloco {
  stroke: var(--logo-bloco);
}

.brilho {
  filter: drop-shadow(0 0 16px var(--logo-brilho));
}

.fundo { fill: var(--bg); }
.no-v { fill: var(--logo-no-v); }
.no-g { fill: var(--logo-no-g); }

.pop {
  transform-box: fill-box;
  transform-origin: center;
}
</style>
