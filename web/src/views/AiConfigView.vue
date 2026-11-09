<template>
  <div class="page">
    <!-- Page head -->
    <div class="page-head">
      <div>
        <div class="page-eyebrow">Configuração · modelos de IA</div>
        <h1 class="page-title">Modelos de IA</h1>
        <div class="page-sub">Configure provedores, modelos e credenciais de IA para o agente de revisão de código.</div>
      </div>
    </div>

    <div class="stack">
      <!-- Configs card -->
      <div class="card">
        <div class="card-head">
          <div>
            <h2 class="card-title">Configurações</h2>
            <div class="card-sub">{{ configs.length }} {{ configs.length === 1 ? 'configuração' : 'configurações' }} · prioridade: projeto → cliente → global</div>
          </div>
          <button class="btn primary" @click="openCreate">
            <AppIcon name="plus" :size="13" />
            Nova configuração
          </button>
        </div>

        <SkeletonRows v-if="loading && !configs.length" :cols="6" />
        <div v-else-if="configs.length === 0" class="empty-state">
          <div class="empty-title">Nenhuma configuração cadastrada</div>
          <div class="empty-sub">Configure um modelo de IA para usar na revisão de código. Sem configuração, o sistema usa o fallback do ambiente.</div>
        </div>
        <table v-else class="table">
          <thead>
            <tr>
              <th>Nome</th>
              <th>Provedor</th>
              <th>Modo</th>
              <th>Credencial</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="c in configs" :key="c.id">
              <td class="name">{{ c.name }}</td>
              <td class="muted">{{ providerLabel(c.provider) }}</td>
              <td>
                <span class="pill" :class="c.mode === 'account' ? 'warn' : 'accent'">
                  <span class="dot"></span>{{ modeLabel(c.mode) }}
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
                  <button class="btn sm ghost" type="button" title="Editar" :aria-label="`Editar ${c.name}`" @click="openEdit(c)">
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

      <!-- Assignments card -->
      <div class="card">
        <div class="card-head">
          <div>
            <h2 class="card-title">Atribuições</h2>
            <div class="card-sub">Fallback do ambiente usado quando nenhuma atribuição resolve</div>
          </div>
          <button class="btn primary" @click="showAssignForm = true">
            <AppIcon name="plus" :size="13" />
            Nova atribuição
          </button>
        </div>

        <div v-if="assignments.length === 0" class="empty-state">
          <div class="empty-title">Nenhuma atribuição configurada</div>
          <div class="empty-sub">O sistema está usando a configuração padrão do ambiente (AI_PROVIDER + variáveis relacionadas).</div>
        </div>
        <table v-else class="table">
          <thead>
            <tr>
              <th class="col-scope">Escopo</th>
              <th>Cliente</th>
              <th>Projeto</th>
              <th>Configuração</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in assignments" :key="a.id">
              <td>
                <span class="pill" :class="a.scope === 'global' ? 'accent' : 'skip'">
                  <span class="dot"></span>{{ scopeLabel(a.scope) }}
                </span>
              </td>
              <td class="muted">{{ a.client_name || '—' }}</td>
              <td class="muted">{{ a.project_ref || '—' }}</td>
              <td>{{ a.config_name }}</td>
              <td>
                <div class="row-actions">
                  <button class="btn sm ghost danger" type="button" title="Remover atribuição" aria-label="Remover atribuição" @click="removeAssign(a)">
                    <AppIcon name="trash" :size="12" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Config Form Modal -->
    <AiConfigForm
      v-if="showForm"
      :initial="editingConfig"
      ref="formRef"
      @close="showForm = false"
      @saved="handleSaved"
      @connected="onConnected"
    />

    <!-- Assignment Form Modal -->
    <AiConfigAssignmentForm
      v-if="showAssignForm"
      ref="assignFormRef"
      @close="showAssignForm = false"
      @saved="handleAssignSaved"
    />

    <!-- Test result modal -->
    <Modal v-if="testResult || testError" :title="`Resultado do teste — ${testName}`" max-width="680px" @close="closeTest">
      <div v-if="testError" class="error-box">{{ testError }}</div>
      <template v-else-if="testResult">
        <p class="usage">
          {{ testResult.usage.TotalTokens }} tokens
          (entrada {{ testResult.usage.PromptTokens }} / saída {{ testResult.usage.CompletionTokens }})
        </p>
        <pre class="review">{{ testResult.review_markdown }}</pre>
      </template>
    </Modal>

    <!-- Delete confirm -->
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
import { ref, onMounted } from 'vue'
import SkeletonRows from '@/components/SkeletonRows.vue'
import AppIcon from '@/components/AppIcon.vue'
import AiConfigForm from '@/components/AiConfigForm.vue'
import AiConfigAssignmentForm from '@/components/AiConfigAssignmentForm.vue'
import Modal from '@/components/Modal.vue'
import ConfirmModal from '@/components/ConfirmModal.vue'
import { useToastStore } from '@/stores/toast'
import {
  listConfigs, createConfig, updateConfig, deleteConfig,
  listConfigAssignments, setConfigAssignment, removeConfigAssignment, testConfig,
  type AIConfig, type AIConfigInput, type AIConfigAssignment, type AIAssignmentInput, type TestResult,
} from '@/api/aiConfigs'

