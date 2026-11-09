<template>
  <FormModal :title="initial ? 'Editar configuração' : 'Nova configuração de IA'" max-width="520px" :saving="saving" :error="error" @close="$emit('close')" @submit="onSubmit">
    <div class="field">
      <label for="ai-name">Nome</label>
      <input id="ai-name" v-model="form.name" required placeholder="Ex: Claude produção" />
    </div>

    <div class="field-row">
      <div class="field">
        <label for="ai-provider">Provedor</label>
        <select id="ai-provider" v-model="form.provider" required @change="onProviderChange">
          <option v-for="p in providers" :key="p.id" :value="p.id">{{ p.label }}</option>
        </select>
      </div>
      <div class="field">
        <label for="ai-mode">Modo</label>
        <select id="ai-mode" v-model="form.mode" :disabled="!provider?.supports_account">
          <option v-for="m in AI_MODES" :key="m.value" :value="m.value">{{ m.label }} ({{ m.hint }})</option>
        </select>
      </div>
    </div>

    <p v-if="!provider?.supports_account && accountLabels" class="hint">O modo "Conta" só está disponível para {{ accountLabels }}.</p>

    <div v-if="form.mode === 'api'" class="field">
      <label for="ai-api-key">Chave de API</label>
      <input
        id="ai-api-key"
        v-model="form.api_key"
        type="password"
        :placeholder="initial?.has_api_key ? '•••••• (deixe vazio para manter)' : 'Cole a chave de API'"
      />
    </div>

    <div class="field">
      <label for="ai-model">Modelo <span class="opt">(opcional)</span></label>
      <input id="ai-model" v-model="form.model" :placeholder="provider?.default_model" />
    </div>

    <div class="field">
      <label for="ai-base-url">Base URL <span class="opt">(opcional)</span></label>
      <input id="ai-base-url" v-model="form.base_url" :placeholder="provider?.default_base_url" />
    </div>

    <div class="field-row">
      <div class="field">
        <label for="ai-price-in">Preço / M tokens (entrada)</label>
        <input id="ai-price-in" v-model.number="form.price_per_m_input_tokens" type="number" step="0.01" min="0" />
      </div>
      <div class="field">
        <label for="ai-price-out">Preço / M tokens (saída)</label>
        <input id="ai-price-out" v-model.number="form.price_per_m_output_tokens" type="number" step="0.01" min="0" />
      </div>
    </div>

    <div class="field-row">
      <div class="field">
        <label for="ai-max-tokens">Máx. de tokens de saída <span class="opt">(0 = padrão)</span></label>
        <input id="ai-max-tokens" v-model.number="form.max_tokens" type="number" min="0" placeholder="0" />
      </div>
      <div class="field">
        <label for="ai-timeout">Timeout (segundos) <span class="opt">(0 = padrão)</span></label>
        <input id="ai-timeout" v-model.number="form.timeout_seconds" type="number" min="0" placeholder="0" />
      </div>
    </div>

    <p class="hint">{{ limitsHint }}</p>

    <div v-if="form.mode === 'account'" class="oauth-box">
      <div v-if="!initial" class="hint">Salve a configuração primeiro para conectar a conta Claude.</div>
      <template v-else>
        <div class="oauth-status">
          <span :class="['dot', connected ? 'ok' : 'off']" />
          {{ connected ? `Conta conectada${expiresLabel}` : 'Conta não conectada' }}
        </div>
        <p v-if="oauthError" class="error" role="alert">{{ oauthError }}</p>
        <button type="button" class="btn" :disabled="oauthBusy" @click="startConnect">
          <span v-if="oauthBusy" class="spinner" />
          {{ connected ? 'Reconectar conta Claude' : 'Conectar conta Claude' }}
        </button>
        <div v-if="awaitingCode" class="code-step">
          <p class="hint">Autorize na aba aberta e cole o código retornado abaixo.</p>
          <!-- Enter aqui conclui a conexão; sem o prevent, enviaria o formulário e o código se perderia. -->
          <input
            v-model="pastedCode"
            aria-label="Código de autorização"
            placeholder="Cole o código de autorização"
            @keydown.enter.prevent="finishConnect"
          />
          <button type="button" class="btn primary" :disabled="oauthBusy || !pastedCode" @click="finishConnect">
            Concluir conexão
          </button>
        </div>
      </template>
    </div>
  </FormModal>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import FormModal from '@/components/FormModal.vue'
import { useSubmit } from '@/composables/useSubmit'
import { apiErrorMessage } from '@/api/http'
import { AI_MODES } from '@/domain'
import {
  startClaudeOAuth, exchangeClaudeOAuth,
  type AIConfig, type AIConfigInput, type AIProviderInfo,
} from '@/api/aiConfigs'

const props = defineProps<{
  initial: AIConfig | null
  providers: AIProviderInfo[]
  save: (data: AIConfigInput) => Promise<void>
}>()
const emit = defineEmits<{ close: []; connected: [] }>()

const { saving, error, submit } = useSubmit()

const form = ref<AIConfigInput>({
  name: props.initial?.name ?? '',
  provider: props.initial?.provider ?? props.providers[0]?.id ?? '',
  mode: props.initial?.mode ?? 'api',
  base_url: props.initial?.base_url ?? '',
  model: props.initial?.model ?? '',
  api_key: '',
  price_per_m_input_tokens: props.initial?.price_per_m_input_tokens ?? 0,
  price_per_m_output_tokens: props.initial?.price_per_m_output_tokens ?? 0,
  max_tokens: props.initial?.max_tokens ?? 0,
  timeout_seconds: props.initial?.timeout_seconds ?? 0,
})

const provider = computed(() => props.providers.find((p) => p.id === form.value.provider))
// 0 grava "usar o padrão do servidor": mudar o default no backend alcança também as configs já salvas.
const limitsHint = computed(() => {
  const n = provider.value?.default_max_tokens
  const tokens = n ? `${provider.value?.label}: ${n.toLocaleString('pt-BR')} tokens` : 'o provedor decide'
  return `0 usa o padrão: tokens de saída (${tokens}); timeout de 120 s.`
})
const accountLabels = computed(() => props.providers.filter((p) => p.supports_account).map((p) => p.label).join(', '))

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

function onProviderChange() {
  if (!provider.value?.supports_account) form.value.mode = 'api'
}

async function startConnect() {
  oauthError.value = ''
  oauthBusy.value = true
  try {
    const { authorize_url, state } = await startClaudeOAuth()
    oauthState.value = state
    window.open(authorize_url, '_blank', 'noopener')
    awaitingCode.value = true
  } catch (e) {
    oauthError.value = apiErrorMessage(e, 'Não foi possível iniciar a conexão')
  } finally {
    oauthBusy.value = false
  }
}

async function finishConnect() {
  if (!props.initial || !pastedCode.value || oauthBusy.value) return
  oauthError.value = ''
  oauthBusy.value = true
  try {
    await exchangeClaudeOAuth(props.initial.id, oauthState.value, pastedCode.value.trim())
    connected.value = true
    awaitingCode.value = false
    pastedCode.value = ''
    emit('connected')
  } catch (e) {
    oauthError.value = apiErrorMessage(e, 'Falha ao conectar conta')
  } finally {
    oauthBusy.value = false
  }
}

function onSubmit() {
  const data: AIConfigInput = { ...form.value }
  if (!data.api_key) delete data.api_key
  submit(() => props.save(data))
}
</script>

<style scoped>
.field-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.opt {
  font-weight: 400;
}

.hint {
  font-size: 11.5px;
  color: var(--muted);
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
.dot.off { background: var(--skip); }

.code-step {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
</style>
