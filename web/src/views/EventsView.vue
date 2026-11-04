<template>
  <div class="page">
    <!-- Page head -->
    <div class="page-head">
      <div>
        <div class="page-eyebrow">histórico</div>
        <h1 class="page-title">Revisões</h1>
        <div class="page-sub">Cada linha é uma MR/PR processada pelo agente. Use os filtros para refinar a lista.</div>
      </div>
      <div class="page-actions">
        <button class="btn" type="button" @click="exportCSV">
          <AppIcon name="document" :size="13" />
          Exportar CSV
        </button>
      </div>
    </div>

    <!-- Filterbar -->
    <div class="filterbar">
      <div class="field">
        <AppIcon name="search" :size="13" />
        <input v-model="filterSearch" aria-label="Buscar" placeholder="Buscar por título, autor ou hash…" @keyup.enter="applyFilter" />
      </div>
      <div class="field">
        <label for="filtro-cliente">Cliente</label>
        <select id="filtro-cliente" v-model="filterClient">
          <option value="">todos</option>
          <option v-for="c in clients" :key="c" :value="c">{{ c }}</option>
        </select>
      </div>
      <div class="field">
        <label for="filtro-situacao">Situação</label>
        <select id="filtro-situacao" v-model="filterStatus">
          <option value="">todas</option>
          <option v-for="s in statuses" :key="s" :value="s">{{ statusMark(s).label }}</option>
        </select>
      </div>
      <div class="field">
        <AppIcon name="calendar" :size="13" />
        <input v-model="filterSince" type="date" aria-label="Data inicial" @change="applyFilter" />
        <span class="seta" aria-hidden="true">→</span>
        <input v-model="filterUntil" type="date" aria-label="Data final" @change="applyFilter" />
      </div>
      <button class="btn primary" type="button" @click="applyFilter">Aplicar</button>
      <span class="counter">{{ formatCount(total) }} resultados · {{ formatCount(totalTokens) }} tokens · {{ formatUsd(totalCost) }}</span>
    </div>

    <!-- Table -->
    <div class="card">
      <EventsTable :events="events" :loading="loading" />
    </div>

    <!-- Pagination -->
    <div class="pagination">
      <span>Página {{ page + 1 }} de {{ totalPages }} · {{ formatCount(total) }} resultados</span>
      <div class="pagination-controls">
        <button class="btn sm" type="button" :disabled="page === 0" @click="prevPage">
          <AppIcon name="chevron-left" :size="12" />
          Anterior
        </button>
        <button class="btn sm" type="button" :disabled="page >= totalPages - 1" @click="nextPage">
          Próxima
          <AppIcon name="chevron-right" :size="12" />
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import EventsTable from '@/components/EventsTable.vue'
import AppIcon from '@/components/AppIcon.vue'
import { statusMark } from '@/brand/status'
import { formatCount, formatUsd } from '@/brand/format'
import { listEvents, type ReviewEvent } from '@/api/events'
import { listClients } from '@/api/clients'

const events = ref<ReviewEvent[]>([])
const clients = ref<string[]>([])
const total = ref(0)
const page = ref(0)
const loading = ref(false)
const pageSize = 25
const statuses = ['success', 'skipped', 'error']

const filterSearch = ref('')
const filterClient = ref('')
const filterStatus = ref('')
const filterSince = ref('')
const filterUntil = ref('')

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

const totalTokens = computed(() => {
  return events.value.reduce((acc, e) => acc + (e.prompt_tokens || 0) + (e.completion_tokens || 0), 0)
})

const totalCost = computed(() => {
  return events.value.reduce((acc, e) => acc + (e.cost_usd || 0), 0)
})

async function load() {
  loading.value = true
  try {
    const res = await listEvents({
      client: filterClient.value || undefined,
      search: filterSearch.value || undefined,
      since: filterSince.value ? filterSince.value + 'T00:00:00Z' : undefined,
      until: filterUntil.value ? filterUntil.value + 'T23:59:59Z' : undefined,
      limit: pageSize,
      offset: page.value * pageSize,
    })
    events.value = res.items ?? []
    total.value = res.total
  } finally {
    loading.value = false
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

function applyFilter() {
  page.value = 0
  load()
}

function prevPage() {
  page.value--
  load()
}

function nextPage() {
  page.value++
  load()
}

function exportCSV() {
  const headers = ['Time', 'Client', 'Project', 'MR IID', 'MR Title', 'Author', 'Status', 'Input Tokens', 'Output Tokens', 'Total Tokens', 'Cost (USD)', 'Duration (ms)', 'Reviewed At']
  const rows = events.value.map(e => [
    e.reviewed_at,
    e.client_name,
    e.project_ref,
    e.mr_iid,
    `"${(e.mr_title || '').replace(/"/g, '""')}"`,
    e.author_username,
    e.status,
    e.prompt_tokens,
    e.completion_tokens,
    e.total_tokens,
    e.cost_usd.toFixed(6),
    e.duration_ms,
    e.reviewed_at,
  ])
  const csv = [headers.join(','), ...rows.map(r => r.join(','))].join('\n')
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `mergeye-events-${new Date().toISOString().slice(0, 10)}.csv`
  link.click()
  URL.revokeObjectURL(url)
}

onMounted(() => {
  load()
  loadClients()
})
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.seta {
  color: var(--muted);
}

.counter {
  overflow-wrap: anywhere;
}
</style>
