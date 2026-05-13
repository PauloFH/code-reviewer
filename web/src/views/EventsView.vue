<template>
  <div>
    <h2>Events</h2>

    <div class="filters">
      <input v-model="filterClient" placeholder="Filter by client" @keyup.enter="load" />
      <input v-model="filterSince" type="date" @change="load" />
      <input v-model="filterUntil" type="date" @change="load" />
      <button @click="load">Apply</button>
    </div>

    <div class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Time</th>
            <th>Client</th>
            <th>MR</th>
            <th>Status</th>
            <th>Tokens</th>
            <th>Cost</th>
            <th>Duration</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="e in events" :key="e.id" :class="e.status">
            <td>{{ fmtDate(e.reviewed_at) }}</td>
            <td>{{ e.client_name }}</td>
            <td>
              <a v-if="e.mr_url" :href="e.mr_url" target="_blank" rel="noopener">
                {{ e.mr_title || '#' + e.mr_iid }}
              </a>
              <span v-else>{{ e.mr_title || '#' + e.mr_iid }}</span>
            </td>
            <td><span :class="['badge', e.status]">{{ e.status }}</span></td>
            <td>{{ e.total_tokens }}</td>
            <td>${{ e.cost_usd.toFixed(5) }}</td>
            <td>{{ e.duration_ms }}ms</td>
          </tr>
          <tr v-if="events.length === 0">
            <td colspan="7" class="empty">No events found</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="pagination">
      <button :disabled="page === 0" @click="prevPage">← Prev</button>
      <span>Page {{ page + 1 }} of {{ totalPages }}</span>
      <button :disabled="page >= totalPages - 1" @click="nextPage">Next →</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { listEvents, type ReviewEvent } from '@/api/events'

const events = ref<ReviewEvent[]>([])
const total = ref(0)
const page = ref(0)
const pageSize = 25

const filterClient = ref('')
const filterSince = ref('')
const filterUntil = ref('')

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

onMounted(load)

async function load() {
  const res = await listEvents({
    client: filterClient.value || undefined,
    since: filterSince.value ? filterSince.value + 'T00:00:00Z' : undefined,
    until: filterUntil.value ? filterUntil.value + 'T23:59:59Z' : undefined,
    limit: pageSize,
    offset: page.value * pageSize,
  })
  events.value = res.items ?? []
  total.value = res.total
}

function prevPage() { page.value--; load() }
function nextPage() { page.value++; load() }

function fmtDate(ts: string) {
  return new Date(ts).toLocaleString()
}
</script>

<style scoped>
h2 { font-size: 1.5rem; font-weight: 700; color: #f8fafc; margin-bottom: 1.5rem; }
.filters { display: flex; gap: .75rem; margin-bottom: 1.5rem; flex-wrap: wrap; }
.filters input {
  padding: .5rem .75rem;
  background: #1e293b;
  border: 1px solid #334155;
  border-radius: 6px;
  color: #e2e8f0;
  font-size: .875rem;
}
.filters button {
  padding: .5rem 1rem;
  background: #6366f1;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: .875rem;
}
.table-wrap { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; font-size: .875rem; }
th { text-align: left; padding: .6rem .75rem; color: #64748b; font-weight: 600; border-bottom: 1px solid #334155; }
td { padding: .6rem .75rem; border-bottom: 1px solid #1e293b; color: #cbd5e1; }
tr.error td { background: rgba(239,68,68,.05); }
.badge { font-size: .7rem; padding: .15rem .5rem; border-radius: 999px; font-weight: 600; }
.badge.success { background: #052e16; color: #4ade80; }
.badge.skipped { background: #1e293b; color: #94a3b8; }
.badge.error { background: #450a0a; color: #f87171; }
a { color: #818cf8; text-decoration: none; }
a:hover { text-decoration: underline; }
.empty { text-align: center; color: #475569; padding: 2rem; }
.pagination { display: flex; align-items: center; gap: 1rem; margin-top: 1.25rem; font-size: .875rem; color: #94a3b8; }
.pagination button {
  padding: .4rem .75rem;
  background: #1e293b;
  border: 1px solid #334155;
  color: #e2e8f0;
  border-radius: 6px;
  cursor: pointer;
}
.pagination button:disabled { opacity: .4; cursor: not-allowed; }
</style>
