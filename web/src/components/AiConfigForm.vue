<template>
  <Modal :title="initial ? 'Editar Configuração' : 'Nova Configuração de IA'" max-width="520px" @close="$emit('close')">
    <p v-if="error" class="error">{{ error }}</p>
    <form @submit.prevent="submit" class="form">
      <div class="field">
        <label>Nome</label>
        <input v-model="form.name" required placeholder="Ex: Claude produção" />
      </div>

      <div class="field-row">
        <div class="field">
          <label>Provedor</label>
          <select v-model="form.provider" @change="onProviderChange">
            <option value="openai">OpenAI</option>
            <option value="claude">Claude (Anthropic)</option>
            <option value="gemini">Gemini (Google)</option>
            <option value="deepseek">DeepSeek</option>
          </select>
        </div>
        <div class="field">
          <label>Modo</label>
          <select v-model="form.mode" :disabled="form.provider !== 'claude'">
            <option value="api">API (chave)</option>
            <option value="account">Conta (login)</option>
          </select>
        </div>
      </div>

      <p v-if="form.provider !== 'claude'" class="hint">O modo "Conta" só está disponível para o Claude.</p>

      <div v-if="form.mode === 'api'" class="field">
        <label>Chave de API</label>
        <input
          v-model="form.api_key"
          type="password"
          :placeholder="initial?.has_api_key ? '•••••• (deixe vazio para manter)' : 'Cole a chave de API'"
        />
      </div>

      <div class="field">
        <label>Modelo <span class="opt">(opcional)</span></label>
        <input v-model="form.model" :placeholder="modelPlaceholder" />
      </div>

      <div class="field">
        <label>Base URL <span class="opt">(opcional)</span></label>
        <input v-model="form.base_url" :placeholder="baseURLPlaceholder" />
      </div>

      <div class="field-row">
        <div class="field">
          <label>Preço / M tokens (entrada)</label>
          <input v-model.number="form.price_per_m_input_tokens" type="number" step="0.01" min="0" />
        </div>
        <div class="field">
          <label>Preço / M tokens (saída)</label>
          <input v-model.number="form.price_per_m_output_tokens" type="number" step="0.01" min="0" />
        </div>
      </div>

      <div class="field-row">
        <div class="field">
          <label>Máx tokens saída <span class="opt">(0 = padrão)</span></label>
          <input v-model.number="form.max_tokens" type="number" min="0" placeholder="0" />
        </div>
        <div class="field">
          <label>Timeout (seg)</label>
          <input v-model.number="form.timeout_seconds" type="number" min="1" />
        </div>
      </div>

      <p class="hint">0 usa o padrão do provedor (Claude: 4096).</p>

      <div v-if="form.mode === 'account'" class="oauth-box">
        <div v-if="!initial" class="hint">Salve a configuração primeiro para conectar a conta Claude.</div>
        <template v-else>
          <div class="oauth-status">
            <span :class="['dot', connected ? 'ok' : 'off']" />
            {{ connected ? `Conta conectada${expiresLabel}` : 'Conta não conectada' }}
          </div>
          <p v-if="oauthError" class="error oauth-error">{{ oauthError }}</p>
          <button type="button" class="btn" :disabled="oauthBusy" @click="startConnect">
            <span v-if="oauthBusy" class="spinner" />
            {{ connected ? 'Reconectar conta Claude' : 'Conectar conta Claude' }}
          </button>
          <div v-if="awaitingCode" class="code-step">
            <p class="hint">Autorize na aba aberta e cole o código retornado abaixo.</p>
            <input v-model="pastedCode" placeholder="Cole o código de autorização" class="code-input" />
            <button type="button" class="btn primary" :disabled="oauthBusy || !pastedCode" @click="finishConnect">
              Concluir conexão
            </button>
          </div>
        </template>
      </div>
    </form>
    <template #footer>
      <button class="btn" type="button" @click="$emit('close')">Cancelar</button>
      <button class="btn primary" type="submit" :disabled="saving" @click="submit">
        <span v-if="saving" class="spinner" />
        {{ saving ? 'Salvando…' : 'Salvar' }}
      </button>
    </template>
  </Modal>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import Modal from '@/components/Modal.vue'
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
.form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.field label {
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 500;
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.field input,
.field select {
  padding: 8px 12px;
  background: var(--bg-2);
  border: 1px solid var(--hairline);
  border-radius: var(--radius);
  color: var(--text);
  font-size: 13px;
  font-family: inherit;
  transition: border-color 0.1s;
}

.field input:focus,
.field select:focus {
  outline: none;
  border-color: var(--accent);
}

.field input::placeholder {
  color: var(--muted-2);
}

select:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.opt {
  color: var(--muted-2);
  font-weight: 400;
  text-transform: none;
  letter-spacing: 0;
}

.hint {
  font-size: 11.5px;
  color: var(--muted);
  margin: 0;
}

.oauth-box {
  border: 1px solid var(--hairline);
  border-radius: var(--radius);
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: var(--bg-2);
}

.oauth-status {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12.5px;
  color: var(--text-2);
  font-family: var(--font-mono);
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.dot.ok { background: var(--ok); }
.dot.off { background: var(--muted); }

.oauth-error {
  margin: 0;
}

.code-step {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.code-input {
  width: 100%;
}

.error {
  color: var(--danger);
  font-size: 12.5px;
  padding: 10px 12px;
  background: rgba(248, 113, 113, 0.08);
  border: 1px solid rgba(248, 113, 113, 0.2);
  border-radius: var(--radius);
}
</style>