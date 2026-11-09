<template>
  <div class="feed">
    <div v-if="!loaded" aria-busy="true">
      <span class="sr-only" role="status">Carregando…</span>
      <div v-for="n in 3" :key="n" class="ev" aria-hidden="true">
        <span class="skeleton" />
        <div class="corpo">
          <span class="skeleton" :style="{ width: 40 + n * 12 + '%' }" />
          <span class="skeleton meta-skeleton" />
        </div>
      </div>
    </div>
    <div v-else-if="feed.length === 0" class="feed-empty">
      <AppIcon name="activity" :size="20" />
      <span>Aguardando revisões…</span>
    </div>
    <div v-for="e in feed" :key="e.id" class="ev" :class="{ novo: novos.has(e.id) }">
      <span class="sinal" :class="statusMark(e.status).tone">
        <span aria-hidden="true">{{ statusMark(e.status).sign }}</span>
        <span class="sr-only">{{ statusMark(e.status).label }}</span>
      </span>
      <div class="corpo">
        <a v-if="e.mr_url" class="titulo" :href="e.mr_url" target="_blank" rel="noopener">{{ mrTitle(e) }}</a>
        <span v-else class="titulo">{{ mrTitle(e) }}</span>
        <div class="meta">{{ meta(e) }}</div>
        <div v-if="e.error_message" class="erro" :class="{ falha: e.status === 'error' }">{{ e.error_message }}</div>
      </div>
      <span class="quando">{{ formatRelative(e.reviewed_at) }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { createEventSource, listEvents, type ReviewEvent } from '@/api/events'
import { apiErrorMessage } from '@/api/http'
import { useAuthStore } from '@/stores/auth'
import { useToastStore } from '@/stores/toast'
import AppIcon from '@/components/AppIcon.vue'
import { statusMark } from '@/domain'
import { createEventAdmitter } from '@/brand/feed'
import { formatCount, formatDuration, formatRelative, formatUsd, mrTitle } from '@/brand/format'

const props = defineProps<{ client?: string }>()
const emit = defineEmits<{ connection: [connected: boolean] }>()
const auth = useAuthStore()
const toast = useToastStore()
const feed = ref<ReviewEvent[]>([])
// Só eventos que chegam por SSE ganham a passada de marca-texto; a carga inicial não.
const novos = ref(new Set<number>())
const loaded = ref(false)

const admit = createEventAdmitter()
let es: EventSource | null = null
// Desmontar durante o await de loadRecent não pode deixar uma conexão SSE aberta.
let disposed = false

onMounted(async () => {
  await loadRecent()
  if (!disposed) startSSE()
})

onUnmounted(() => {
  disposed = true
  es?.close()
})

async function loadRecent() {
  try {
    const res = await listEvents({ client: props.client, limit: 20 })
    feed.value = res.items || []
  } catch (e) {
    toast.push(apiErrorMessage(e, 'Erro ao carregar revisões recentes'), 'error')
  }
  loaded.value = true
}

function startSSE() {
  if (!auth.token) return
  es = createEventSource(auth.token)
  es.onopen = () => emit('connection', true)
  es.onerror = () => emit('connection', false)
  // Mensagem ilegível lança no console e só ela se perde; a conexão continua.
  es.onmessage = (ev) => {
    const e: ReviewEvent = JSON.parse(ev.data)
    if (props.client && e.client_name !== props.client) return
    if (!admit(feed.value, e)) return
    novos.value.add(e.id)
    feed.value.unshift(e)
    if (feed.value.length > 50) novos.value.delete(feed.value.pop()!.id)
  }
}

function meta(e: ReviewEvent) {
  return [
    e.client_name,
    e.project_ref,
    e.total_tokens ? `${formatCount(e.total_tokens)} tokens` : 'sem tokens',
    formatUsd(e.cost_usd),
    formatDuration(e.duration_ms),
  ].join(' · ')
}
</script>

<style scoped>
.feed {
  max-height: 400px;
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: var(--hairline-strong) transparent;
}

.meta-skeleton {
  width: 30%;
  margin-top: 8px;
  height: 10px;
}

.feed-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 48px 32px;
  color: var(--muted);
  font-size: 13px;
  font-family: var(--font-mono);
}

.ev {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr) auto;
  gap: 10px;
  align-items: start;
  padding: 12px 16px;
  border-bottom: 1px solid var(--hairline);
}

.ev:last-child {
  border-bottom: 0;
}

.sinal {
  position: relative;
  font: 500 18px/1.2 var(--font-mono);
  text-align: center;
}

.sinal.ok {
  color: var(--ok);
}

.sinal.danger {
  color: var(--danger);
}

.sinal.warn {
  color: var(--warn);
}

.corpo {
  min-width: 0;
}

.titulo {
  color: inherit;
  font-weight: 600;
  text-decoration: none;
  overflow-wrap: anywhere;
}

a.titulo:hover {
  text-decoration: underline;
}

.meta {
  margin-top: 2px;
  font: 12px var(--font-mono);
  color: var(--muted);
  overflow-wrap: anywhere;
}

.erro {
  margin-top: 3px;
  font: 12px var(--font-mono);
  color: var(--muted);
  overflow-wrap: anywhere;
}

.erro.falha {
  color: var(--danger);
}

.quando {
  font: 12px var(--font-mono);
  color: var(--muted);
  white-space: nowrap;
}

.novo {
  animation: entra 0.45s cubic-bezier(0.16, 1, 0.3, 1) both;
}

.novo .titulo {
  background: var(--highlight) left 60% / 0 100% no-repeat;
  animation: risca 2.6s cubic-bezier(0.16, 1, 0.3, 1) both;
}

@keyframes entra {
  from {
    opacity: 0;
    transform: translateY(-8px);
  }
}

@keyframes risca {
  0% {
    background-size: 0 100%;
    color: inherit;
  }
  18%,
  72% {
    background-size: 100% 100%;
    color: var(--ink);
  }
  100% {
    background-position: right 60%;
    background-size: 0 100%;
  }
}

@media (max-width: 760px) {
  .ev {
    grid-template-columns: 24px minmax(0, 1fr);
  }

  .quando {
    grid-column: 2;
  }
}

@media (prefers-reduced-motion: reduce) {
  .novo,
  .novo .titulo {
    animation: none;
  }
}
</style>
