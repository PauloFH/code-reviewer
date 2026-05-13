<template>
  <div class="page">
    <h2>Events</h2>

    <div class="filters">
      <input v-model="filterClient" placeholder="Filter by client" @keyup.enter="applyFilter" />
      <input v-model="filterSince" type="date" @change="applyFilter" />
      <input v-model="filterUntil" type="date" @change="applyFilter" />
      <button @click="applyFilter">Apply</button>
    </div>

    <EventsTable :events="events" :loading="loading" />

    <div class="pagination">
      <button :disabled="page === 0" @click="prevPage">← Prev</button>
      <span>Page {{ page + 1 }} of {{ totalPages }}</span>
      <button :disabled="page >= totalPages - 1" @click="nextPage">Next →</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import EventsTable from '@/components/EventsTable.vue'
import { listEvents, type ReviewEvent } from '@/api/events'

const events = ref<ReviewEvent[]>([])
const total = ref(0)
const page = ref(0)
const loading = ref(false)
const pageSize = 25

const filterClient = ref('')
const filterSince = ref('')
const filterUntil = ref('')

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

onMounted(load)

async function load() {
  loading.value = true
  try {
    const res = await listEvents({
      client: filterClient.value || undefined,
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

function applyFilter() { page.value = 0; load() }
function prevPage() { page.value--; load() }
function nextPage() { page.value++; load() }
</script>

<style scoped>
.page { display: flex; flex-direction: column; gap: 1.25rem; }
h2 { font-size: 1.5rem; font-weight: 700; color: #f8fafc; }
.filters { display: flex; gap: .75rem; flex-wrap: wrap; }
.filters input {
  padding: .5rem .75rem;
  background: #1e293b; border: 1px solid #334155;
  border-radius: 6px; color: #e2e8f0; font-size: .875rem;
}
.filters input:focus { outline: none; border-color: #6366f1; }
.filters button {
  padding: .5rem 1rem; background: #6366f1; color: white;
  border: none; border-radius: 6px; cursor: pointer; font-size: .875rem;
}
.pagination { display: flex; align-items: center; gap: 1rem; font-size: .875rem; color: #94a3b8; }
.pagination button {
  padding: .4rem .75rem; background: #1e293b;
  border: 1px solid #334155; color: #e2e8f0; border-radius: 6px; cursor: pointer;
}
.pagination button:disabled { opacity: .4; cursor: not-allowed; }
</style>
