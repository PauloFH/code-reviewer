<template>
  <div>
    <div v-if="loading" class="loading">
      <span class="spinner" /> Loading…
    </div>
    <div v-else class="table-wrap">
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
            <th>Ações</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="e in events" :key="e.id" :class="e.status">
            <td class="nowrap">{{ fmtDate(e.reviewed_at) }}</td>
            <td>{{ e.client_name }}</td>
            <td class="mr-cell">
              <a v-if="e.mr_url" :href="e.mr_url" target="_blank" rel="noopener">
                {{ e.mr_title || '#' + e.mr_iid }}
              </a>
              <span v-else>{{ e.mr_title || '#' + e.mr_iid }}</span>
              <div v-if="e.error_message" class="err-msg">{{ e.error_message }}</div>
            </td>
            <td><span :class="['badge', e.status]">{{ e.status }}</span></td>
            <td class="num">{{ e.total_tokens }}</td>
            <td class="num">${{ e.cost_usd.toFixed(5) }}</td>
            <td class="num">{{ e.duration_ms }}ms</td>
            <td>
              <button
                class="btn-reanalyze"
                :disabled="reanalyzing === e.id"
                @click="handleReanalyze(e)"
              >
                {{ reanalyzing === e.id ? '...' : '↻' }}
              </button>
            </td>
          </tr>
          <tr v-if="events.length === 0">
            <td colspan="8" class="empty">No events found</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { ReviewEvent } from '@/api/events'
import { reanalyze } from '@/api/reanalyze'
import { useToastStore } from '@/stores/toast'

defineProps<{ events: ReviewEvent[]; loading: boolean }>()

const toast = useToastStore()
const reanalyzing = ref<number | null>(null)

function fmtDate(ts: string) {
  return new Date(ts).toLocaleString()
}

async function handleReanalyze(e: ReviewEvent) {
  reanalyzing.value = e.id
  try {
    await reanalyze({
      client_name: e.client_name,
      project_ref: e.project_ref,
      mr_iid: e.mr_iid,
    })
    toast.push('Re-análise iniciada', 'success')
  } catch {
    toast.push('Erro ao re-analisar', 'error')
  } finally {
    reanalyzing.value = null
  }
}
</script>

<style scoped>
.loading { display: flex; align-items: center; gap: .5rem; color: #64748b; padding: 2rem; }
.spinner {
  width: 14px; height: 14px;
  border: 2px solid #334155;
  border-top-color: #6366f1;
  border-radius: 50%;
  animation: spin .6s linear infinite;
  flex-shrink: 0;
}
@keyframes spin { to { transform: rotate(360deg); } }
.table-wrap { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; font-size: .875rem; }
th { text-align: left; padding: .6rem .75rem; color: #64748b; font-weight: 600; border-bottom: 1px solid #334155; white-space: nowrap; }
td { padding: .6rem .75rem; border-bottom: 1px solid #0f172a; color: #cbd5e1; vertical-align: top; }
tr.error td { background: rgba(239,68,68,.04); }
.nowrap { white-space: nowrap; }
.num { font-variant-numeric: tabular-nums; white-space: nowrap; }
.mr-cell { max-width: 260px; }
.badge { font-size: .7rem; padding: .15rem .5rem; border-radius: 999px; font-weight: 600; white-space: nowrap; }
.badge.success { background: #052e16; color: #4ade80; }
.badge.skipped { background: #1e293b; color: #94a3b8; }
.badge.error   { background: #450a0a; color: #f87171; }
a { color: #818cf8; text-decoration: none; }
a:hover { text-decoration: underline; }
.err-msg { font-size: .72rem; color: #f87171; margin-top: .2rem; }
.empty { text-align: center; color: #475569; padding: 2rem; }
.btn-reanalyze {
  background: none; border: 1px solid #475569; color: #94a3b8;
  padding: .2rem .5rem; border-radius: 4px; cursor: pointer; font-size: .85rem;
}
.btn-reanalyze:hover { background: #334155; color: #e2e8f0; }
.btn-reanalyze:disabled { opacity: .4; cursor: not-allowed; }
</style>
