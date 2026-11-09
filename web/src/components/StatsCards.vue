<template>
  <section class="kpis" aria-label="Indicadores">
    <div class="kpi">
      <div class="kpi-label">Revisões no total</div>
      <div class="kpi-value">{{ formatCount(stats.total_reviews) }}</div>
      <div class="kpi-foot">desde o início</div>
    </div>
    <div class="kpi">
      <div class="kpi-label">Revisões em 30 dias</div>
      <div class="kpi-value">{{ formatCount(stats.reviews_last_30d) }}</div>
    </div>
    <div class="kpi">
      <div class="kpi-label">Tokens no total</div>
      <div class="kpi-value">{{ formatCount(stats.total_tokens) }}</div>
    </div>
    <div class="kpi">
      <div class="kpi-label">Custo total</div>
      <div class="kpi-value">{{ formatUsd(stats.total_cost_usd) }}</div>
    </div>
    <div class="kpi">
      <div class="kpi-label">Custo em 30 dias</div>
      <div class="kpi-value">{{ formatUsd(stats.cost_last_30d) }}</div>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { ReviewStats } from '@/api/events'
import { formatCount, formatUsd } from '@/brand/format'

defineProps<{ stats: ReviewStats }>()
</script>

<style scoped>
.kpi-label {
  font-size: 13px;
  color: var(--muted);
}

.kpis {
  border-radius: 0;
}

.kpi {
  padding: 16px 16px 19px;
}

.kpi-value {
  margin-top: 8px;
  color: var(--text);
  font-family: var(--font-sans);
  font-size: clamp(22px, 7vw, 32px);
  white-space: nowrap;
  font-weight: 800;
  line-height: 1.15;
  letter-spacing: -0.03em;
  font-variant-numeric: tabular-nums;
}

.kpi-foot {
  margin-top: 4px;
  font-size: 11.5px;
  color: var(--muted);
  font-family: var(--font-mono);
}
</style>
