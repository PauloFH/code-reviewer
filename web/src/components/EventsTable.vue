<template>
  <div>
    <div v-if="loading" class="loading">
      <span class="spinner" /> Loading…
    </div>
    <table v-else class="table">
      <thead>
        <tr>
          <th>Time</th>
          <th>Client</th>
          <th>MR</th>
          <th style="width: 100px;">Status</th>
          <th class="num">Tokens</th>
          <th class="num">Cost</th>
          <th class="num">Duration</th>
          <th style="width: 60px;"></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="e in events" :key="e.id" :class="{ error: e.status === 'error' }">
          <td class="mono">{{ fmtDate(e.reviewed_at) }}</td>
          <td class="muted">{{ e.client_name }}</td>
          <td class="mr-cell">
            <a v-if="e.mr_url" :href="e.mr_url" target="_blank" rel="noopener" class="mr-link">
              {{ e.mr_title || '#' + e.mr_iid }}
            </a>
            <span v-else class="mr-title">{{ e.mr_title || '#' + e.mr_iid }}</span>
            <div v-if="e.error_message" class="err-msg">{{ e.error_message }}</div>
          </td>
          <td>
            <span class="pill" :class="statusClass(e.status)">
              <span class="dot"></span>{{ e.status }}
            </span>
          </td>
          <td class="num mono">{{ fmtTokens(e.total_tokens) }}</td>
          <td class="num mono">${{ e.cost_usd.toFixed(5) }}</td>
          <td class="num mono">{{ e.duration_ms }}ms</td>
          <td>
            <button
              class="btn sm ghost"
              :disabled="reanalyzing === e.id"
              title="Re-analisar"
              @click="handleReanalyze(e)"
            >
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
                <path d="M3 12a9 9 0 0 1 15.5-6.3L21 8"/>
                <path d="M21 3v5h-5"/>
                <path d="M21 12a9 9 0 0 1-15.5 6.3L3 16"/>
                <path d="M3 21v-5h5"/>
              </svg>
            </button>
          </td>
        </tr>
        <tr v-if="events.length === 0">
          <td colspan="8" class="empty">No events found</td>
        </tr>
      </tbody>
    </table>
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
  return new Date(ts).toLocaleString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function fmtTokens(n: number) {
  return n >= 1000 ? (n / 1000).toFixed(1) + 'k' : String(n)
}

function statusClass(status: string) {
  if (status === 'success') return 'ok'
  if (status === 'skipped') return 'skip'
  return 'danger'
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
.loading {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--muted);
  padding: 24px;
}

.mono {
  font-family: var(--font-mono);
  font-size: 12px;
  white-space: nowrap;
}

.muted {
  color: var(--muted);
}

.mr-cell {
  max-width: 260px;
}

.mr-link {
  color: var(--text);
  text-decoration: none;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: block;
}

.mr-link:hover {
  color: var(--accent);
  text-decoration: underline;
  text-decoration-color: var(--accent);
  text-underline-offset: 3px;
}

.mr-title {
  color: var(--text);
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: block;
}

.err-msg {
  font-size: 11px;
  color: var(--danger);
  margin-top: 4px;
  font-family: var(--font-mono);
}

.num {
  text-align: right;
}

.empty {
  text-align: center;
  color: var(--muted);
  padding: 48px;
}
</style>