<template>
  <div class="page">
    <!-- Page head -->
    <div class="page-head">
      <div>
        <div class="page-eyebrow">Reviews · histórico</div>
        <h1 class="page-title">Events</h1>
        <div class="page-sub">Cada evento representa uma MR/PR processada pelo agente. Filtros e drill-down disponíveis.</div>
      </div>
      <div class="page-actions">
        <button class="btn" @click="exportCSV">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
            <path d="M15 3h6v6M10 14L21 3"/>
          </svg>
          Exportar CSV
        </button>
      </div>
    </div>

    <!-- Filterbar -->
    <div class="filterbar">
      <div class="field">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="11" cy="11" r="7"/>
          <path d="M21 21l-4.3-4.3"/>
        </svg>
        <input v-model="filterSearch" placeholder="Buscar por título, autor ou hash…" @keyup.enter="applyFilter" />
      </div>
      <div class="field">
        <label>Cliente</label>
        <select v-model="filterClient">
          <option value="">todos</option>
          <option v-for="c in clients" :key="c" :value="c">{{ c }}</option>
        </select>
      </div>
      <div class="field">
        <label>Status</label>
        <select v-model="filterStatus">
          <option value="">todos</option>
          <option value="success">success</option>
          <option value="skipped">skipped</option>
          <option value="error">error</option>
        </select>
      </div>
      <div class="field">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="5" width="18" height="16" rx="2"/>
          <path d="M3 10h18M8 3v4M16 3v4"/>
        </svg>
        <input v-model="filterSince" type="date" @change="applyFilter" />
        <span style="color: var(--muted)">→</span>
        <input v-model="filterUntil" type="date" @change="applyFilter" />
      </div>
      <button class="btn primary" @click="applyFilter">Aplicar</button>
      <span class="counter">{{ total.toLocaleString() }} resultados · {{ totalTokens }} tokens · ${{ totalCost.toFixed(4) }}</span>
    </div>

    <!-- Table -->
    <div class="card">
      <EventsTable :events="events" :loading="loading" />
    </div>

    <!-- Pagination -->
    <div class="pagination">
      <span>Página {{ page + 1 }} de {{ totalPages }} · {{ total.toLocaleString() }} resultados</span>
      <div class="pagination-controls">
        <button class="btn sm" :disabled="page === 0" @click="prevPage">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M15 18l-6-6 6-6"/>
          </svg>
          Anterior
        </button>
        <button class="btn sm" :disabled="page >= totalPages - 1" @click="nextPage">
          Próxima
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M9 18l6-6-6-6"/>
          </svg>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import EventsTable from '@/components/EventsTable.vue'
import { listEvents, type ReviewEvent } from '@/api/events'
import { listClients } from '@/api/clients'

const events = ref<ReviewEvent[]>([])
const clients = ref<string[]>([])
const total = ref(0)
const page = ref(0)
const loading = ref(false)
const pageSize = 25

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
  link.download = `code-reviewer-events-${new Date().toISOString().slice(0, 10)}.csv`
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
}
</style>