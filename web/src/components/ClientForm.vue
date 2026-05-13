<template>
  <div class="overlay" @click.self="$emit('close')">
    <div class="modal">
      <h3>{{ isEdit ? 'Edit Client' : 'New Client' }}</h3>
      <p v-if="error" class="error">{{ error }}</p>
      <form @submit.prevent="submit">
        <label>
          Name
          <input v-model="form.name" required :disabled="isEdit" />
        </label>
        <label>
          Platform
          <select v-model="form.platform" required>
            <option value="gitlab">GitLab</option>
            <option value="gitlab-selfhosted">GitLab Self-hosted</option>
            <option value="github">GitHub</option>
            <option value="gitea">Gitea</option>
            <option value="gitea-selfhosted">Gitea Self-hosted</option>
          </select>
        </label>
        <label>
          Base URL
          <input v-model="form.base_url" placeholder="https://gitlab.com" required />
        </label>
        <label>
          API Token
          <input v-model="form.token" type="password" :placeholder="isEdit ? 'leave blank to keep' : ''" />
        </label>
        <label>
          Webhook Secret
          <input v-model="form.webhook_secret" type="password" :placeholder="isEdit ? 'leave blank to keep' : ''" />
        </label>
        <label>
          Watch Username
          <input v-model="form.username" required />
        </label>
        <div class="actions">
          <button type="button" @click="$emit('close')">Cancel</button>
          <button type="submit" :disabled="saving">
            <span v-if="saving" class="spinner" />
            {{ saving ? 'Saving…' : 'Save' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
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

const emptyForm = (): ClientInput => ({
  name: '', platform: 'gitlab', base_url: '', token: '', webhook_secret: '', username: '',
})

const form = ref<ClientInput>(
  props.initial
    ? { name: props.initial.name, platform: props.initial.platform, base_url: props.initial.base_url, token: '', webhook_secret: '', username: props.initial.username }
    : emptyForm(),
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

// Allow parent to signal save completion / error
defineExpose({ setSaving: (v: boolean) => { saving.value = v }, setError: (e: string) => { error.value = e } })
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
  padding: 1.5rem; width: 100%; max-width: 480px;
}
h3 { font-size: 1.1rem; font-weight: 600; color: #f8fafc; margin-bottom: 1rem; }
form { display: flex; flex-direction: column; gap: .75rem; }
label { display: flex; flex-direction: column; gap: .3rem; font-size: .8rem; color: #94a3b8; }
input, select {
  padding: .5rem .75rem; background: #0f172a; border: 1px solid #475569;
  border-radius: 6px; color: #e2e8f0; font-size: .875rem;
}
input:focus, select:focus { outline: none; border-color: #6366f1; }
input:disabled { opacity: .5; cursor: not-allowed; }
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
