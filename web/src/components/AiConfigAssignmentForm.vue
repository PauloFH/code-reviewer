<template>
  <div class="overlay" @click.self="$emit('close')">
    <div class="modal">
      <h3>Nova Atribuição</h3>
      <p v-if="error" class="error">{{ error }}</p>
      <form @submit.prevent="submit">
        <label>
          Configuração de IA
          <select v-model.number="form.config_id" required>
            <option :value="0" disabled>Selecione uma configuração</option>
            <option v-for="c in configs" :key="c.id" :value="c.id">{{ c.name }} ({{ c.provider }})</option>
          </select>
        </label>
        <label>
          Escopo
          <select v-model="form.scope" required>
            <option value="global">Global (padrão para todos)</option>
            <option value="client">Client (provedor Git)</option>
            <option value="project">Projeto (repositório específico)</option>
          </select>
        </label>
        <label v-if="form.scope !== 'global'">
          Client
          <select v-model="form.client_name" required>
            <option value="" disabled>Selecione um client</option>
            <option v-for="c in clients" :key="c.name" :value="c.name">{{ c.name }} ({{ c.platform }})</option>
          </select>
        </label>
        <label v-if="form.scope === 'project'">
          Projeto (project ref)
          <input v-model="form.project_ref" required placeholder="Ex: org/repo-name" />
        </label>
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
import { ref, onMounted } from 'vue'
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
.overlay {
  position: fixed; inset: 0;
  background: rgba(0,0,0,.6);
  display: flex; align-items: center; justify-content: center;
  z-index: 100;
}
.modal {
  background: #1e293b; border: 1px solid #334155; border-radius: 12px;
  padding: 1.5rem; width: 100%; max-width: 500px;
}
h3 { font-size: 1.1rem; font-weight: 600; color: #f8fafc; margin-bottom: 1rem; }
form { display: flex; flex-direction: column; gap: .75rem; }
label { display: flex; flex-direction: column; gap: .3rem; font-size: .8rem; color: #94a3b8; }
input, select {
  padding: .5rem .75rem; background: #0f172a; border: 1px solid #475569;
  border-radius: 6px; color: #e2e8f0; font-size: .875rem;
}
input:focus, select:focus { outline: none; border-color: #6366f1; }
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
