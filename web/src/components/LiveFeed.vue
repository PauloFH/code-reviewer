<template>
  <div class="feed">
    <div v-if="feed.length === 0" class="feed-empty">
      <AppIcon name="activity" :size="20" />
      <span>Aguardando revisões…</span>
    </div>
    <div v-for="e in feed" :key="e.id" class="ev" :class="{ novo: novos.has(e.id) }">
      <span class="sinal" :class="statusMark(e.status).tone">
        <span aria-hidden="true">{{ statusMark(e.status).sign }}</span>
        <span class="sr">{{ statusMark(e.status).label }}</span>
      </span>
      <div class="corpo">
        <a v-if="e.mr_url" class="titulo" :href="e.mr_url" target="_blank" rel="noopener">{{ title(e) }}</a>
        <span v-else class="titulo">{{ title(e) }}</span>
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
import { useAuthStore } from '@/stores/auth'
import AppIcon from '@/components/AppIcon.vue'
import { statusMark } from '@/brand/status'
import { createEventAdmitter } from '@/brand/feed'
import { formatCount, formatRelative, formatUsd } from '@/brand/format'

const props = defineProps<{ client?: string }>()
const emit = defineEmits<{ connection: [connected: boolean] }>()
const auth = useAuthStore()
const feed = ref<ReviewEvent[]>([])
// Só eventos que chegam por SSE ganham a passada de marca-texto; a carga inicial não.
const novos = ref(new Set<number>())

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
  } catch { /* ignore */ }
}

function startSSE() {
  if (!auth.token) return
  es = createEventSource(auth.token)
  es.onopen = () => emit('connection', true)
  es.onerror = () => emit('connection', false)
  es.onmessage = (ev) => {
    try {
      const e: ReviewEvent = JSON.parse(ev.data)
      if (props.client && e.client_name !== props.client) return
      if (!admit(feed.value, e)) return
      novos.value.add(e.id)
      feed.value.unshift(e)
      if (feed.value.length > 50) feed.value.pop()
    } catch { /* ignore */ }
  }
}

const title = (e: ReviewEvent) => e.mr_title || 'MR #' + e.mr_iid

function meta(e: ReviewEvent) {
  const seconds = (e.duration_ms / 1000).toLocaleString('pt-BR', { minimumFractionDigits: 1, maximumFractionDigits: 1 })
  return [
    e.client_name,
    e.project_ref,
    e.total_tokens ? `${formatCount(e.total_tokens)} tokens` : 'sem tokens',
    formatUsd(e.cost_usd),
    `${seconds}s`,
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

.sr {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip-path: inset(50%);
  white-space: nowrap;
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
