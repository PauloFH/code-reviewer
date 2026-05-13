<template>
  <div class="stats-grid">
    <div class="card">
      <div class="label">Total Reviews</div>
      <div class="value">{{ fmt(stats.total_reviews) }}</div>
    </div>
    <div class="card">
      <div class="label">Last 30 days</div>
      <div class="value">{{ fmt(stats.reviews_last_30d) }}</div>
    </div>
    <div class="card">
      <div class="label">Total Tokens</div>
      <div class="value">{{ fmtK(stats.total_tokens) }}</div>
    </div>
    <div class="card">
      <div class="label">Total Cost</div>
      <div class="value">${{ stats.total_cost_usd.toFixed(4) }}</div>
    </div>
    <div class="card highlight">
      <div class="label">Cost (30d)</div>
      <div class="value">${{ stats.cost_last_30d.toFixed(4) }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ReviewStats } from '@/api/events'

defineProps<{ stats: ReviewStats }>()

function fmt(n: number) { return new Intl.NumberFormat().format(n) }
function fmtK(n: number) { return n >= 1_000_000 ? (n/1_000_000).toFixed(1)+'M' : n >= 1000 ? (n/1000).toFixed(1)+'k' : String(n) }
</script>

<style scoped>
.stats-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 1rem; }
.card { background: #1e293b; border: 1px solid #334155; border-radius: 10px; padding: 1.2rem; }
.card.highlight { border-color: #4f46e5; }
.label { font-size: .72rem; color: #64748b; margin-bottom: .5rem; text-transform: uppercase; letter-spacing: .05em; }
.value { font-size: 1.5rem; font-weight: 700; color: #f1f5f9; }
</style>
