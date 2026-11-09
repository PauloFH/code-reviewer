<template>
  <Modal :title="isEdit ? 'Editar cliente' : 'Novo cliente'" max-width="480px" @close="$emit('close')">
    <p v-if="error" class="error" role="alert">{{ error }}</p>
    <form @submit.prevent="submit" class="form">
      <div class="field">
        <label for="client-name">Nome</label>
        <input id="client-name" v-model="form.name" required :disabled="isEdit" />
      </div>
      <div class="field">
        <label for="client-platform">Plataforma</label>
        <select id="client-platform" v-model="form.platform" required>
          <option value="gitlab">GitLab</option>
          <option value="gitlab-selfhosted">GitLab auto-hospedado</option>
          <option value="github">GitHub</option>
          <option value="gitea">Gitea</option>
          <option value="gitea-selfhosted">Gitea auto-hospedado</option>
        </select>
      </div>
      <div class="field">
        <label for="client-base-url">URL base</label>
        <input id="client-base-url" v-model="form.base_url" placeholder="https://gitlab.com" required />
      </div>
      <div class="field">
        <label for="client-token">Token de API</label>
        <input id="client-token" v-model="form.token" type="password" :placeholder="isEdit ? 'deixe em branco para manter' : ''" />
      </div>
      <div class="field">
        <label for="client-secret">Segredo do webhook</label>
        <input id="client-secret" v-model="form.webhook_secret" type="password" :placeholder="isEdit ? 'deixe em branco para manter' : ''" />
      </div>
      <div class="field">
        <label for="client-username">Usuário monitorado</label>
        <input id="client-username" v-model="form.username" required />
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
import { ref, watch } from 'vue'
import Modal from '@/components/Modal.vue'
import type { Client, ClientInput } from '@/api/clients'

const props = defineProps<{
  initial?: Client | null
}>()

const emit = defineEmits<{
  close: []
  saved: [data: ClientInput & { _oldName?: string }]
}>()

const isEdit = !!props.initial
const saving = ref(false)
const error = ref('')

const form = ref<ClientInput>(
  props.initial
    ? { name: props.initial.name, platform: props.initial.platform, base_url: props.initial.base_url, token: '', webhook_secret: '', username: props.initial.username }
    : { name: '', platform: 'gitlab', base_url: '', token: '', webhook_secret: '', username: '' },
)

watch(() => props.initial, (v) => {
  if (v) {
    form.value = { name: v.name, platform: v.platform, base_url: v.base_url, token: '', webhook_secret: '', username: v.username }
  }
})

async function submit() {
  error.value = ''
  saving.value = true
  try {
    emit('saved', { ...form.value, _oldName: props.initial?.name })
  } catch (e: unknown) {
    error.value = String(e)
    saving.value = false
  }
}

defineExpose({ setSaving: (v: boolean) => { saving.value = v }, setError: (e: string) => { error.value = e; saving.value = false } })
</script>

<style scoped>
.form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.field label {
  font-size: 13px;
  color: var(--muted);
}

.field input,
.field select {
  padding: 10px 12px;
  background: var(--bg-2);
  border: 1px solid var(--field-border);
  border-radius: var(--radius);
  color: var(--text);
  font-size: 14px;
  font-family: var(--font-sans);
  transition: border-color 0.12s;
}

.field input:focus,
.field select:focus {
  border-color: var(--accent);
}

.field input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.field input::placeholder {
  color: var(--muted-2);
}

.error {
  color: var(--danger);
  font-size: 13px;
  padding: 8px 12px;
  background: var(--bg-2);
  border: 1px solid var(--danger);
  border-radius: var(--radius);
  margin-bottom: 16px;
  overflow-wrap: anywhere;
}
</style>
