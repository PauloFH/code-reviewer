<template>
  <div>
    <div class="feed-header">
      <h3>Live Feed</h3>
      <span :class="['badge', connected ? 'connected' : 'disconnected']">
        {{ connected ? 'Live' : 'Disconnected' }}
      </span>
    </div>
    <div class="feed" ref="feedEl">
      <TransitionGroup name="feed-item">
        <div v-for="e in feed" :key="e.id" :class="['item', e.status]">
          <span class="dot"></span>
          <div class="body">
            <div class="title">
              <a v-if="e.mr_url" :href="e.mr_url" target="_blank" rel="noopener">
                {{ e.mr_title || 'MR #' + e.mr_iid }}
              </a>
              <span v-else>{{ e.mr_title || 'MR #' + e.mr_iid }}</span>
            </div>
            <div class="meta">
              {{ e.client_name }} · {{ e.project_ref }} ·
              {{ fmtTokens(e.total_tokens) }} tokens · ${{ e.cost_usd.toFixed(5) }} ·
              {{ e.duration_ms }}ms
            </div>
            <div v-if="e.error_message" class="err">{{ e.error_message }}</div>
          </div>
          <div class="time">{{ relTime(e.reviewed_at) }}</div>
        </div>
      </TransitionGroup>
      <div v-if="feed.length === 0" class="empty">Waiting for reviews…</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { createEventSource, type ReviewEvent } from '@/api/events'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const feed = ref<ReviewEvent[]>([])
const connected = ref(false)
const feedEl = ref<HTMLElement | null>(null)

let es: EventSource | null = null

onMounted(startSSE)
onUnmounted(() => es?.close())

function startSSE() {
  if (!auth.token) return
  es = createEventSource(auth.token)
  es.onopen = () => { connected.value = true }
  es.onerror = () => { connected.value = false }
  es.onmessage = (ev) => {
    try {
      const e: ReviewEvent = JSON.parse(ev.data)
      feed.value.unshift(e)
      if (feed.value.length > 100) feed.value.pop()
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
.feed-header { display: flex; align-items: center; gap: .75rem; margin-bottom: .75rem; }
h3 { font-size: 1rem; font-weight: 600; color: #e2e8f0; }
.badge { font-size: .7rem; padding: .2rem .6rem; border-radius: 999px; font-weight: 600; }
.badge.connected    { background: #052e16; color: #4ade80; border: 1px solid #166534; }
.badge.disconnected { background: #450a0a; color: #f87171; border: 1px solid #7f1d1d; }
.feed { background: #1e293b; border: 1px solid #334155; border-radius: 10px; max-height: 400px; overflow-y: auto; }
.item { display: flex; align-items: flex-start; gap: .75rem; padding: .75rem 1rem; border-bottom: 1px solid #0f172a; }
.item:last-child { border-bottom: none; }
.item.success .dot { background: #4ade80; }
.item.skipped .dot { background: #94a3b8; }
.item.error   .dot { background: #f87171; }
.dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; margin-top: .35rem; }
.body { flex: 1; min-width: 0; }
.title { font-size: .875rem; color: #e2e8f0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.title a { color: #818cf8; text-decoration: none; }
.title a:hover { text-decoration: underline; }
.meta { font-size: .72rem; color: #64748b; margin-top: .2rem; }
.err  { font-size: .72rem; color: #f87171; margin-top: .2rem; }
.time { font-size: .72rem; color: #475569; white-space: nowrap; flex-shrink: 0; }
.empty { padding: 2rem; text-align: center; color: #475569; font-size: .875rem; }
.feed-item-enter-active { transition: all .2s ease; }
.feed-item-enter-from   { opacity: 0; transform: translateY(-8px); }
</style>
