<template>
  <div class="page">
    <div class="page-head">
      <div>
        <div class="page-eyebrow">visão geral · {{ today }}</div>
        <h1 class="page-title">Painel</h1>
        <div class="page-sub">Atividade de revisão automatizada em todos os clientes e projetos conectados.</div>
      </div>
      <div class="page-actions">
        <div class="seg" role="group" aria-label="Período">
          <button
            v-for="p in periods"
            :key="p"
            type="button"
            :class="{ on: period === p }"
            :aria-pressed="period === p"
            @click="setPeriod(p)"
          >
            {{ p }}
          </button>
        </div>
        <button class="btn" type="button" @click="loadStats">
          <AppIcon name="refresh" :size="13" />
          Atualizar
        </button>
      </div>
    </div>

    <div v-if="loadingStats" class="loading-row">
      <span class="spinner" /> Carregando indicadores…
    </div>
    <template v-else-if="stats">
      <StatsCards :stats="stats" />

      <div class="grade">
        <section class="card">
          <div class="card-head">
            <div>
              <h2 class="card-title">Custo diário</h2>
              <div class="card-sub">em dólar · {{ periodLabel }}</div>
            </div>
            <select v-model="filterClient" class="client-select" aria-label="Filtrar por cliente" @change="loadStats">
              <option value="">todos os clientes</option>
              <option v-for="c in clients" :key="c" :value="c">{{ c }}</option>
            </select>
          </div>
          <div class="card-body chart-wrap">
            <CostChart :client="filterClient || undefined" :days="periodDays" />
          </div>
        </section>

        <section class="card">
          <div class="card-head">
            <div>
              <h2 class="card-title">Por cliente</h2>
              <div class="card-sub">parte do custo total</div>
            </div>
          </div>
          <div class="card-body flush">
            <ClientStats :stats="stats.by_client" />
            <div v-if="!stats.by_client.length" class="vazio">
              Sem dados ainda.
              <router-link to="/clients">Adicionar cliente →</router-link>
            </div>
          </div>
        </section>
      </div>

      <section class="card">
        <div class="card-head">
          <div>
            <h2 class="card-title">Ao vivo</h2>
            <div class="card-sub">eventos recebidos em tempo real</div>
          </div>
          <span class="vivo" :class="{ off: !connected }">{{ connected ? 'conectado' : 'desconectado' }}</span>
        </div>
        <LiveFeed @connection="connected = $event" />
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import StatsCards from '@/components/StatsCards.vue'
import ClientStats from '@/components/ClientStats.vue'
import CostChart from '@/components/CostChart.vue'
import LiveFeed from '@/components/LiveFeed.vue'
import { getStats, type ReviewStats } from '@/api/events'
import { listClients } from '@/api/clients'
import AppIcon from '@/components/AppIcon.vue'

const stats = ref<ReviewStats | null>(null)
const loadingStats = ref(false)
const periods = ['24h', '7d', '30d', '90d']
const period = ref('7d')
const connected = ref(false)
const filterClient = ref('')
const clients = ref<string[]>([])

const periodDays = computed(() => {
  const map: Record<string, number> = { '24h': 1, '7d': 7, '30d': 30, '90d': 90 }
  return map[period.value] || 7
})

const periodLabel = computed(() =>
  periodDays.value === 1 ? 'último dia' : `últimos ${periodDays.value} dias`,
)

const today = new Date().toLocaleDateString('pt-BR', { day: 'numeric', month: 'long' })

function setPeriod(p: string) {
  period.value = p
  loadStats()
}

async function loadStats() {
  loadingStats.value = true
  try {
    stats.value = await getStats(filterClient.value || undefined)
  } finally {
    loadingStats.value = false
  }
}

async function loadClients() {
  try {
    const c = await listClients()
    clients.value = c.map((cl: { name: string }) => cl.name)
  } catch {
    // ignore
  }
}

onMounted(() => {
  loadStats()
  loadClients()
})
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.page-head {
  flex-wrap: wrap;
  padding-bottom: 0;
  margin-bottom: 2px;
  border-bottom: 0;
}

.page-eyebrow {
  font-family: var(--font-mono);
  font-size: 12.5px;
}

.page-title {
  letter-spacing: -0.04em;
  line-height: 1;
  margin-bottom: 8px;
}

.page-actions {
  flex-wrap: wrap;
}

.seg {
  padding: 0;
  background: var(--panel);
  border-radius: var(--radius);
}

.seg button {
  padding: 7px 11px;
  font-family: var(--font-mono);
  font-size: 12.5px;
  color: var(--text);
  border-radius: 0;
}

.seg button.on {
  background: var(--text);
  color: var(--bg);
  box-shadow: none;
}

.btn {
  padding: 8px 13px;
  font-size: 13.5px;
  border-radius: var(--radius);
}

.loading-row {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--muted);
  font-size: 13px;
  padding: 24px 0;
}

.grade {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(260px, 1fr);
  gap: 20px;
}

.card {
  border-radius: 0;
}

.card-head {
  flex-wrap: wrap;
  gap: 10px;
  padding: 14px 16px;
}

.card-title {
  margin: 0;
  font-size: 17px;
  font-weight: 800;
  letter-spacing: -0.02em;
}

.card-sub {
  font-family: var(--font-sans);
  font-size: 12.5px;
}

.chart-wrap {
  padding: 16px;
}

.vazio {
  padding: 18px 16px;
  font-size: 13px;
  color: var(--muted);
}

.vazio a {
  color: var(--accent);
  margin-left: 6px;
}

.vivo {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--ok);
}

.vivo::before {
  content: '';
  width: 8px;
  height: 8px;
  background: currentColor;
  animation: pulso 1.6s ease-in-out infinite;
}

.vivo.off {
  color: var(--muted);
}

.vivo.off::before {
  animation: none;
}

@keyframes pulso {
  50% {
    opacity: 0.3;
  }
}

.client-select {
  padding: 5px 8px;
  background: var(--bg);
  border: 1px solid var(--hairline);
  border-radius: var(--radius);
  color: var(--text);
  font-size: 13px;
  font-family: var(--font-sans);
  cursor: pointer;
  max-width: 100%;
}

.client-select:focus-visible {
  outline: 2px solid var(--merge-mid);
  outline-offset: 2px;
}

@media (max-width: 960px) {
  .grade {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (prefers-reduced-motion: reduce) {
  .vivo::before {
    animation: none;
  }
}
</style>
