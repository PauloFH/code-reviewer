<template>
  <div>
    <div v-if="loading" class="loading">
      <span class="spinner" /> Carregando…
    </div>
    <div v-else class="scroll-x">
      <table class="table">
        <thead>
          <tr>
            <th>Horário</th>
            <th>Cliente</th>
            <th>MR</th>
            <th class="situacao">Situação</th>
            <th class="num">Tokens</th>
            <th class="num">Custo</th>
            <th class="num">Duração</th>
            <th class="acao"><span class="sr">Ações</span></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="e in events" :key="e.id">
            <td class="mono">{{ fmtDate(e.reviewed_at) }}</td>
            <td class="muted cliente">{{ e.client_name }}</td>
            <td class="mr-cell">
              <a v-if="e.mr_url" :href="e.mr_url" target="_blank" rel="noopener" class="mr-link" :title="mrTitle(e)">
                {{ mrTitle(e) }}
              </a>
              <span v-else class="mr-title" :title="mrTitle(e)">{{ mrTitle(e) }}</span>
              <div v-if="e.error_message" class="err-msg">{{ e.error_message }}</div>
            </td>
            <td>
              <span class="pill" :class="statusMark(e.status).tone">
                <span class="sinal" aria-hidden="true">{{ statusMark(e.status).sign }}</span>{{ statusMark(e.status).label }}
              </span>
            </td>
            <td class="num mono">{{ formatCount(e.total_tokens) }}</td>
            <td class="num mono">{{ formatUsd(e.cost_usd) }}</td>
            <td class="num mono">{{ fmtDuration(e.duration_ms) }}</td>
            <td>
              <button
                class="btn sm ghost"
                type="button"
                :disabled="reanalyzing === e.id"
                title="Re-analisar"
                aria-label="Re-analisar"
                @click="handleReanalyze(e)"
              >
                <AppIcon name="refresh" :size="12" />
              </button>
            </td>
          </tr>
          <tr v-if="events.length === 0">
            <td colspan="8" class="vazio">Nenhuma revisão encontrada</td>
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
import AppIcon from '@/components/AppIcon.vue'
import { statusMark } from '@/brand/status'
import { formatCount, formatUsd } from '@/brand/format'

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

const mrTitle = (e: ReviewEvent) => e.mr_title || '#' + e.mr_iid

function fmtDuration(ms: number) {
  return (ms / 1000).toLocaleString('pt-BR', { minimumFractionDigits: 1, maximumFractionDigits: 1 }) + ' s'
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

/* Abaixo disso a tabela rola dentro de .scroll-x em vez de espremer as colunas. */
.table {
  min-width: 760px;
}

.table th {
  white-space: nowrap;
}

.mono {
  font-family: var(--font-mono);
  font-size: 12px;
  white-space: nowrap;
}

.muted {
  color: var(--muted);
}

.cliente {
  max-width: 160px;
  overflow-wrap: anywhere;
}

.situacao {
  width: 130px;
}

.acao {
  width: 60px;
}

.mr-cell {
  max-width: 280px;
}

.mr-link,
.mr-title {
  color: var(--text);
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: block;
}

.mr-link {
  text-decoration: none;
}

.mr-link:hover {
  color: var(--accent);
  text-decoration: underline;
  text-decoration-color: var(--accent);
  text-underline-offset: 3px;
}

.err-msg {
  font-size: 11px;
  color: var(--danger);
  margin-top: 4px;
  font-family: var(--font-mono);
  overflow-wrap: anywhere;
}

.sinal {
  font-weight: 500;
}

.sr {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip-path: inset(50%);
  white-space: nowrap;
}

.num {
  text-align: right;
}

.vazio {
  text-align: center;
  color: var(--muted);
  padding: 48px;
}
</style>
