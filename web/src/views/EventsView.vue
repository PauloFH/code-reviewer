<template>
  <div class="page">
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

    <div class="filterbar">
      <div class="field">
        <AppIcon name="search" :size="13" />
        <input v-model="filterSearch" aria-label="Buscar" placeholder="Buscar por título, autor ou hash…" @keyup.enter="applyFilter" />
      </div>
      <div v-if="clients.length" class="field">
        <label for="filtro-cliente">Cliente</label>
        <select id="filtro-cliente" v-model="filterClient">
          <option value="">todos</option>
          <option v-for="c in clients" :key="c.name" :value="c.name">{{ c.name }}</option>
        </select>
      </div>
      <div class="field">
        <label for="filtro-situacao">Situação</label>
        <select id="filtro-situacao" v-model="filterStatus">
          <option value="">todas</option>
          <option v-for="s in EVENT_STATUSES" :key="s.value" :value="s.value">{{ s.label }}</option>
        </select>
      </div>
      <div class="field">
        <AppIcon name="calendar" :size="13" />
        <input v-model="filterSince" type="date" aria-label="Data inicial" @change="applyFilter" />
        <span class="seta" aria-hidden="true">→</span>
        <input v-model="filterUntil" type="date" aria-label="Data final" @change="applyFilter" />
      </div>
      <button class="btn primary" type="button" @click="applyFilter">Aplicar</button>
      <span class="counter">{{ formatCount(total) }} {{ total === 1 ? 'resultado' : 'resultados' }}</span>
    </div>

    <div class="card">
      <EventsTable :events="events" :loading="loading" />
    </div>

    <div class="pagination">
      <span>Página {{ page + 1 }} de {{ totalPages }}</span>
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
import { formatCount } from '@/brand/format'
import { toCsv } from '@/brand/csv'
import { listEvents, type ReviewEvent } from '@/api/events'
import { apiErrorMessage } from '@/api/http'
import { useToastStore } from '@/stores/toast'
import { useClients } from '@/composables/useClients'
import { EVENT_STATUSES, type EventStatus } from '@/domain'

const PAGE_SIZE = 25

const toast = useToastStore()
const clients = useClients()
const events = ref<ReviewEvent[]>([])
const total = ref(0)
const page = ref(0)
const loading = ref(false)

const filterSearch = ref('')
const filterClient = ref('')
const filterStatus = ref<EventStatus | ''>('')
const filterSince = ref('')
const filterUntil = ref('')

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE)))

async function load() {
  loading.value = true
  try {
    const res = await listEvents({
      client: filterClient.value || undefined,
      search: filterSearch.value || undefined,
      status: filterStatus.value || undefined,
      since: filterSince.value ? filterSince.value + 'T00:00:00Z' : undefined,
      until: filterUntil.value ? filterUntil.value + 'T23:59:59Z' : undefined,
      limit: PAGE_SIZE,
      offset: page.value * PAGE_SIZE,
    })
    events.value = res.items ?? []
    total.value = res.total
  } catch (e) {
    toast.push(apiErrorMessage(e, 'Erro ao carregar revisões'), 'error')
  } finally {
    loading.value = false
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

// shortcut: exporta só a página visível; exportar tudo pediria um endpoint de exportação no backend.
function exportCSV() {
  const header = ['Client', 'Project', 'MR IID', 'MR Title', 'Author', 'Status', 'Input Tokens', 'Output Tokens', 'Total Tokens', 'Cost (USD)', 'Duration (ms)', 'Reviewed At']
  const rows = events.value.map((e) => [
    e.client_name,
    e.project_ref,
    e.mr_iid,
    e.mr_title,
    e.author_username,
    e.status,
    e.prompt_tokens,
    e.completion_tokens,
    e.total_tokens,
    e.cost_usd.toFixed(6),
    e.duration_ms,
    e.reviewed_at,
  ])
  const blob = new Blob([toCsv([header, ...rows])], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `mergeye-events-${new Date().toISOString().slice(0, 10)}.csv`
  link.click()
  URL.revokeObjectURL(url)
}

onMounted(load)
</script>

<style scoped>
.filterbar {
  display: flex;
  gap: 10px;
  align-items: center;
  flex-wrap: wrap;
  padding: 16px 0 24px;
  border-bottom: 1px solid var(--hairline);
  margin-bottom: 24px;
}

.filterbar .field {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--panel);
  border: 1px solid var(--field-border);
  border-radius: 5px;
  padding: 6px 10px;
  font-size: 12.5px;
  color: var(--text);
  max-width: 100%;
}

.filterbar .field svg {
  width: 13px;
  height: 13px;
  color: var(--muted);
  flex-shrink: 0;
}

.filterbar .field input,
.filterbar .field select {
  background: transparent;
  border: 0;
  outline: 0;
  color: inherit;
  font: inherit;
  min-width: 100px;
}

.filterbar .field:focus-within {
  border-color: var(--accent);
}

.filterbar .field label {
  color: var(--muted);
  font-size: 11.5px;
  font-family: var(--font-mono);
}

/* largura reservada: a linha não quebra quando os dados chegam */
.filterbar .counter {
  margin-left: auto;
  min-width: 14ch;
  text-align: right;
  font-size: 12px;
  color: var(--muted);
  font-family: var(--font-mono);
}

.pagination {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px 16px;
  margin-top: 16px;
  font-size: 12px;
  color: var(--muted);
  font-family: var(--font-mono);
}

.pagination-controls {
  display: flex;
  gap: 6px;
}

.seta {
  color: var(--muted);
}
</style>
