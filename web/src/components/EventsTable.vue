<template>
  <div>
    <SkeletonRows v-if="loading && !events.length" :cols="COLUMNS.length" :rows="25" />
    <div v-else class="scroll-x">
      <table class="table">
        <thead>
          <tr>
            <th v-for="c in COLUMNS" :key="c.label" :class="c.class">
              <span v-if="c.class === 'acao'" class="sr-only">{{ c.label }}</span>
              <template v-else>{{ c.label }}</template>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="e in events" :key="e.id">
            <td class="mono">{{ formatDateTime(e.reviewed_at) }}</td>
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
            <td class="num mono">{{ formatDuration(e.duration_ms) }}</td>
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
            <td :colspan="COLUMNS.length" class="vazio">Nenhuma revisão encontrada</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import SkeletonRows from '@/components/SkeletonRows.vue'
import type { ReviewEvent } from '@/api/events'
import { reanalyze } from '@/api/reanalyze'
import { apiErrorMessage } from '@/api/http'
import { useToastStore } from '@/stores/toast'
import AppIcon from '@/components/AppIcon.vue'
import { statusMark } from '@/domain'
import { formatCount, formatDateTime, formatDuration, formatUsd, mrTitle } from '@/brand/format'

defineProps<{ events: ReviewEvent[]; loading: boolean }>()

const COLUMNS = [
  { label: 'Horário' },
  { label: 'Cliente' },
  { label: 'MR' },
  { label: 'Situação', class: 'situacao' },
  { label: 'Tokens', class: 'num' },
  { label: 'Custo', class: 'num' },
  { label: 'Duração', class: 'num' },
  { label: 'Ações', class: 'acao' },
]

const toast = useToastStore()
const reanalyzing = ref<number | null>(null)

async function handleReanalyze(e: ReviewEvent) {
  reanalyzing.value = e.id
  try {
    await reanalyze({
      client_name: e.client_name,
      project_ref: e.project_ref,
      mr_iid: e.mr_iid,
    })
    toast.push('Re-análise iniciada', 'success')
  } catch (err) {
    toast.push(apiErrorMessage(err, 'Erro ao re-analisar'), 'error')
  } finally {
    reanalyzing.value = null
  }
}
</script>

<style scoped>
.scroll-x {
  overflow-x: auto;
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
  position: relative; /* ancora o .sr-only, que senão escapa do .scroll-x e alarga a página */
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

.num {
  text-align: right;
}

.vazio {
  text-align: center;
  color: var(--muted);
  padding: 48px;
}
</style>
