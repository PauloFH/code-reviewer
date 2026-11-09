<template>
  <div class="page">
    <div class="page-head">
      <div>
        <div class="page-eyebrow">Integrações</div>
        <h1 class="page-title">Clientes</h1>
        <div class="page-sub">Hosts conectados ao Mergeye. Cada cliente expõe seus projetos para revisão automatizada.</div>
      </div>
      <div class="page-actions">
        <button class="btn" type="button" @click="syncClients">
          <AppIcon name="refresh" :size="13" />
          Sincronizar
        </button>
        <button class="btn primary" type="button" @click="openForm(null)">
          <AppIcon name="plus" :size="13" />
          Novo cliente
        </button>
      </div>
    </div>

    <div class="card">
      <div class="card-head">
        <div class="card-title">Hosts</div>
        <div class="card-sub">{{ clients.length }} {{ clients.length === 1 ? 'cadastrado' : 'cadastrados' }}</div>
      </div>

      <SkeletonRows v-if="loading && !clients.length" :cols="4" />
      <div v-else-if="clients.length === 0" class="empty-state">
        <div class="empty-title">Nenhum cliente configurado</div>
        <div class="empty-sub">Conecte um host de plataforma Git (GitLab, GitHub, Gitea) para começar a receber webhooks.</div>
      </div>
      <div v-else>
        <div v-for="c in clients" :key="c.name" class="client-row">
          <div class="client-name">
            <div class="client-mark">{{ platformInitials(c.platform) }}</div>
            <div>
              <div class="client-nome">{{ c.name }}</div>
              <div class="client-meta">{{ c.base_url }} · {{ c.username }}</div>
            </div>
          </div>
          <div class="client-num">
            <div class="num-emph">{{ formatCount(clientStats[c.name]?.reviews || 0) }}</div>
            <div class="client-meta">revisões</div>
          </div>
          <div class="client-num">
            <div class="num-emph">{{ formatUsd(clientStats[c.name]?.cost || 0) }}</div>
            <div class="client-meta">custo</div>
          </div>
          <div class="client-acoes">
            <button class="btn sm ghost" type="button" title="Configurações" :aria-label="`Configurações de ${c.name}`" @click="openForm(c)">
              <AppIcon name="settings" :size="12" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <div class="foothint">
      <AppIcon name="merge" :size="14" />
      Cada cliente pode ser GitLab, GitHub ou Gitea, então um só painel revisa MRs e PRs de plataformas diferentes.
    </div>

    <ClientForm v-if="showForm" :initial="editing" :save="save" @close="showForm = false" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import SkeletonRows from '@/components/SkeletonRows.vue'
import ClientForm from '@/components/ClientForm.vue'
import AppIcon from '@/components/AppIcon.vue'
import { useToastStore } from '@/stores/toast'
import { apiErrorMessage } from '@/api/http'
import { listClients, createClient, updateClient, type Client, type ClientInput } from '@/api/clients'
import { getStats } from '@/api/events'
import { formatCount, formatUsd } from '@/brand/format'
import { platformInitials } from '@/domain'

const toast = useToastStore()
const clients = ref<Client[]>([])
const clientStats = ref<Record<string, { reviews: number; cost: number }>>({})
const loading = ref(false)
const showForm = ref(false)
const editing = ref<Client | null>(null)

onMounted(load)

async function load(): Promise<boolean> {
  loading.value = true
  // As duas chamadas são independentes; sem as métricas, a lista aparece com zeros.
  const statsReq = getStats().catch(() => null)
  try {
    clients.value = await listClients()
    const stats = await statsReq
    if (stats) {
      clientStats.value = {}
      for (const c of stats.by_client) {
        clientStats.value[c.client_name] = { reviews: c.review_count, cost: c.total_cost_usd }
      }
    }
    return true
  } catch (e) {
    toast.push(apiErrorMessage(e, 'Falha ao carregar clientes'), 'error')
    return false
  } finally {
    loading.value = false
  }
}

async function syncClients() {
  if (await load()) toast.push('Clientes sincronizados', 'success')
}

function openForm(c: Client | null) {
  editing.value = c
  showForm.value = true
}

async function save(data: ClientInput) {
  if (editing.value) await updateClient(editing.value.name, data)
  else await createClient(data)
  toast.push(editing.value ? 'Cliente atualizado' : 'Cliente criado', 'success')
  showForm.value = false
  await load()
}
</script>

<style scoped>
.client-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto auto;
  gap: 24px;
  align-items: center;
  padding: 18px 22px;
  border-bottom: 1px solid var(--hairline);
}

.client-row:last-child {
  border-bottom: 0;
}

.client-row:hover {
  background: var(--panel-2);
}

.client-name {
  display: flex;
  align-items: center;
  gap: 12px;
}

.client-mark {
  width: 26px;
  height: 26px;
  border-radius: 6px;
  background: var(--bg-2);
  border: 1px solid var(--hairline);
  display: grid;
  place-items: center;
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 600;
  color: var(--text);
}

.client-meta {
  font-size: 11.5px;
  color: var(--muted);
  font-family: var(--font-mono);
  margin-top: 2px;
}

.foothint {
  margin-top: 28px;
  padding: 14px 18px;
  border: 1px solid var(--hairline);
  border-radius: var(--radius-lg);
  background: var(--panel);
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 12.5px;
  color: var(--muted);
}

.foothint svg {
  width: 14px;
  height: 14px;
  color: var(--accent);
  flex-shrink: 0;
}

.num-emph {
  font-family: var(--font-mono);
  color: var(--text);
  font-weight: 500;
}

/* Ações do cabeçalho quebram de linha em vez de sair da tela. */
.page-head,
.page-actions {
  flex-wrap: wrap;
}

.client-nome {
  font-weight: 500;
}

.client-num {
  text-align: right;
}

.client-acoes {
  display: flex;
  gap: 8px;
  align-items: center;
}

/* Em tela estreita: nome e URL em cima; números e ações embaixo. */
@media (max-width: 640px) {
  .client-row {
    grid-template-columns: auto auto minmax(0, 1fr);
    gap: 12px 20px;
  }

  .client-name {
    grid-column: 1 / -1;
  }

  .client-num {
    text-align: left;
  }

  .client-acoes {
    justify-content: flex-end;
  }
}

/* URLs longas quebram em vez de empurrar a linha. */
.client-name > div:last-child {
  min-width: 0;
  overflow-wrap: anywhere;
}
</style>