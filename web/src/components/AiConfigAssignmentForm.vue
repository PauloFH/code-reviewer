<template>
  <Modal title="Nova atribuição" max-width="480px" @close="$emit('close')">
    <p v-if="error" class="error" role="alert">{{ error }}</p>
    <form @submit.prevent="submit" class="form">
      <div class="field">
        <label for="ai-assign-config">Configuração de IA</label>
        <select id="ai-assign-config" v-model.number="form.config_id" required>
          <option :value="0" disabled>Selecione uma configuração</option>
          <option v-for="c in configs" :key="c.id" :value="c.id">{{ c.name }} ({{ c.provider }})</option>
        </select>
      </div>
      <div class="field">
        <label for="ai-assign-scope">Escopo</label>
        <select id="ai-assign-scope" v-model="form.scope" required>
          <option value="global">Global (padrão para todos)</option>
          <option value="client">Cliente (host Git)</option>
          <option value="project">Projeto (repositório específico)</option>
        </select>
      </div>
      <div v-if="form.scope !== 'global'" class="field">
        <label for="ai-assign-client">Cliente</label>
        <select id="ai-assign-client" v-model="form.client_name" required>
          <option value="" disabled>Selecione um cliente</option>
          <option v-for="c in clients" :key="c.name" :value="c.name">{{ c.name }} ({{ c.platform }})</option>
        </select>
      </div>
      <div v-if="form.scope === 'project'" class="field">
        <label for="ai-assign-project">Projeto (referência)</label>
        <input id="ai-assign-project" v-model="form.project_ref" required placeholder="Ex: org/repo-name" />
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
import { ref, onMounted } from 'vue'
import Modal from '@/components/Modal.vue'
import { listConfigs, type AIConfig, type AIAssignmentInput } from '@/api/aiConfigs'
import { listClients, type Client } from '@/api/clients'

const emit = defineEmits<{
  close: []
  saved: [data: AIAssignmentInput]
}>()

const saving = ref(false)
const error = ref('')
const configs = ref<AIConfig[]>([])
const clients = ref<Client[]>([])

const form = ref<AIAssignmentInput>({
  config_id: 0,
  scope: 'global',
  client_name: '',
  project_ref: '',
})

onMounted(async () => {
  try {
    const [cfg, cl] = await Promise.all([listConfigs(), listClients()])
    configs.value = cfg
    clients.value = cl
  } catch {
    error.value = 'Erro ao carregar dados'
  }
})

function submit() {
  if (form.value.config_id === 0) {
    error.value = 'Selecione uma configuração'
    return
  }
  error.value = ''
  saving.value = true
  const data: AIAssignmentInput = { ...form.value }
  if (data.scope === 'global') {
    data.client_name = ''
    data.project_ref = ''
  } else if (data.scope === 'client') {
    data.project_ref = ''
  }
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

.field label {
  font-size: 12.5px;
  font-weight: 500;
  color: var(--muted);
}

.field input,
.field select {
  padding: 8px 12px;
  background: var(--bg-2);
  border: 1px solid var(--field-border);
  border-radius: var(--radius);
  color: var(--text);
  font-size: 13px;
  font-family: inherit;
  transition: border-color 0.1s;
}

.field input:focus,
.field select:focus {
  border-color: var(--accent);
}

.field input::placeholder {
  color: var(--muted-2);
}

.error {
  color: var(--danger);
  font-size: 12.5px;
  padding: 10px 12px;
  background: color-mix(in srgb, var(--danger) 8%, transparent);
  border: 1px solid color-mix(in srgb, var(--danger) 30%, transparent);
  border-radius: var(--radius);
  margin-bottom: 16px;
}
</style>