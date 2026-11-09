<template>
  <div class="page">
    <div class="page-head">
      <div>
        <div class="page-eyebrow">Configuração · modelos de IA</div>
        <h1 class="page-title">Modelos de IA</h1>
        <div class="page-sub">Configure provedores, modelos e credenciais de IA para o agente de revisão de código.</div>
      </div>
    </div>

    <div class="stack">
      <div class="card">
        <div class="card-head">
          <div>
            <h2 class="card-title">Configurações</h2>
            <div class="card-sub">{{ configs.length }} {{ configs.length === 1 ? 'configuração' : 'configurações' }} · prioridade: projeto → cliente → global</div>
            <p v-if="providersError" class="error providers-error" role="alert">{{ providersError }}</p>
          </div>
          <button class="btn primary" type="button" :disabled="!providers.length" @click="openForm(null)">
            <AppIcon name="plus" :size="13" />
            Nova configuração
          </button>
        </div>

        <SkeletonRows v-if="loading && !configs.length" :cols="COLUMNS.length" />
        <div v-else-if="configs.length === 0" class="empty-state">
          <div class="empty-title">Nenhuma configuração cadastrada</div>
          <div class="empty-sub">Configure um modelo de IA para usar na revisão de código. Sem configuração, o sistema usa o fallback do ambiente.</div>
        </div>
        <table v-else class="table">
          <thead>
            <tr>
              <th v-for="c in COLUMNS" :key="c">{{ c }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="c in configs" :key="c.id">
              <td class="name">{{ c.name }}</td>
              <td class="muted">{{ providerLabel(c.provider) }}</td>
              <td>
                <span class="pill" :class="c.mode === 'account' ? 'warn' : 'accent'">
                  <span class="dot"></span>{{ aiModeLabel(c.mode) }}
                </span>
              </td>
              <td>
                <span v-if="c.mode === 'account'" :class="['cred', c.account_connected ? 'ok' : 'off']">
                  <AppIcon v-if="c.account_connected" name="check" :size="12" />
                  {{ c.account_connected ? 'Conectado' : 'Não conectado' }}
                </span>
                <span v-else :class="['cred', c.has_api_key ? 'ok' : 'off']">
                  <AppIcon v-if="c.has_api_key" name="check" :size="12" />
                  {{ c.has_api_key ? 'Chave definida' : 'Sem chave' }}
                </span>
              </td>
              <td>
                <div class="row-actions">
                  <button class="btn sm" type="button" :disabled="testingId === c.id" :aria-label="`Testar ${c.name}`" @click="doTest(c)">
                    <AppIcon v-if="testingId !== c.id" name="play" :size="12" />
                    <span v-else class="spinner" />
                    Testar
                  </button>
                  <button class="btn sm ghost" type="button" title="Editar" :aria-label="`Editar ${c.name}`" :disabled="!providers.length" @click="openForm(c)">
                    <AppIcon name="edit" :size="12" />
                  </button>
                  <button class="btn sm ghost danger" type="button" title="Excluir" :aria-label="`Excluir ${c.name}`" @click="deleteTarget = c">
                    <AppIcon name="trash" :size="12" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <AssignmentsCard
        :api="aiConfigsApi.assignments"
        :assignments="assignments"
        :loading="loading"
        :targets="targets"
        target-label="Configuração"
        empty-sub="O sistema está usando a configuração padrão do ambiente (AI_PROVIDER + variáveis relacionadas)."
        @changed="load"
      />
    </div>

    <AiConfigForm
      v-if="showForm"
      :initial="editing"
      :providers="providers"
      :save="save"
      @close="showForm = false"
      @connected="onConnected"
    />

    <Modal v-if="testResult || testError" :title="`Resultado do teste — ${testName}`" max-width="680px" @close="closeTest">
      <p v-if="testError" class="error" role="alert">{{ testError }}</p>
      <template v-else-if="testResult">
        <p class="usage">
          {{ testResult.usage.total_tokens }} tokens
          (entrada {{ testResult.usage.prompt_tokens }} / saída {{ testResult.usage.completion_tokens }})
        </p>
        <pre class="review">{{ testResult.review_markdown }}</pre>
      </template>
    </Modal>

    <ConfirmModal
      v-if="deleteTarget"
      :name="deleteTarget.name"
      warning="Atribuições vinculadas serão removidas."
      :loading="deleting"
      @close="deleteTarget = null"
      @confirm="doDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import SkeletonRows from '@/components/SkeletonRows.vue'
import AppIcon from '@/components/AppIcon.vue'
import AiConfigForm from '@/components/AiConfigForm.vue'
import AssignmentsCard from '@/components/AssignmentsCard.vue'
import Modal from '@/components/Modal.vue'
import ConfirmModal from '@/components/ConfirmModal.vue'
import { useToastStore } from '@/stores/toast'
import { apiErrorMessage } from '@/api/http'
import { useLibrary } from '@/composables/useLibrary'
import { aiModeLabel } from '@/domain'
import {
  aiConfigsApi, listProviders, testConfig,
  type AIConfig, type AIProviderInfo, type TestResult,
} from '@/api/aiConfigs'

const COLUMNS = ['Nome', 'Provedor', 'Modo', 'Credencial', '']

const toast = useToastStore()
const { items: configs, assignments, loading, deleting, showForm, editing, deleteTarget, load, openForm, save, doDelete } =
  useLibrary(
    aiConfigsApi,
    {
      loadError: 'Erro ao carregar configurações',
      created: 'Configuração criada',
      updated: 'Configuração atualizada',
      deleted: (n) => `Configuração "${n}" excluída`,
    },
    // Conectar a conta exige o id: o modal continua aberto, agora editando a config criada.
    (created) => created.mode === 'account',
  )

const providers = ref<AIProviderInfo[]>([])
// O formulário depende do catálogo de provedores: sem ele, criar e editar ficam desabilitados.
const providersError = ref('')
const testingId = ref<number | null>(null)
const testResult = ref<TestResult | null>(null)
const testError = ref('')
const testName = ref('')

const providerLabel = (id: string) => providers.value.find((p) => p.id === id)?.label ?? id
const targets = computed(() => configs.value.map((c) => ({ id: c.id, name: `${c.name} (${providerLabel(c.provider)})` })))

onMounted(async () => {
  try {
    providers.value = await listProviders()
  } catch (e) {
    providersError.value = apiErrorMessage(e, 'Não foi possível carregar os provedores de IA; criar e editar configurações fica indisponível. Recarregue a página.')
  }
})

async function onConnected() {
  toast.push('Conta Claude conectada', 'success')
  await load()
}

async function doTest(c: AIConfig) {
  testingId.value = c.id
  testError.value = ''
  testResult.value = null
  testName.value = c.name
  try {
    const res = await testConfig(c.id)
    if (res.ok && res.usage && res.review_markdown !== undefined) {
      testResult.value = {
        review_markdown: res.review_markdown,
        inline_comments: res.inline_comments ?? null,
        usage: res.usage,
      }
    } else {
      testError.value = res.error || 'Falha no teste'
    }
  } catch (e) {
    testError.value = apiErrorMessage(e, 'Falha no teste')
  } finally {
    testingId.value = null
  }
}

function closeTest() {
  testResult.value = null
  testError.value = ''
}
</script>

<style scoped>
.providers-error {
  margin-top: 8px;
}

.cred {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  font-family: var(--font-mono);
}

.cred.ok {
  color: var(--ok);
}

.cred.off {
  color: var(--muted);
}

.usage {
  font-size: 12px;
  color: var(--muted);
  margin: 0 0 16px;
  font-family: var(--font-mono);
}

.review {
  background: var(--bg-2);
  border: 1px solid var(--hairline);
  border-radius: var(--radius);
  padding: 16px;
  color: var(--text);
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  overflow-y: auto;
  max-height: 400px;
  margin: 0;
}
</style>