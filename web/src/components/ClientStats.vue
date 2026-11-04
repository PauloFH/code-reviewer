<template>
  <div class="wrap">
    <div v-for="s in stats" :key="s.client_name" class="cli">
      <strong class="nome">{{ s.client_name }}</strong>
      <code class="parte">{{ formatShare(getShare(s.total_cost_usd)) }}</code>
      <small class="meta">
        {{ formatCount(s.review_count) }} {{ s.review_count === 1 ? 'revisão' : 'revisões' }} · {{ formatUsd(s.total_cost_usd) }}
      </small>
      <i
        class="barra"
        aria-hidden="true"
        :style="{ '--p': getShare(s.total_cost_usd) + '%', '--n': Math.max(getShare(s.total_cost_usd), 1) }"
      ></i>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ReviewStats } from '@/api/events'
import { formatCount, formatUsd } from '@/brand/format'

const props = defineProps<{ stats: ReviewStats['by_client'] }>()

const totalCost = computed(() => props.stats.reduce((acc, s) => acc + s.total_cost_usd, 0))

function getShare(cost: number) {
  return totalCost.value > 0 ? (cost / totalCost.value) * 100 : 0
}

function formatShare(v: number) {
  return v.toLocaleString('pt-BR', { minimumFractionDigits: 1, maximumFractionDigits: 1 }) + '%'
}
</script>

<style scoped>
.cli {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 2px 10px;
  align-items: baseline;
  padding: 13px 16px;
  border-bottom: 1px solid var(--hairline);
}

.cli:last-child {
  border-bottom: 0;
}

.nome {
  font-weight: 600;
  overflow-wrap: anywhere;
}

.parte {
  font-family: var(--font-mono);
  font-size: 13px;
}

.meta {
  grid-column: 1 / -1;
  font-family: var(--font-mono);
  font-size: 11.5px;
  color: var(--muted);
  overflow-wrap: anywhere;
}

.barra {
  display: block;
  grid-column: 1 / -1;
  height: 5px;
  margin-top: 6px;
  background: color-mix(in srgb, var(--text) 10%, transparent);
}

/* O gradiente é dimensionado pela trilha inteira, então a barra curta mostra só o começo do espectro. */
.barra::after {
  content: '';
  display: block;
  height: 100%;
  width: var(--p);
  background: var(--spectrum);
  background-size: calc(100% * 100 / var(--n)) 100%;
  background-repeat: no-repeat;
}
</style>
