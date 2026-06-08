<template>
  <Modal :title="isEdit ? 'Edit Client' : 'New Client'" max-width="480px" @close="$emit('close')">
    <p v-if="error" class="error">{{ error }}</p>
    <form @submit.prevent="submit" class="form">
      <div class="field">
        <label>Name</label>
        <input v-model="form.name" required :disabled="isEdit" />
      </div>
      <div class="field">
        <label>Platform</label>
        <select v-model="form.platform" required>
          <option value="gitlab">GitLab</option>
          <option value="gitlab-selfhosted">GitLab Self-hosted</option>
          <option value="github">GitHub</option>
          <option value="gitea">Gitea</option>
          <option value="gitea-selfhosted">Gitea Self-hosted</option>
        </select>
      </div>
      <div class="field">
        <label>Base URL</label>
        <input v-model="form.base_url" placeholder="https://gitlab.com" required />
      </div>
      <div class="field">
        <label>API Token</label>
        <input v-model="form.token" type="password" :placeholder="isEdit ? 'leave blank to keep' : ''" />
      </div>
      <div class="field">
        <label>Webhook Secret</label>
        <input v-model="form.webhook_secret" type="password" :placeholder="isEdit ? 'leave blank to keep' : ''" />
      </div>
      <div class="field">
        <label>Watch Username</label>
        <input v-model="form.username" required />
      </div>
    </form>
    <template #footer>
      <button class="btn" type="button" @click="$emit('close')">Cancel</button>
      <button class="btn primary" type="submit" :disabled="saving" @click="submit">
        <span v-if="saving" class="spinner" />
        {{ saving ? 'Saving…' : 'Save' }}
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

.field input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.field input::placeholder {
  color: var(--muted-2);
}

.error {
  color: var(--danger);
  font-size: 12.5px;
  padding: 10px 12px;
  background: rgba(248, 113, 113, 0.08);
  border: 1px solid rgba(248, 113, 113, 0.2);
  border-radius: var(--radius);
  margin-bottom: 16px;
}
</style>