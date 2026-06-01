<template>
  <div class="page">
    <!-- Configs Section -->
    <div class="section">
      <div class="header">
        <h2>Configurações de IA</h2>
        <button @click="openCreate">+ Nova Configuração</button>
      </div>

      <div v-if="loading" class="loading-row">
        <span class="spinner" /> Carregando…
      </div>
      <div v-else class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Nome</th>
              <th>Provedor</th>
              <th>Modo</th>
              <th>Credencial</th>
              <th>Ações</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="c in configs" :key="c.id">
              <td>{{ c.name }}</td>
              <td>{{ providerLabel(c.provider) }}</td>
              <td><span class="badge" :class="c.mode">{{ modeLabel(c.mode) }}</span></td>
              <td>
                <span v-if="c.mode === 'account'" :class="['cred', c.account_connected ? 'ok' : 'off']">
                  {{ c.account_connected ? 'Conectado' : 'Não conectado' }}
                </span>
                <span v-else :class="['cred', c.has_api_key ? 'ok' : 'off']">
                  {{ c.has_api_key ? 'Chave definida' : 'Sem chave' }}
                </span>
              </td>
              <td class="actions">
                <button :disabled="testingId === c.id" @click="doTest(c)">
                  <span v-if="testingId === c.id" class="spinner" />
                  {{ testingId === c.id ? 'Testando…' : 'Testar' }}
                </button>
                <button @click="openEdit(c)">Editar</button>
                <button class="danger" @click="deleteTarget = c">Excluir</button>
              </td>
            </tr>
            <tr v-if="configs.length === 0">
              <td colspan="5" class="empty">Nenhuma configuração cadastrada</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Assignments Section -->
    <div class="section">
      <div class="header">
        <h2>Atribuições</h2>
        <button @click="showAssignForm = true">+ Nova Atribuição</button>
      </div>

      <div class="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Escopo</th>
              <th>Client</th>
              <th>Projeto</th>
              <th>Configuração</th>
              <th>Ações</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="a in assignments" :key="a.id">
              <td><span class="badge" :class="a.scope">{{ scopeLabel(a.scope) }}</span></td>
              <td>{{ a.client_name || '—' }}</td>
              <td>{{ a.project_ref || '—' }}</td>
              <td>{{ a.config_name }}</td>
              <td class="actions">
                <button class="danger" @click="removeAssign(a)">Remover</button>
              </td>
            </tr>
            <tr v-if="assignments.length === 0">
              <td colspan="5" class="empty">Nenhuma atribuição configurada (usa o fallback do ambiente)</td>
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
    <div v-if="testResult || testError" class="overlay" @click.self="closeTest">
      <div class="test-modal">
        <div class="test-head">
          <h3>Resultado do teste — {{ testName }}</h3>
          <button class="close-x" @click="closeTest">✕</button>
        </div>
        <p v-if="testError" class="error">{{ testError }}</p>
        <template v-else-if="testResult">
          <p class="usage">
            Tokens: {{ testResult.usage.TotalTokens }}
            (entrada {{ testResult.usage.PromptTokens }} / saída {{ testResult.usage.CompletionTokens }})
          </p>
          <pre class="review">{{ testResult.review_markdown }}</pre>
        </template>
      </div>
    </div>

    <!-- Delete confirm -->
    <div v-if="deleteTarget" class="overlay" @click.self="deleteTarget = null">
      <div class="confirm-modal">
        <p>Excluir configuração <strong>{{ deleteTarget.name }}</strong>?</p>
        <p class="warn">Atribuições vinculadas serão removidas.</p>
        <div class="confirm-actions">
          <button @click="deleteTarget = null">Cancelar</button>
          <button class="danger" :disabled="deleting" @click="doDelete">
            <span v-if="deleting" class="spinner" />
            {{ deleting ? 'Excluindo…' : 'Excluir' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import AiConfigForm from '@/components/AiConfigForm.vue'
import AiConfigAssignmentForm from '@/components/AiConfigAssignmentForm.vue'
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
      // Account-mode configs need a follow-up OAuth connect; reopen in edit mode.
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
  if (scope === 'client') return 'Client'
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
.page { display: flex; flex-direction: column; gap: 2rem; }
.section { display: flex; flex-direction: column; gap: 1rem; }
h2 { font-size: 1.3rem; font-weight: 700; color: #f8fafc; }
.header { display: flex; align-items: center; justify-content: space-between; }
.header button {
  padding: .5rem 1rem; background: #6366f1; color: white;
  border: none; border-radius: 6px; cursor: pointer; font-size: .875rem;
}
.loading-row { display: flex; align-items: center; gap: .5rem; color: #64748b; }
.table-wrap { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; font-size: .875rem; }
th { text-align: left; padding: .6rem .75rem; color: #64748b; font-weight: 600; border-bottom: 1px solid #334155; white-space: nowrap; }
td { padding: .6rem .75rem; border-bottom: 1px solid #0f172a; color: #cbd5e1; }
.badge { font-size: .7rem; padding: .15rem .5rem; border-radius: 999px; }
.badge.global { background: #1e3a5f; color: #93c5fd; }
.badge.client { background: #1e3b1e; color: #86efac; }
.badge.project { background: #3b1e3b; color: #d8b4fe; }
.badge.api { background: #1e3a5f; color: #93c5fd; }
.badge.account { background: #3b2e1e; color: #fcd34d; }
.cred { font-size: .8rem; }
.cred.ok { color: #86efac; }
.cred.off { color: #64748b; }
.actions { display: flex; gap: .5rem; }
.actions button {
  padding: .3rem .6rem; border: 1px solid #334155; background: #1e293b;
  color: #e2e8f0; border-radius: 4px; cursor: pointer; font-size: .8rem;
}
.actions button.danger { color: #f87171; border-color: #7f1d1d; }
.empty { text-align: center; color: #475569; padding: 2rem; }
.overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,.6);
  display: flex; align-items: center; justify-content: center; z-index: 100;
}
.test-modal {
  background: #1e293b; border: 1px solid #334155; border-radius: 12px;
  padding: 1.5rem; max-width: 680px; width: 100%; max-height: 85vh;
  display: flex; flex-direction: column; gap: .75rem;
}
.test-head { display: flex; align-items: center; justify-content: space-between; }
.test-head h3 { font-size: 1.05rem; font-weight: 600; color: #f8fafc; }
.close-x { background: none; border: none; color: #64748b; cursor: pointer; font-size: 1rem; }
.usage { font-size: .8rem; color: #94a3b8; }
.review {
  background: #0f172a; border: 1px solid #334155; border-radius: 8px;
  padding: 1rem; color: #cbd5e1; font-size: .8rem; line-height: 1.5;
  white-space: pre-wrap; word-break: break-word; overflow-y: auto; margin: 0;
}
.error { color: #f87171; font-size: .85rem; }
.confirm-modal {
  background: #1e293b; border: 1px solid #334155; border-radius: 12px;
  padding: 1.5rem; max-width: 360px; width: 100%; text-align: center;
}
.confirm-modal p { color: #e2e8f0; margin-bottom: .5rem; }
.confirm-modal .warn { color: #fbbf24; font-size: .8rem; margin-bottom: 1rem; }
.confirm-actions { display: flex; justify-content: center; gap: .75rem; }
.confirm-actions button {
  padding: .5rem 1.1rem; border-radius: 6px; cursor: pointer; font-size: .875rem;
  border: 1px solid #334155; background: #334155; color: #e2e8f0;
  display: flex; align-items: center; gap: .4rem;
}
.confirm-actions button.danger { background: #7f1d1d; border-color: #7f1d1d; color: #fca5a5; }
.confirm-actions button:disabled { opacity: .6; cursor: not-allowed; }
.spinner {
  width: 12px; height: 12px;
  border: 2px solid rgba(255,255,255,.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin .6s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>
