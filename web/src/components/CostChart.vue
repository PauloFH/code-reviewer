<template>
  <div class="cost-chart">
    <span v-if="loading && points.length === 0" class="spinner" />
    <div v-else-if="points.length === 0" class="empty-chart">Sem dados ainda</div>
    <div v-else class="chart-box">
      <Line :data="chartData" :options="chartOptions" class="chart" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { Line } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Tooltip,
  Filler,
  type ScriptableContext,
} from 'chart.js'
import { getDailyCosts, type DailyCostPoint } from '@/api/events'
import { formatUsd } from '@/brand/format'
import { useTheme } from '@/composables/useTheme'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Filler)

const props = defineProps<{ client?: string; days?: number }>()
const { theme } = useTheme()

const points = ref<DailyCostPoint[]>([])
const loading = ref(false)

onMounted(load)
watch(() => props.client, load)
watch(() => props.days, load)

async function load() {
  loading.value = true
  try {
    points.value = await getDailyCosts(props.client, props.days || 30)
  } finally {
    loading.value = false
  }
}

// O canvas não entende var(--x): as cores são resolvidas aqui e refeitas quando o tema muda.
const colors = computed(() => {
  void theme.value
  const css = getComputedStyle(document.documentElement)
  const v = (name: string) => css.getPropertyValue(name).trim()
  return {
    danger: v('--danger'),
    mid: v('--merge-mid'),
    ok: v('--ok'),
    text: v('--text'),
    muted: v('--muted'),
    panel: v('--panel-solid'),
    border: v('--hairline-strong'),
    mono: v('--font-mono'),
  }
})

// Aceita só #rgb/#rrggbb; qualquer outro formato volta sem transparência.
function withAlpha(color: string, alpha: number) {
  const m = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i.exec(color)
  if (!m) return color
  const h = m[1].length === 3 ? m[1].replace(/./g, '$&$&') : m[1]
  const n = parseInt(h, 16)
  return `rgba(${n >> 16}, ${(n >> 8) & 255}, ${n & 255}, ${alpha})`
}

const MESES = ['jan', 'fev', 'mar', 'abr', 'mai', 'jun', 'jul', 'ago', 'set', 'out', 'nov', 'dez']

function formatDay(iso: string) {
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(iso)
  return m ? `${Number(m[3])} ${MESES[Number(m[2]) - 1]}` : iso
}

const chartData = computed(() => {
  const c = colors.value
  return {
    labels: points.value.map((p) => formatDay(p.date)),
    datasets: [
      {
        label: 'Custo (US$)',
        data: points.value.map((p) => p.cost_usd),
        borderColor: (ctx: ScriptableContext<'line'>) => {
          const area = ctx.chart.chartArea
          if (!area) return c.mid
          const g = ctx.chart.ctx.createLinearGradient(area.left, 0, area.right, 0)
          g.addColorStop(0, c.danger)
          g.addColorStop(0.5, c.mid)
          g.addColorStop(1, c.ok)
          return g
        },
        backgroundColor: (ctx: ScriptableContext<'line'>) => {
          const area = ctx.chart.chartArea
          if (!area) return withAlpha(c.mid, 0.2)
          const g = ctx.chart.ctx.createLinearGradient(0, area.top, 0, area.bottom)
          g.addColorStop(0, withAlpha(c.mid, 0.26))
          g.addColorStop(1, withAlpha(c.mid, 0))
          return g
        },
        borderWidth: 3,
        fill: true,
        tension: 0,
        // com um único ponto (24h) não há linha para desenhar; o ponto precisa aparecer
        pointRadius: points.value.length === 1 ? 4 : 0,
        pointBackgroundColor: c.mid,
        pointHoverRadius: 4,
        pointHoverBackgroundColor: c.mid,
        pointHoverBorderColor: c.text,
      },
    ],
  }
})

const reducedMotion = typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches

const chartOptions = computed(() => {
  const c = colors.value
  const font = { family: c.mono, size: 11 }
  return {
    responsive: true,
    maintainAspectRatio: false,
    layout: { padding: { top: 8, right: 8 } },
    animation: reducedMotion ? (false as const) : undefined,
    interaction: { mode: 'index' as const, intersect: false },
    plugins: {
      legend: { display: false },
      tooltip: {
        displayColors: false,
        cornerRadius: 2,
        backgroundColor: c.panel,
        borderColor: c.border,
        borderWidth: 1,
        titleColor: c.text,
        bodyColor: c.text,
        titleFont: font,
        bodyFont: font,
        callbacks: {
          label: (ctx: { raw: unknown }) => formatUsd(Number(ctx.raw)),
        },
      },
    },
    scales: {
      x: {
        offset: points.value.length === 1, // ponto único centralizado em vez de colado à borda
        ticks: { color: c.muted, maxTicksLimit: 6, maxRotation: 0, font },
        grid: { display: false },
        border: { display: false },
      },
      y: { beginAtZero: true, display: false },
    },
  }
})
</script>

<style scoped>
.cost-chart {
  min-height: 190px;
}

.chart-box {
  position: relative;
  height: 220px;
}

.chart {
  display: block;
  width: 100% !important;
  height: 100% !important;
}

.empty-chart {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 190px;
  color: var(--muted);
  font-size: 13px;
  font-family: var(--font-mono);
}
</style>
