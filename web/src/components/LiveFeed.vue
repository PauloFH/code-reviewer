<template>
  <div class="feed" ref="feedEl">
    <div v-if="feed.length === 0" class="empty">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
        <path d="M22 12h-3.5l-2.5 7-5-16-2.5 9H2"/>
      </svg>
      <span>Waiting for reviews…</span>
    </div>
    <TransitionGroup name="feed-item">
      <div v-for="e in feed" :key="e.id" class="feed-item">
        <div :class="['feed-marker', e.status === 'success' ? '' : e.status === 'skipped' ? 'skip' : 'danger']"></div>
        <div>
          <div class="feed-title">
            <a v-if="e.mr_url" :href="e.mr_url" target="_blank" rel="noopener">
              {{ e.mr_title || 'MR #' + e.mr_iid }}
            </a>
            <span v-else>{{ e.mr_title || 'MR #' + e.mr_iid }}</span>
          </div>
          <div class="feed-meta">
            <span>{{ e.client_name }}</span>
            <span>{{ e.project_ref }}</span>
            <span>{{ fmtTokens(e.total_tokens) }} tokens</span>
            <span>${{ e.cost_usd.toFixed(5) }}</span>
            <span>{{ e.duration_ms }}ms</span>
            <span :class="['status', e.status]">{{ e.status }}</span>
          </div>
          <div v-if="e.error_message" class="feed-error">{{ e.error_message }}</div>
        </div>
        <div class="feed-time">{{ relTime(e.reviewed_at) }}</div>
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { createEventSource, listEvents, type ReviewEvent } from '@/api/events'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const feed = ref<ReviewEvent[]>([])
const feedEl = ref<HTMLElement | null>(null)

let es: EventSource | null = null

onMounted(async () => {
  await loadRecent()
  startSSE()
})

onUnmounted(() => es?.close())

async function loadRecent() {
  try {
    const res = await listEvents({ limit: 20 })
    feed.value = (res.items || []).filter(e => e.status === 'success')
  } catch { /* ignore */ }
}

function startSSE() {
  if (!auth.token) return
  es = createEventSource(auth.token)
  es.onmessage = (ev) => {
    try {
      const e: ReviewEvent = JSON.parse(ev.data)
      if (e.status !== 'success') return
      feed.value.unshift(e)
      if (feed.value.length > 50) feed.value.pop()
    } catch { /* ignore */ }
  }
}

function fmtTokens(n: number) {
  return n >= 1000 ? (n / 1000).toFixed(1) + 'k' : String(n)
}

function relTime(ts: string) {
  const s = Math.floor((Date.now() - new Date(ts).getTime()) / 1000)
  if (s < 60) return `${s}s ago`
  if (s < 3600) return `${Math.floor(s / 60)}m ago`
  return `${Math.floor(s / 3600)}h ago`
}
</script>

<style scoped>
.feed {
  max-height: 400px;
  overflow-y: auto;
}

.empty {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 48px 32px;
  color: var(--muted);
  font-size: 13px;
  font-family: var(--font-mono);
}

.empty svg {
  opacity: 0.5;
}

.feed-item-enter-active {
  transition: all 0.2s ease;
}

.feed-item-enter-from {
  opacity: 0;
  transform: translateY(-8px);
}

.feed-error {
  font-size: 11px;
  color: var(--danger);
  margin-top: 4px;
  font-family: var(--font-mono);
}

.status {
  font-family: var(--font-mono);
  font-size: 11px;
}

.status.success {
  color: var(--ok);
}

.status.skipped {
  color: var(--skip);
}

.status.failed {
  color: var(--danger);
}
</style>