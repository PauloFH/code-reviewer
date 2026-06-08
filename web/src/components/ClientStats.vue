<template>
  <div class="wrap">
    <div v-for="s in stats" :key="s.client_name" class="stat-row">
      <div class="stat-info">
        <div class="stat-name">{{ s.client_name }}</div>
        <div class="stat-meta">{{ s.review_count }} reviews · ${{ s.total_cost_usd.toFixed(4) }}</div>
      </div>
      <div class="stat-bar">
        <div class="stat-fill" :style="{ width: getShare(s.total_cost_usd) + '%' }"></div>
      </div>
      <div class="stat-share mono">{{ getShare(s.total_cost_usd).toFixed(1) }}%</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ReviewStats } from '@/api/events'

const props = defineProps<{ stats: ReviewStats['by_client'] }>()

const totalCost = computed(() => {
  return props.stats.reduce((acc, s) => acc + s.total_cost_usd, 0)
})

function getShare(cost: number) {
  if (totalCost.value === 0) return 0
  return (cost / totalCost.value) * 100
}
</script>

<style scoped>
.wrap {
  display: flex;
  flex-direction: column;
}

.stat-row {
  display: grid;
  grid-template-columns: 1fr auto auto;
  gap: 16px;
  align-items: center;
  padding: 14px 22px;
  border-bottom: 1px solid var(--hairline);
}

.stat-row:last-child {
  border-bottom: 0;
}

.stat-row:hover {
  background: var(--panel-2);
}

.stat-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--text);
}

.stat-meta {
  font-size: 11px;
  color: var(--muted);
  font-family: var(--font-mono);
  margin-top: 2px;
}

.stat-bar {
  width: 80px;
  height: 4px;
  background: var(--hairline);
  border-radius: 2px;
  overflow: hidden;
}

.stat-fill {
  height: 100%;
  background: var(--accent);
  border-radius: 2px;
  transition: width 0.3s ease;
}

.stat-share {
  font-size: 11px;
  color: var(--muted);
  min-width: 40px;
  text-align: right;
}
</style>