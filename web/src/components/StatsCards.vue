<template>
  <div class="kpis">
    <div class="kpi">
      <div class="kpi-label">Total Reviews</div>
      <div class="kpi-value mono">{{ fmt(stats.total_reviews) }}</div>
    </div>
    <div class="kpi">
      <div class="kpi-label">Last 30 days</div>
      <div class="kpi-value mono">{{ fmt(stats.reviews_last_30d) }}</div>
    </div>
    <div class="kpi">
      <div class="kpi-label">Total Tokens</div>
      <div class="kpi-value mono">{{ fmtK(stats.total_tokens) }}</div>
    </div>
    <div class="kpi">
      <div class="kpi-label">Total Cost</div>
      <div class="kpi-value mono">${{ stats.total_cost_usd.toFixed(5) }}</div>
    </div>
    <div class="kpi">
      <div class="kpi-label">Cost (30d)</div>
      <div class="kpi-value mono">${{ stats.cost_last_30d.toFixed(5) }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ReviewStats } from '@/api/events'

defineProps<{ stats: ReviewStats }>()

function fmt(n: number) {
  return new Intl.NumberFormat().format(n)
}

function fmtK(n: number) {
  return n >= 1_000_000 ? (n / 1_000_000).toFixed(1) + 'M' : n >= 1000 ? (n / 1000).toFixed(1) + 'k' : String(n)
}
</script>

<style scoped>
/* StatsCards uses global .kpis and .kpi classes from tokens.css */
</style>