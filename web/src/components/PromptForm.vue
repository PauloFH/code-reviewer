<template>
  <Modal :title="isEdit ? 'Editar Prompt' : 'Novo Prompt'" max-width="700px" @close="$emit('close')">
    <p v-if="error" class="error">{{ error }}</p>
    <form @submit.prevent="submit" class="form">
      <div class="field">
        <label>Nome</label>
        <input v-model="form.name" required placeholder="Ex: Review Backend Go" />
      </div>
      <div class="field field-tall">
        <label>System Prompt</label>
        <RichTextEditor v-model="form.system_prompt" placeholder="Instruções para a IA..." />
      </div>
      <div class="field field-tall">
        <label>Template de Saída</label>
        <RichTextEditor v-model="form.output_template" placeholder="Formato esperado do review..." />
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
import { ref } from 'vue'
import Modal from '@/components/Modal.vue'
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

.field-tall {
  flex: 1;
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