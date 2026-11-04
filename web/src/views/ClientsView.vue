<template>
  <div class="page">
    <!-- Page head -->
    <div class="page-head">
      <div>
        <div class="page-eyebrow">Integrações</div>
        <h1 class="page-title">Clientes</h1>
        <div class="page-sub">Hosts conectados ao Mergeye. Cada cliente expõe seus projetos para revisão automatizada.</div>
      </div>
      <div class="page-actions">
        <button class="btn" @click="syncClients">
          <AppIcon name="refresh" :size="13" />
          Sincronizar
        </button>
        <button class="btn primary" @click="openCreate">
          <AppIcon name="plus" :size="13" />
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
        <span class="spinner" /> Carregando…
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
            <div class="client-meta">revisões</div>
          </div>
          <div style="text-align: right;">
            <div class="num-emph">${{ (clientStats[c.name]?.cost || 0).toFixed(4) }}</div>
            <div class="client-meta">custo</div>
          </div>
          <div style="display: flex; gap: 8px; align-items: center;">
            <span class="pill ok">
              <span class="dot"></span>conectado
            </span>
            <button class="btn sm ghost" title="Configurações" aria-label="Configurações" @click="openEdit(c)">
              <AppIcon name="settings" :size="12" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Footer hint -->
    <div class="foothint">
      <AppIcon name="merge" :size="14" />
      Conecte outros hosts (GitHub, Bitbucket, Azure DevOps) para revisar MRs em múltiplas plataformas a partir do mesmo painel.
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
import AppIcon from '@/components/AppIcon.vue'
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
    toast.push('Falha ao carregar clientes', 'error')
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
      toast.push('Cliente atualizado', 'success')
    } else {
      await createClient(data)
      toast.push('Cliente criado', 'success')
    }
    showForm.value = false
    await load()
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: string } })?.response?.data || 'Ocorreu um erro'
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

/* URLs longas quebram em vez de empurrar a linha. */
.client-name > div:last-child {
  min-width: 0;
  overflow-wrap: anywhere;
}
</style>