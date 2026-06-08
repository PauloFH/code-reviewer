<template>
  <div class="page">
    <!-- Page head -->
    <div class="page-head">
      <div>
        <div class="page-eyebrow">Overview · {{ today }}</div>
        <h1 class="page-title">Dashboard</h1>
        <div class="page-sub">Atividade de revisão automatizada — todos os clientes e projetos conectados.</div>
      </div>
      <div class="page-actions">
        <div class="seg">
          <button :class="{ on: period === '24h' }" @click="setPeriod('24h')">24h</button>
          <button :class="{ on: period === '7d' }" @click="setPeriod('7d')">7d</button>
          <button :class="{ on: period === '30d' }" @click="setPeriod('30d')">30d</button>
          <button :class="{ on: period === '90d' }" @click="setPeriod('90d')">90d</button>
        </div>
        <button class="btn" @click="loadStats">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M3 12a9 9 0 0 1 15.5-6.3L21 8"/>
            <path d="M21 3v5h-5"/>
            <path d="M21 12a9 9 0 0 1-15.5 6.3L3 16"/>
            <path d="M3 21v-5h5"/>
          </svg>
          Atualizar
        </button>
      </div>
    </div>

    <!-- KPI Row -->
    <div v-if="loadingStats" class="loading-row">
      <span class="spinner" /> Loading stats…
    </div>
    <template v-else-if="stats">
      <StatsCards :stats="stats" />

      <!-- Charts grid -->
      <div class="grid-3-1">
        <div class="card">
          <div class="card-head">
            <div>
              <div class="card-title">Daily cost</div>
              <div class="card-sub">USD · últimos {{ period === '24h' ? '1 dia' : period === '7d' ? '7 dias' : period === '30d' ? '30 dias' : '90 dias' }}</div>
            </div>
            <div style="display: flex; align-items: center; gap: 12px;">
              <select v-model="filterClient" class="client-select" @change="loadStats">
                <option value="">todos os clientes</option>
                <option v-for="c in clients" :key="c" :value="c">{{ c }}</option>
              </select>
              <span class="card-sub" style="display: flex; align-items: center; gap: 8px;">
                <span class="legend-dot"></span>cost
              </span>
            </div>
          </div>
          <div class="card-body chart-wrap">
            <CostChart :client="filterClient || undefined" :days="periodDays" />
          </div>
        </div>

        <div class="card">
          <div class="card-head">
            <div>
              <div class="card-title">By client</div>
              <div class="card-sub">share of total cost</div>
            </div>
          </div>
          <div class="card-body flush" style="padding: 0;">
            <ClientStats :stats="stats.by_client" />
            <div v-if="!stats.by_client.length" style="padding: 14px 22px; font-size: 12px; color: var(--muted);">
              Apenas 1 cliente conectado. <router-link to="/clients" style="color: var(--accent);">Adicionar cliente →</router-link>
            </div>
          </div>
        </div>
      </div>

      <!-- Live Feed -->
      <div class="card" style="margin-top: 32px;">
        <div class="card-head">
          <div>
            <div class="card-title">Live feed</div>
            <div class="card-sub">eventos recebidos em tempo real</div>
          </div>
          <div style="display: flex; gap: 10px; align-items: center;">
            <span class="pill ok"><span class="dot pulse"></span>live</span>
          </div>
        </div>
        <LiveFeed />
      </div>
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

const stats = ref<ReviewStats | null>(null)
const loadingStats = ref(false)
const period = ref('7d')
const filterClient = ref('')
const clients = ref<string[]>([])

const periodDays = computed(() => {
  const map: Record<string, number> = { '24h': 1, '7d': 7, '30d': 30, '90d': 90 }
  return map[period.value] || 7
})

const today = computed(() => {
  return new Date().toLocaleDateString('en-US', { day: '2-digit', month: 'short', year: 'numeric' }).toUpperCase()
})

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
}

.loading-row {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--muted);
  font-size: 13px;
  padding: 24px 0;
}

.chart-wrap {
  position: relative;
  padding: 8px 14px 4px;
}

.legend-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--accent);
  display: inline-block;
  margin-right: 5px;
  vertical-align: middle;
}

.client-select {
  padding: 4px 8px;
  background: var(--bg-2);
  border: 1px solid var(--hairline);
  border-radius: var(--radius);
  color: var(--text-2);
  font-size: 12px;
  font-family: var(--font-sans);
  cursor: pointer;
}

.client-select:focus {
  outline: none;
  border-color: var(--accent);
}
</style>