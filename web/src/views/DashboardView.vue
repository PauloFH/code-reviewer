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
            v-for="p in PERIODS"
            :key="p.days"
            type="button"
            :class="{ on: days === p.days }"
            :aria-pressed="days === p.days"
            @click="days = p.days"
          >
            {{ p.label }}
          </button>
        </div>
        <button class="btn" type="button" @click="refresh">
          <AppIcon name="refresh" :size="13" />
          Atualizar
        </button>
      </div>
    </div>

    <div v-if="loadingStats && !stats" aria-busy="true">
      <span class="sr-only" role="status">Carregando indicadores…</span>
      <section class="kpis" aria-hidden="true">
        <div v-for="n in 5" :key="n" class="kpi">
          <span class="skeleton kpi-label-skeleton" />
          <span class="skeleton kpi-skeleton" />
        </div>
      </section>
      <div class="grade" aria-hidden="true">
        <section class="card"><div class="card-body chart-wrap"><span class="skeleton card-skeleton" /></div></section>
        <section class="card"><div class="card-body chart-wrap"><span class="skeleton card-skeleton" /></div></section>
      </div>
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
            <select v-if="clients.length" v-model="filterClient" class="client-select" aria-label="Filtrar por cliente" @change="loadStats">
              <option value="">todos os clientes</option>
              <option v-for="c in clients" :key="c.name" :value="c.name">{{ c.name }}</option>
            </select>
          </div>
          <div class="card-body chart-wrap">
            <CostChart :key="refreshes" :client="filterClient || undefined" :days="days" />
          </div>
        </section>

        <section class="card por-cliente">
          <div class="card-head">
            <div>
              <h2 class="card-title">Por cliente</h2>
              <div class="card-sub">parte do custo total</div>
            </div>
          </div>
          <div class="card-body">
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
            <div class="card-sub">eventos recebidos em tempo real{{ filterClient ? ` · ${filterClient}` : '' }}</div>
          </div>
          <span class="vivo" :class="{ off: !connected }">{{ connected ? 'conectado' : 'desconectado' }}</span>
        </div>
        <LiveFeed :key="`${refreshes}-${filterClient}`" :client="filterClient || undefined" @connection="connected = $event" />
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
import AppIcon from '@/components/AppIcon.vue'
import { getStats, type ReviewStats } from '@/api/events'
import { apiErrorMessage } from '@/api/http'
import { useToastStore } from '@/stores/toast'
import { useClients } from '@/composables/useClients'

// O período só muda o gráfico; os indicadores são sempre do total e dos últimos 30 dias.
const PERIODS = [
  { label: '24h', days: 1 },
  { label: '7d', days: 7 },
  { label: '30d', days: 30 },
  { label: '90d', days: 90 },
]

const toast = useToastStore()
const clients = useClients()
const stats = ref<ReviewStats | null>(null)
const loadingStats = ref(false)
const days = ref(7)
const connected = ref(false)
const filterClient = ref('')

const periodLabel = computed(() => (days.value === 1 ? 'último dia' : `últimos ${days.value} dias`))
const today = new Date().toLocaleDateString('pt-BR', { day: 'numeric', month: 'long' })

// "Atualizar" refaz indicadores e gráfico e reconecta o feed (o EventSource não volta sozinho após erro HTTP).
const refreshes = ref(0)
function refresh() {
  refreshes.value++
  loadStats()
}

async function loadStats() {
  loadingStats.value = true
  try {
    stats.value = await getStats(filterClient.value || undefined)
  } catch (e) {
    toast.push(apiErrorMessage(e, 'Erro ao carregar indicadores'), 'error')
  } finally {
    loadingStats.value = false
  }
}

onMounted(loadStats)
</script>

<style scoped>
.page {
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
  display: inline-flex;
  background: var(--panel);
  border: 1px solid var(--hairline);
  border-radius: var(--radius);
}

.seg button {
  background: transparent;
  border: 0;
  padding: 7px 11px;
  font: inherit;
  font-family: var(--font-mono);
  font-size: 12.5px;
  color: var(--text);
  cursor: pointer;
}

.seg button.on {
  background: var(--text);
  color: var(--bg);
}

.btn {
  padding: 8px 13px;
  font-size: 13.5px;
  border-radius: var(--radius);
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

.kpi-label-skeleton {
  width: 60%;
}

.kpi-skeleton {
  width: 45%;
  height: 28px;
  margin-top: 12px;
}

.card-skeleton {
  height: 220px;
}

.chart-wrap {
  padding: 16px;
}

.por-cliente {
  display: flex;
  flex-direction: column;
}

/* O corpo ocupa a altura do cartão (que acompanha o Custo diário), para centralizar o vazio na vertical. */
.por-cliente .card-body {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.vazio {
  flex: 1;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 6px 10px;
  padding: 48px 32px;
  font-size: 13px;
  font-family: var(--font-mono);
  color: var(--muted);
  text-align: center;
}

.vazio a {
  color: var(--accent);
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
  border: 1px solid var(--field-border);
  border-radius: var(--radius);
  color: var(--text);
  font-size: 13px;
  font-family: var(--font-sans);
  cursor: pointer;
  max-width: 100%;
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
