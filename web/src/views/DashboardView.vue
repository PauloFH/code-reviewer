<template>
  <div class="page">
    <h2>Dashboard</h2>

    <div v-if="loadingStats" class="loading-row">
      <span class="spinner" /> Loading stats…
    </div>
    <template v-else-if="stats">
      <StatsCards :stats="stats" />
      <ClientStats :stats="stats.by_client" />
    </template>

    <CostChart />

    <LiveFeed />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import StatsCards from '@/components/StatsCards.vue'
import ClientStats from '@/components/ClientStats.vue'
import CostChart from '@/components/CostChart.vue'
import LiveFeed from '@/components/LiveFeed.vue'
import { getStats, type ReviewStats } from '@/api/events'

const stats = ref<ReviewStats | null>(null)
const loadingStats = ref(false)

onMounted(async () => {
  loadingStats.value = true
  try {
    stats.value = await getStats()
  } finally {
    loadingStats.value = false
  }
})
</script>

<style scoped>
.page { display: flex; flex-direction: column; gap: 1.5rem; }
h2 { font-size: 1.5rem; font-weight: 700; color: #f8fafc; }
.loading-row { display: flex; align-items: center; gap: .5rem; color: #64748b; }
.spinner {
  width: 14px; height: 14px;
  border: 2px solid #334155;
  border-top-color: #6366f1;
  border-radius: 50%;
  animation: spin .6s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>
