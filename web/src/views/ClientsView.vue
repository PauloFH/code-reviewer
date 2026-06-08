<template>
  <div class="page">
    <!-- Page head -->
    <div class="page-head">
      <div>
        <div class="page-eyebrow">Integrações</div>
        <h1 class="page-title">Clients</h1>
        <div class="page-sub">Hosts conectados ao Code Reviewer. Cada cliente expõe seus projetos para revisão automatizada.</div>
      </div>
      <div class="page-actions">
        <button class="btn" @click="syncClients">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M3 12a9 9 0 0 1 15.5-6.3L21 8"/>
            <path d="M21 3v5h-5"/>
            <path d="M21 12a9 9 0 0 1-15.5 6.3L3 16"/>
            <path d="M3 21v-5h5"/>
          </svg>
          Sincronizar
        </button>
        <button class="btn primary" @click="openCreate">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 5v14M5 12h14"/>
          </svg>
          Novo cliente
        </button>
      </div>
    </div>

    <!-- Clients card -->
    <div class="card">
      <div class="card-head">
        <div class="card-title">Hosts ativos</div>
        <div class="card-sub">{{ clients.length }} · conectado{{ clients.length !== 1 ? 's' : '' }}</div>
      </div>

      <div v-if="loading" class="loading-row">
        <span class="spinner" /> Loading…
      </div>
      <div v-else-if="clients.length === 0" class="empty-state">
        <div class="empty-title">Nenhum cliente configurado</div>
        <div class="empty-sub">Conecte um host de plataforma Git (GitLab, GitHub, Gitea) para começar a receber webhooks.</div>
      </div>
      <div v-else>
        <div v-for="c in clients" :key="c.name" class="client-row">
          <div class="client-name">
            <div class="client-mark">{{ getPlatformInitials(c.platform) }}</div>
            <div>
              <div style="font-weight: 500;">{{ c.name }}</div>
              <div class="client-meta">{{ c.base_url }} · {{ c.username }}</div>
            </div>
          </div>
          <div style="text-align: right;">
            <div class="num-emph">{{ clientStats[c.name]?.reviews || 0 }}</div>
            <div class="client-meta">reviews</div>
          </div>
          <div style="text-align: right;">
            <div class="num-emph">${{ (clientStats[c.name]?.cost || 0).toFixed(4) }}</div>
            <div class="client-meta">cost</div>
          </div>
          <div style="display: flex; gap: 8px; align-items: center;">
            <span class="pill ok">
              <span class="dot"></span>connected
            </span>
            <button class="btn sm ghost" title="Configurações" @click="openEdit(c)">
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="12" cy="12" r="3"/>
                <path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z"/>
              </svg>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Footer hint -->
    <div class="foothint">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
        <path d="M12 3l1.5 4.5L18 9l-4.5 1.5L12 15l-1.5-4.5L6 9l4.5-1.5zM19 16l.7 2 2 .7-2 .7-.7 2-.7-2-2-.7 2-.7z"/>
      </svg>
      Conecte outros hosts (GitHub, Bitbucket, Azure DevOps) para revisar MRs em múltiplas plataformas a partir do mesmo dashboard.
    </div>

    <!-- Create/Edit modal -->
    <ClientForm
      v-if="showForm"
      :initial="editingClient"
      ref="formRef"
      @close="showForm = false"
      @saved="handleSaved"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import ClientForm from '@/components/ClientForm.vue'
import { useToastStore } from '@/stores/toast'
import { listClients, createClient, updateClient, type Client, type ClientInput } from '@/api/clients'
import { getStats } from '@/api/events'

const toast = useToastStore()

const clients = ref<Client[]>([])
const clientStats = ref<Record<string, { reviews: number; cost: number }>>({})
const loading = ref(false)
const showForm = ref(false)
const editingClient = ref<Client | null>(null)
const formRef = ref<InstanceType<typeof ClientForm> | null>(null)

function getPlatformInitials(platform: string): string {
  const map: Record<string, string> = {
    gitlab: 'GL',
    'gitlab-selfhosted': 'GL',
    github: 'GH',
    gitea: 'GT',
    'gitea-selfhosted': 'GT',
  }
  return map[platform] || platform.slice(0, 2).toUpperCase()
}

async function load() {
  loading.value = true
  try {
    clients.value = await listClients()
    // Load stats for each client
    try {
      const stats = await getStats()
      clientStats.value = {}
      for (const c of stats.by_client) {
        clientStats.value[c.client_name] = { reviews: c.review_count, cost: c.total_cost_usd }
      }
    } catch {
      // ignore stats error
    }
  } catch {
    toast.push('Failed to load clients', 'error')
  } finally {
    loading.value = false
  }
}

function syncClients() {
  load()
  toast.push('Clientes sincronizados', 'success')
}

function openCreate() {
  editingClient.value = null
  showForm.value = true
}

function openEdit(c: Client) {
  editingClient.value = c
  showForm.value = true
}

async function handleSaved(data: ClientInput & { _oldName?: string }) {
  formRef.value?.setSaving(true)
  try {
    if (data._oldName) {
      const { _oldName, ...rest } = data
      await updateClient(_oldName, rest)
      toast.push('Client updated', 'success')
    } else {
      await createClient(data)
      toast.push('Client created', 'success')
    }
    showForm.value = false
    await load()
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: string } })?.response?.data || 'An error occurred'
    formRef.value?.setError(msg)
    formRef.value?.setSaving(false)
  }
}

onMounted(load)
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
}

.loading-row {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--muted);
  padding: 24px;
}

.empty-state {
  padding: 56px 32px;
  text-align: center;
}
</style>