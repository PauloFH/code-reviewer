<template>
  <div class="chart-wrap">
    <div class="chart-header">
      <h3>Daily Cost — Last 30 Days</h3>
      <span v-if="loading" class="spinner" />
    </div>
    <div v-if="!loading && points.length === 0" class="empty">No data yet</div>
    <div v-else-if="points.length > 0" class="chart-box">
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
  Title,
  Tooltip,
  Filler,
} from 'chart.js'
import { getDailyCosts, type DailyCostPoint } from '@/api/events'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Filler)

const props = defineProps<{ client?: string; days?: number }>()

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

const chartData = computed(() => ({
  labels: points.value.map((p) => p.date),
  datasets: [
    {
      label: 'Cost (USD)',
      data: points.value.map((p) => p.cost_usd),
      borderColor: 'var(--accent)',
      backgroundColor: 'rgba(129,140,248,0.12)',
      fill: true,
      tension: 0.4,
      pointRadius: 3,
      pointBackgroundColor: 'var(--accent)',
    },
  ],
}))

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (ctx: { raw: unknown }) => `$${Number(ctx.raw).toFixed(5)}`,
      },
    },
  },
  scales: {
    x: {
      ticks: { color: 'var(--muted)', maxTicksLimit: 10 },
      grid: { color: 'var(--hairline)' },
    },
    y: {
      ticks: {
        color: 'var(--muted)',
        callback: (v: unknown) => `$${Number(v).toFixed(4)}`,
      },
      grid: { color: 'var(--hairline)' },
    },
  },
}
</script>

<style scoped>
.chart-wrap {
  background: var(--panel);
  border: 1px solid var(--hairline);
  border-radius: var(--radius-lg);
  padding: 1.25rem;
}
.chart-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-bottom: 1rem;
}
h3 { font-size: 13.5px; font-weight: 500; color: var(--text); }
.chart-box {
  position: relative;
  height: 220px;
  min-height: 220px;
  max-height: 220px;
}
.chart {
  display: block;
  width: 100% !important;
  height: 100% !important;
}
.empty {
  text-align: center;
  color: var(--muted);
  padding: 3rem 0;
  font-size: 13px;
  font-family: var(--font-mono);
}
</style>
