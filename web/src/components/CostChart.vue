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

const props = defineProps<{ client?: string }>()

const points = ref<DailyCostPoint[]>([])
const loading = ref(false)

onMounted(load)
watch(() => props.client, load)

async function load() {
  loading.value = true
  try {
    points.value = await getDailyCosts(props.client, 30)
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
      borderColor: '#6366f1',
      backgroundColor: 'rgba(99,102,241,.15)',
      fill: true,
      tension: 0.4,
      pointRadius: 3,
      pointBackgroundColor: '#6366f1',
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
      ticks: { color: '#64748b', maxTicksLimit: 10 },
      grid: { color: '#1e293b' },
    },
    y: {
      ticks: {
        color: '#64748b',
        callback: (v: unknown) => `$${Number(v).toFixed(4)}`,
      },
      grid: { color: '#1e293b' },
    },
  },
}
</script>

<style scoped>
.chart-wrap {
  background: #1e293b;
  border: 1px solid #334155;
  border-radius: 10px;
  padding: 1.25rem;
}
.chart-header {
  display: flex;
  align-items: center;
  gap: .75rem;
  margin-bottom: 1rem;
}
h3 { font-size: 1rem; font-weight: 600; color: #e2e8f0; }
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
.empty { text-align: center; color: #475569; padding: 3rem 0; font-size: .875rem; }
.spinner {
  width: 14px; height: 14px;
  border: 2px solid #334155;
  border-top-color: #6366f1;
  border-radius: 50%;
  animation: spin .6s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>
