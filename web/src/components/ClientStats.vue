<template>
  <div class="wrap" v-if="stats.length > 0">
    <h3>By Client</h3>
    <table>
      <thead>
        <tr>
          <th>Client</th>
          <th>Reviews</th>
          <th>Tokens</th>
          <th>Cost</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="s in stats" :key="s.client_name">
          <td>{{ s.client_name }}</td>
          <td class="num">{{ s.review_count }}</td>
          <td class="num">{{ fmtK(s.total_tokens) }}</td>
          <td class="num">${{ s.total_cost_usd.toFixed(4) }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import type { ReviewStats } from '@/api/events'

const { stats } = defineProps<{ stats: ReviewStats['by_client'] }>()

function fmtK(n: number) {
  return n >= 1_000_000 ? (n / 1_000_000).toFixed(1) + 'M'
       : n >= 1000     ? (n / 1000).toFixed(1) + 'k'
       : String(n)
}
</script>

<style scoped>
.wrap { background: #1e293b; border: 1px solid #334155; border-radius: 10px; padding: 1.25rem; }
h3 { font-size: 1rem; font-weight: 600; color: #e2e8f0; margin-bottom: .75rem; }
table { width: 100%; border-collapse: collapse; font-size: .875rem; }
th { text-align: left; padding: .5rem .75rem; color: #64748b; font-weight: 600; border-bottom: 1px solid #334155; }
td { padding: .5rem .75rem; border-bottom: 1px solid #0f172a; color: #cbd5e1; }
tr:last-child td { border-bottom: none; }
.num { font-variant-numeric: tabular-nums; text-align: right; }
</style>
