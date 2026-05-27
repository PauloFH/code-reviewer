<template>
  <div class="overlay" @click.self="$emit('close')">
    <div class="modal">
      <h3>{{ initial ? 'Editar Configuração' : 'Nova Configuração de IA' }}</h3>
      <p v-if="error" class="error">{{ error }}</p>
      <form @submit.prevent="submit">
        <label>
          Nome
          <input v-model="form.name" required placeholder="Ex: Claude produção" />
        </label>

        <div class="row">
          <label>
            Provedor
            <select v-model="form.provider" @change="onProviderChange">
              <option value="openai">OpenAI</option>
              <option value="claude">Claude (Anthropic)</option>
              <option value="gemini">Gemini (Google)</option>
              <option value="deepseek">DeepSeek</option>
            </select>
          </label>
          <label>
            Modo
            <select v-model="form.mode" :disabled="form.provider !== 'claude'">
              <option value="api">API (chave)</option>
              <option value="account">Conta (login)</option>
            </select>
          </label>
        </div>
        <p v-if="form.provider !== 'claude'" class="hint">
          O modo "Conta" só está disponível para o Claude.
        </p>

        <label v-if="form.mode === 'api'">
          Chave de API
          <input
            v-model="form.api_key"
            type="password"
            :placeholder="initial?.has_api_key ? '•••••• (deixe vazio para manter)' : 'Cole a chave de API'"
          />
        </label>

        <label>
          Modelo <span class="opt">(opcional)</span>
          <input v-model="form.model" :placeholder="modelPlaceholder" />
        </label>
        <label>
          Base URL <span class="opt">(opcional)</span>
          <input v-model="form.base_url" :placeholder="baseURLPlaceholder" />
        </label>

        <div class="row">
          <label>
            Preço / M tokens (entrada)
            <input v-model.number="form.price_per_m_input_tokens" type="number" step="0.01" min="0" />
          </label>
          <label>
            Preço / M tokens (saída)
            <input v-model.number="form.price_per_m_output_tokens" type="number" step="0.01" min="0" />
          </label>
        </div>
        <div class="row">
          <label>
            Máximo de tokens (saída) <span class="opt">(0 = padrão)</span>
            <input v-model.number="form.max_tokens" type="number" min="0" placeholder="0" />
          </label>
          <label>
            Timeout (segundos)
            <input v-model.number="form.timeout_seconds" type="number" min="1" />
          </label>
        </div>
        <p class="hint">0 usa o padrão do provedor (Claude usa 4096; OpenAI/Gemini/DeepSeek usam o default deles).</p>

        <!-- OAuth connect (account mode, existing config only) -->
        <div v-if="form.mode === 'account'" class="oauth-box">
          <div v-if="!initial" class="hint">
            Salve a configuração primeiro para conectar a conta Claude.
          </div>
          <template v-else>
            <div class="oauth-status">
              <span :class="['dot', connected ? 'ok' : 'off']" />
              {{ connected ? `Conta conectada${expiresLabel}` : 'Conta não conectada' }}
            </div>
            <p v-if="oauthError" class="error">{{ oauthError }}</p>
            <button type="button" class="oauth-btn" :disabled="oauthBusy" @click="startConnect">
              <span v-if="oauthBusy" class="spinner" />
              {{ connected ? 'Reconectar conta Claude' : 'Conectar conta Claude' }}
            </button>
            <div v-if="awaitingCode" class="code-step">
              <p class="hint">
                Autorize na aba aberta e cole o código retornado abaixo.
              </p>
              <input v-model="pastedCode" placeholder="Cole o código de autorização" />
              <button type="button" class="oauth-btn" :disabled="oauthBusy || !pastedCode" @click="finishConnect">
                Concluir conexão
              </button>
            </div>
          </template>
        </div>

        <div class="actions">
          <button type="button" @click="$emit('close')">Cancelar</button>
          <button type="submit" :disabled="saving">
            <span v-if="saving" class="spinner" />
            {{ saving ? 'Salvando…' : 'Salvar' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import {
  startClaudeOAuth, exchangeClaudeOAuth,
  type AIConfig, type AIConfigInput, type AIProvider,
} from '@/api/aiConfigs'

const props = defineProps<{ initial?: AIConfig | null }>()
const emit = defineEmits<{
  close: []
  saved: [data: AIConfigInput & { _id?: number }]
  connected: []
}>()

const saving = ref(false)
const error = ref('')

const form = ref<AIConfigInput>({
  name: props.initial?.name ?? '',
  provider: props.initial?.provider ?? 'claude',
  mode: props.initial?.mode ?? 'api',
  base_url: props.initial?.base_url ?? '',
  model: props.initial?.model ?? '',
  api_key: '',
  price_per_m_input_tokens: props.initial?.price_per_m_input_tokens ?? 0,
  price_per_m_output_tokens: props.initial?.price_per_m_output_tokens ?? 0,
  max_tokens: props.initial?.max_tokens ?? 0,
  timeout_seconds: props.initial?.timeout_seconds ?? 120,
})

// ── OAuth connect state ──────────────────────────────────────────────────────
const connected = ref(props.initial?.account_connected ?? false)
const oauthBusy = ref(false)
const oauthError = ref('')
const awaitingCode = ref(false)
const pastedCode = ref('')
const oauthState = ref('')

const expiresLabel = computed(() => {
  if (!props.initial?.oauth_expires_at) return ''
  const d = new Date(props.initial.oauth_expires_at)
  if (isNaN(d.getTime()) || d.getFullYear() < 2000) return ''
  return ` (expira ${d.toLocaleString('pt-BR')})`
})

const modelPlaceholder = computed(() => {
  switch (form.value.provider) {
    case 'gemini': return 'gemini-flash-latest'
    case 'claude': return 'claude-sonnet-4-5'
    case 'openai': return 'gpt-4o'
    case 'deepseek': return 'deepseek-chat'
    default: return ''
  }
})

const baseURLPlaceholder = computed(() => {
  switch (form.value.provider) {
    case 'gemini': return 'https://generativelanguage.googleapis.com/v1beta'
    case 'claude': return 'https://api.anthropic.com'
    case 'openai': return 'https://api.openai.com/v1'
    case 'deepseek': return 'https://api.deepseek.com/v1'
    default: return ''
  }
})

function onProviderChange() {
  if (form.value.provider !== 'claude' && form.value.mode === 'account') {
    form.value.mode = 'api'
  }
}

async function startConnect() {
  oauthError.value = ''
  oauthBusy.value = true
  try {
    const { authorize_url, state } = await startClaudeOAuth()
    oauthState.value = state
    window.open(authorize_url, '_blank', 'noopener')
    awaitingCode.value = true
  } catch {
    oauthError.value = 'Não foi possível iniciar a conexão'
  } finally {
    oauthBusy.value = false
  }
}

async function finishConnect() {
  if (!props.initial || !pastedCode.value) return
  oauthError.value = ''
  oauthBusy.value = true
  try {
    await exchangeClaudeOAuth(props.initial.id, oauthState.value, pastedCode.value.trim())
    connected.value = true
    awaitingCode.value = false
    pastedCode.value = ''
    emit('connected')
  } catch (e: unknown) {
    oauthError.value = (e as { response?: { data?: string } })?.response?.data || 'Falha ao conectar conta'
  } finally {
    oauthBusy.value = false
  }
}

function submit() {
  error.value = ''
  if (form.value.provider !== ('claude' as AIProvider) && form.value.mode === 'account') {
    error.value = 'O modo Conta só é suportado para o Claude'
    return
  }
  saving.value = true
  const data: AIConfigInput & { _id?: number } = { ...form.value }
  if (!data.api_key) delete data.api_key
  if (props.initial) data._id = props.initial.id
  emit('saved', data)
}

defineExpose({
  setSaving: (v: boolean) => { saving.value = v },
  setError: (e: string) => { error.value = e; saving.value = false },
})
</script>

<style scoped>
.overlay {
  position: fixed; inset: 0;
  background: rgba(0,0,0,.6);
  display: flex; align-items: center; justify-content: center;
  z-index: 100; padding: 1rem;
}
.modal {
  background: #1e293b; border: 1px solid #334155; border-radius: 12px;
  padding: 1.5rem; width: 100%; max-width: 520px;
  max-height: 90vh; overflow-y: auto;
}
h3 { font-size: 1.1rem; font-weight: 600; color: #f8fafc; margin-bottom: 1rem; }
form { display: flex; flex-direction: column; gap: .75rem; }
.row { display: flex; gap: .75rem; }
.row label { flex: 1; }
label { display: flex; flex-direction: column; gap: .3rem; font-size: .8rem; color: #94a3b8; }
.opt { color: #64748b; font-weight: 400; }
input, select {
  padding: .5rem .75rem; background: #0f172a; border: 1px solid #475569;
  border-radius: 6px; color: #e2e8f0; font-size: .875rem;
}
input:focus, select:focus { outline: none; border-color: #6366f1; }
select:disabled { opacity: .6; cursor: not-allowed; }
.hint { font-size: .75rem; color: #64748b; }
.oauth-box {
  border: 1px solid #334155; border-radius: 8px; padding: .75rem;
  display: flex; flex-direction: column; gap: .5rem; background: #0f172a;
}
.oauth-status { display: flex; align-items: center; gap: .5rem; font-size: .8rem; color: #cbd5e1; }
.dot { width: 8px; height: 8px; border-radius: 50%; }
.dot.ok { background: #22c55e; }
.dot.off { background: #64748b; }
.oauth-btn {
  padding: .45rem .8rem; border-radius: 6px; cursor: pointer; font-size: .8rem;
  border: 1px solid #6366f1; background: #312e81; color: #c7d2fe;
  display: flex; align-items: center; gap: .4rem; align-self: flex-start;
}
.oauth-btn:disabled { opacity: .6; cursor: not-allowed; }
.code-step { display: flex; flex-direction: column; gap: .5rem; }
.actions { display: flex; justify-content: flex-end; gap: .75rem; margin-top: .5rem; }
.actions button {
  display: flex; align-items: center; gap: .4rem;
  padding: .5rem 1.1rem; border-radius: 6px; cursor: pointer; font-size: .875rem;
  border: 1px solid #334155; background: #334155; color: #e2e8f0;
}
.actions button[type="submit"] { background: #6366f1; border-color: #6366f1; color: white; }
.actions button:disabled { opacity: .6; cursor: not-allowed; }
.spinner {
  width: 12px; height: 12px;
  border: 2px solid rgba(255,255,255,.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin .6s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
.error { color: #f87171; font-size: .8rem; margin-bottom: .25rem; }
</style>
