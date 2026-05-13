<template>
  <div>
    <h2>Dashboard</h2>
    <div v-if="stats" class="stats-grid">
      <div class="card">
        <div class="label">Total Reviews</div>
        <div class="value">{{ stats.total_reviews }}</div>
      </div>
      <div class="card">
        <div class="label">Last 30 days</div>
        <div class="value">{{ stats.reviews_last_30d }}</div>
      </div>
      <div class="card">
        <div class="label">Total Tokens</div>
        <div class="value">{{ fmtNum(stats.total_tokens) }}</div>
      </div>
      <div class="card">
        <div class="label">Total Cost</div>
        <div class="value">${{ stats.total_cost_usd.toFixed(4) }}</div>
      </div>
      <div class="card">
        <div class="label">Cost (30d)</div>
        <div class="value">${{ stats.cost_last_30d.toFixed(4) }}</div>
      </div>
    </div>

    <div class="section-header">
      <h3>Live Feed</h3>
      <span :class="['badge', connected ? 'connected' : 'disconnected']">
        {{ connected ? 'Live' : 'Disconnected' }}
      </span>
    </div>
    <div class="feed" ref="feedEl">
      <div v-for="e in feed" :key="e.id" :class="['feed-item', e.status]">
        <span class="status-dot"></span>
        <div class="feed-body">
          <div class="feed-title">{{ e.mr_title || 'MR #' + e.mr_iid }}</div>
          <div class="feed-meta">
            {{ e.client_name }} · {{ e.project_ref }} ·
            {{ fmtTokens(e.total_tokens) }} tokens · ${{ e.cost_usd.toFixed(5) }} ·
            {{ e.duration_ms }}ms
          </div>
          <div v-if="e.error_message" class="feed-error">{{ e.error_message }}</div>
        </div>
        <div class="feed-time">{{ relTime(e.reviewed_at) }}</div>
      </div>
      <div v-if="feed.length === 0" class="feed-empty">Waiting for reviews…</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { getStats, createEventSource, type ReviewEvent, type ReviewStats } from '@/api/events'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const stats = ref<ReviewStats | null>(null)
const feed = ref<ReviewEvent[]>([])
const feedEl = ref<HTMLElement | null>(null)
const connected = ref(false)

let es: EventSource | null = null

onMounted(async () => {
  stats.value = await getStats()
  startSSE()
})

onUnmounted(() => { es?.close() })

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
      nextTick(() => {
        feedEl.value?.scrollTo({ top: 0, behavior: 'smooth' })
      })
    } catch { /* ignore malformed */ }
  }
}

function fmtNum(n: number) {
  return new Intl.NumberFormat().format(n)
}

function fmtTokens(n: number) {
  return n >= 1000 ? (n / 1000).toFixed(1) + 'k' : String(n)
}

function relTime(ts: string) {
  const diff = Date.now() - new Date(ts).getTime()
  const s = Math.floor(diff / 1000)
  if (s < 60) return `${s}s ago`
  if (s < 3600) return `${Math.floor(s / 60)}m ago`
  return `${Math.floor(s / 3600)}h ago`
}
</script>

<style scoped>
h2 { font-size: 1.5rem; font-weight: 700; color: #f8fafc; margin-bottom: 1.5rem; }
h3 { font-size: 1.1rem; font-weight: 600; color: #e2e8f0; }
.stats-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(160px, 1fr)); gap: 1rem; margin-bottom: 2rem; }
.card { background: #1e293b; border: 1px solid #334155; border-radius: 10px; padding: 1.2rem; }
.label { font-size: .75rem; color: #64748b; margin-bottom: .5rem; text-transform: uppercase; letter-spacing: .05em; }
.value { font-size: 1.6rem; font-weight: 700; color: #f1f5f9; }
.section-header { display: flex; align-items: center; gap: 1rem; margin-bottom: .75rem; }
.badge { font-size: .7rem; padding: .2rem .6rem; border-radius: 999px; font-weight: 600; }
.badge.connected { background: #052e16; color: #4ade80; border: 1px solid #166534; }
.badge.disconnected { background: #450a0a; color: #f87171; border: 1px solid #7f1d1d; }
.feed { background: #1e293b; border: 1px solid #334155; border-radius: 10px; max-height: 420px; overflow-y: auto; }
.feed-item { display: flex; align-items: flex-start; gap: .75rem; padding: .8rem 1rem; border-bottom: 1px solid #1e293b; }
.feed-item:last-child { border-bottom: none; }
.feed-item.success .status-dot { background: #4ade80; }
.feed-item.skipped .status-dot { background: #94a3b8; }
.feed-item.error .status-dot { background: #f87171; }
.status-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; margin-top: .35rem; }
.feed-body { flex: 1; }
.feed-title { font-size: .9rem; color: #e2e8f0; }
.feed-meta { font-size: .75rem; color: #64748b; margin-top: .2rem; }
.feed-error { font-size: .75rem; color: #f87171; margin-top: .2rem; }
.feed-time { font-size: .75rem; color: #475569; white-space: nowrap; }
.feed-empty { padding: 2rem; text-align: center; color: #475569; font-size: .9rem; }
</style>
