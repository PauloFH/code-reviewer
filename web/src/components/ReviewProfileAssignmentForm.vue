<template>
  <Modal title="Nova atribuição" max-width="480px" @close="$emit('close')">
    <p v-if="error" class="error" role="alert">{{ error }}</p>
    <form @submit.prevent="submit" class="form">
      <div class="field">
        <label for="profile-assign-profile">Perfil</label>
        <select id="profile-assign-profile" v-model.number="form.profile_id" required>
          <option :value="0" disabled>Selecione um perfil</option>
          <option v-for="p in profiles" :key="p.id" :value="p.id">{{ p.name }}</option>
        </select>
      </div>
      <div class="field">
        <label for="profile-assign-scope">Escopo</label>
        <select id="profile-assign-scope" v-model="form.scope" required>
          <option value="global">Global (padrão para todos)</option>
          <option value="client">Cliente (host Git)</option>
          <option value="project">Projeto (repositório específico)</option>
        </select>
      </div>
      <div v-if="form.scope !== 'global'" class="field">
        <label for="profile-assign-client">Cliente</label>
        <select id="profile-assign-client" v-model="form.client_name" required>
          <option value="" disabled>Selecione um cliente</option>
          <option v-for="c in clients" :key="c.name" :value="c.name">{{ c.name }} ({{ c.platform }})</option>
        </select>
      </div>
      <div v-if="form.scope === 'project'" class="field">
        <label for="profile-assign-project">Projeto (referência)</label>
        <input id="profile-assign-project" v-model="form.project_ref" required placeholder="Ex: org/repo-name" />
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
import type { ReviewProfile, AssignmentInput } from '@/api/reviewProfiles'
import { listClients, type Client } from '@/api/clients'

defineProps<{
  profiles: ReviewProfile[]
}>()

const emit = defineEmits<{
  close: []
  saved: [data: AssignmentInput]
}>()

const saving = ref(false)
const error = ref('')
const clients = ref<Client[]>([])

const form = ref<AssignmentInput>({
  profile_id: 0,
  scope: 'global',
  client_name: '',
  project_ref: '',
})

onMounted(async () => {
  try {
    clients.value = await listClients()
  } catch {
    error.value = 'Erro ao carregar clientes'
  }
})

function submit() {
  if (form.value.profile_id === 0) {
    error.value = 'Selecione um perfil'
    return
  }
  error.value = ''
  saving.value = true
  const data: AssignmentInput = { ...form.value }
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