const toast = useToastStore()

const configs = ref<AIConfig[]>([])
const assignments = ref<AIConfigAssignment[]>([])
const loading = ref(false)
const deleting = ref(false)
const showForm = ref(false)
const showAssignForm = ref(false)
const editingConfig = ref<AIConfig | null>(null)
const deleteTarget = ref<AIConfig | null>(null)
const formRef = ref<InstanceType<typeof AiConfigForm> | null>(null)
const assignFormRef = ref<InstanceType<typeof AiConfigAssignmentForm> | null>(null)
const testingId = ref<number | null>(null)
const testResult = ref<TestResult | null>(null)
const testError = ref('')
const testName = ref('')

onMounted(load)

async function load() {
  loading.value = true
  try {
    const [cfg, a] = await Promise.all([listConfigs(), listConfigAssignments()])
    configs.value = cfg
    assignments.value = a
  } catch {
    toast.push('Erro ao carregar configurações', 'error')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingConfig.value = null
  showForm.value = true
}

function openEdit(c: AIConfig) {
  editingConfig.value = c
  showForm.value = true
}

async function handleSaved(data: AIConfigInput & { _id?: number }) {
  formRef.value?.setSaving(true)
  try {
    const { _id, ...payload } = data
    if (_id) {
      await updateConfig(_id, payload)
      toast.push('Configuração atualizada', 'success')
    } else {
      const created = await createConfig(payload)
      toast.push('Configuração criada', 'success')
      if (created.mode === 'account') {
        await load()
        editingConfig.value = configs.value.find((c) => c.id === created.id) ?? created
        formRef.value?.setSaving(false)
        return
      }
    }
    showForm.value = false
    await load()
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: string } })?.response?.data || 'Erro ao salvar'
    formRef.value?.setError(msg)
  }
}

async function onConnected() {
  toast.push('Conta Claude conectada', 'success')
  await load()
}

async function handleAssignSaved(data: AIAssignmentInput) {
  assignFormRef.value?.setSaving(true)
  try {
    await setConfigAssignment(data)
    toast.push('Atribuição salva', 'success')
    showAssignForm.value = false
    await load()
  } catch (e: unknown) {
    const msg = (e as { response?: { data?: string } })?.response?.data || 'Erro ao salvar'
    assignFormRef.value?.setError(msg)
  }
}

async function doDelete() {
  if (!deleteTarget.value) return
  deleting.value = true
  try {
    await deleteConfig(deleteTarget.value.id)
    toast.push(`Configuração "${deleteTarget.value.name}" excluída`, 'success')
    deleteTarget.value = null
    await load()
  } catch {
    toast.push('Erro ao excluir', 'error')
  } finally {
    deleting.value = false
  }
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
  } catch (e: unknown) {
    testError.value = (e as { response?: { data?: string } })?.response?.data || 'Falha no teste'
  } finally {
    testingId.value = null
  }
}

function closeTest() {
  testResult.value = null
  testError.value = ''
}

async function removeAssign(a: AIConfigAssignment) {
  try {
    await removeConfigAssignment(a.scope, a.client_name, a.project_ref)
    toast.push('Atribuição removida', 'success')
    await load()
  } catch {
    toast.push('Erro ao remover', 'error')
  }
}

function scopeLabel(scope: string) {
  if (scope === 'global') return 'Global'
  if (scope === 'client') return 'Cliente'
  return 'Projeto'
}

function modeLabel(mode: string) {
  return mode === 'account' ? 'Conta' : 'API'
}

function providerLabel(provider: string) {
  const map: Record<string, string> = {
    openai: 'OpenAI', claude: 'Claude', gemini: 'Gemini', deepseek: 'DeepSeek',
  }
  return map[provider] ?? provider
}
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
}

.empty-state {
  padding: 56px 32px;
  text-align: center;
}

.name {
  font-weight: 500;
  overflow-wrap: anywhere;
}

.col-scope {
  width: 110px;
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

.error-box {
  color: var(--danger);
  font-size: 13px;
  padding: 12px;
  background: color-mix(in srgb, var(--danger) 8%, transparent);
  border: 1px solid color-mix(in srgb, var(--danger) 30%, transparent);
  border-radius: var(--radius);
  margin: 0;
}
</style>