<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useTheme } from '@/composables/useTheme'

const props = withDefaults(defineProps<{ intensity?: number }>(), { intensity: 1 })

const { theme } = useTheme()
const canvas = ref<HTMLCanvasElement | null>(null)

const GAP = 150
const SPEED = 16
const CYCLE = 6
const STATIC_FRAME_MS = 9000
// Fundo decorativo não precisa de 60 fps: a posição vem do tempo, então só cai a taxa de quadros.
const FRAME_MS = 1000 / 25

const rnd = (n: number) => {
  const x = Math.sin(n * 127.1) * 43758.5453
  return x - Math.floor(x)
}

let ctx: CanvasRenderingContext2D | null = null
let w = 0
let h = 0
let raf = 0
let reduced: MediaQueryList | null = null

function size() {
  const cv = canvas.value
  if (!cv || !ctx) return
  const r = devicePixelRatio || 1
  w = innerWidth
  h = innerHeight
  cv.width = w * r
  cv.height = h * r
  ctx.setTransform(r, 0, 0, r, 0, 0)
}

// Canvas não entende var(--x): as cores são lidas do CSS a cada quadro, para acompanhar o tema.
function draw(ms: number) {
  const c = ctx
  if (!c) return
  const t = ms / 1000
  const st = getComputedStyle(document.documentElement)
  const red = st.getPropertyValue('--danger').trim()
  const green = st.getPropertyValue('--ok').trim()
  const mid = st.getPropertyValue('--merge-mid').trim()
  const light = theme.value === 'light'
  const k = props.intensity
  const a = (v: number) => Math.min(1, v * k)

  c.clearRect(0, 0, w, h)
  const lanes = Math.ceil(w / GAP) + 1
  c.lineWidth = 2
  for (let i = 0; i < lanes; i++) {
    const x = i * GAP + 40
    const color = i % 2 ? green : red
    c.globalAlpha = a(light ? 0.13 : 0.1)
    c.strokeStyle = color
    c.beginPath()
    c.moveTo(x, 0)
    c.lineTo(x, h)
    c.stroke()
    c.globalAlpha = a(light ? 0.3 : 0.26)
    c.fillStyle = color
    for (let j = 0; j < 7; j++) {
      const y = h + 20 - ((t * SPEED * (0.8 + rnd(i) * 0.5) + rnd(i * 9 + j) * 900 + j * 170) % (h + 40))
      c.fillRect(x - 4, y - 4, 8, 8)
    }
  }
  // branch sendo puxada: desenha-se entre duas raias e leva um commit até o merge
  for (let i = 0; i < lanes - 1; i += 2) {
    const n = i + Math.floor((t + i) / CYCLE) * 11
    const p = ((t + i) % CYCLE) / CYCLE
    const pull = Math.min(1, p / 0.7)
    const e = 1 - Math.pow(1 - pull, 3)
    const fade = p > 0.85 ? (1 - p) / 0.15 : 1
    const x0 = i * GAP + 40
    const x1 = x0 + GAP
    const y0 = h * (0.2 + rnd(n) * 0.75)
    const y1 = y0 - 150
    const g = c.createLinearGradient(x0, y0, x1, y1)
    g.addColorStop(0, i % 4 ? green : red)
    g.addColorStop(1, mid)
    c.globalAlpha = a(light ? 0.34 : 0.3) * fade
    c.strokeStyle = g
    c.lineWidth = 2.5
    c.beginPath()
    const steps = 40
    for (let s = 0; s <= steps * e; s++) {
      const u = s / steps
      const q = 1 - u
      const px = q * q * q * x0 + 3 * q * q * u * x0 + 3 * q * u * u * x1 + u * u * u * x1
      const py = q * q * q * y0 + 3 * q * q * u * (y0 - 75) + 3 * q * u * u * (y0 - 75) + u * u * u * y1
      if (s) c.lineTo(px, py)
      else c.moveTo(px, py)
      if (s + 1 > steps * e) {
        c.stroke()
        c.globalAlpha = a(light ? 0.6 : 0.55) * fade
        c.fillStyle = mid
        c.fillRect(px - 5, py - 5, 10, 10)
      }
    }
  }
  c.globalAlpha = 1
}

let last = 0
function loop(ms: number) {
  const dt = ms - last
  if (dt >= FRAME_MS && !document.hidden) {
    draw(ms)
    // guarda a sobra para alternar 2 e 3 quadros de 60 Hz e manter a média em 25 fps
    last = ms - (dt % FRAME_MS)
  }
  raf = requestAnimationFrame(loop)
}

function onResize() {
  size()
  if (reduced?.matches) draw(STATIC_FRAME_MS)
}

// Com movimento reduzido há um único quadro estático; só o tema o invalida.
watch(theme, () => {
  if (reduced?.matches) draw(STATIC_FRAME_MS)
})

onMounted(() => {
  ctx = canvas.value?.getContext('2d') ?? null
  if (!ctx) return
  reduced = matchMedia('(prefers-reduced-motion: reduce)')
  size()
  addEventListener('resize', onResize)
  if (reduced.matches) draw(STATIC_FRAME_MS)
  else raf = requestAnimationFrame(loop)
})

onBeforeUnmount(() => {
  cancelAnimationFrame(raf)
  removeEventListener('resize', onResize)
})
</script>

<template>
  <canvas ref="canvas" class="bg-motion" aria-hidden="true"></canvas>
</template>

<style scoped>
.bg-motion {
  position: fixed;
  inset: 0;
  width: 100%;
  height: 100%;
  z-index: 0;
  pointer-events: none;
}
</style>
