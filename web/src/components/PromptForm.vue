<template>
  <div class="overlay" @click.self="$emit('close')">
    <div class="modal">
      <h3>{{ isEdit ? 'Editar Prompt' : 'Novo Prompt' }}</h3>
      <p v-if="error" class="error">{{ error }}</p>
      <form @submit.prevent="submit">
        <label>
          Nome
          <input v-model="form.name" required placeholder="Ex: Review Backend Go" />
        </label>
        <label>
          System Prompt
          <RichTextEditor v-model="form.system_prompt" placeholder="Instruções para a IA..." />
        </label>
        <label>
          Template de Saída
          <RichTextEditor v-model="form.output_template" placeholder="Formato esperado do review..." />
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
import { ref } from 'vue'
import RichTextEditor from '@/components/RichTextEditor.vue'
import type { Prompt, PromptInput } from '@/api/prompts'

const props = defineProps<{
  initial?: Prompt | null
}>()

const emit = defineEmits<{
  close: []
  saved: [data: PromptInput & { _id?: number }]
}>()

const isEdit = !!props.initial
const saving = ref(false)
const error = ref('')

const form = ref<PromptInput>({
  name: props.initial?.name || '',
  system_prompt: props.initial?.system_prompt || '',
  output_template: props.initial?.output_template || '',
})

function submit() {
  error.value = ''
  saving.value = true
  emit('saved', { ...form.value, _id: props.initial?.id })
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
  display: flex; align-items: flex-start; justify-content: center;
  z-index: 100;
  padding-top: 5vh;
  overflow-y: auto;
}
.modal {
  background: #1e293b; border: 1px solid #334155; border-radius: 12px;
  padding: 1.5rem; width: 100%; max-width: 700px; margin-bottom: 5vh;
}
h3 { font-size: 1.1rem; font-weight: 600; color: #f8fafc; margin-bottom: 1rem; }
form { display: flex; flex-direction: column; gap: 1rem; }
label { display: flex; flex-direction: column; gap: .4rem; font-size: .8rem; color: #94a3b8; }
input {
  padding: .5rem .75rem; background: #0f172a; border: 1px solid #475569;
  border-radius: 6px; color: #e2e8f0; font-size: .875rem;
}
input:focus { outline: none; border-color: #6366f1; }
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
