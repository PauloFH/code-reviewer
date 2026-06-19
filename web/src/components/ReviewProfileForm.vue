<template>
  <Modal :title="isEdit ? 'Editar Perfil de Revisão' : 'Novo Perfil de Revisão'" max-width="500px" @close="$emit('close')">
    <p v-if="error" class="error">{{ error }}</p>
    <form @submit.prevent="submit" class="form">
      <div class="field">
        <label>Nome</label>
        <input v-model="form.name" required placeholder="Ex: Revisão Completa" />
      </div>
      <div class="field">
        <label>Camadas de revisão</label>
        <div class="checkboxes">
          <label class="checkbox-row">
            <input type="checkbox" v-model="form.include_related" />
            <span>Incluir contexto relacionado (issues/MRs/discussão)</span>
          </label>
          <label class="checkbox-row">
            <input type="checkbox" v-model="form.agentic" />
            <span>Revisão agêntica (checkout + navegação)</span>
          </label>
          <label class="checkbox-row">
            <input type="checkbox" v-model="form.execution" />
            <span>Execução em sandbox (Fase 2)</span>
          </label>
        </div>
      </div>
      <div class="field">
        <label>Máximo de iterações agênticas (0 = padrão)</label>
        <input type="number" v-model.number="form.max_iterations" min="0" placeholder="0" />
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
import type { ReviewProfile, ReviewProfileInput } from '@/api/reviewProfiles'

const props = defineProps<{
  initial?: ReviewProfile | null
}>()

const emit = defineEmits<{
  close: []
  saved: [data: ReviewProfileInput & { _id?: number }]
}>()

const isEdit = !!props.initial
const saving = ref(false)
const error = ref('')

const form = ref<ReviewProfileInput>({
  name: props.initial?.name ?? '',
  include_related: props.initial?.include_related ?? false,
  agentic: props.initial?.agentic ?? false,
  execution: props.initial?.execution ?? false,
  max_iterations: props.initial?.max_iterations ?? 0,
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

.field label {
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 500;
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.field input[type="text"],
.field input[type="number"],
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

.field input[type="text"]:focus,
.field input[type="number"]:focus,
.field select:focus {
  outline: none;
  border-color: var(--accent);
}

.field input::placeholder {
  color: var(--muted-2);
}

.checkboxes {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  background: var(--bg-2);
  border: 1px solid var(--hairline);
  border-radius: var(--radius);
}

.checkbox-row {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  font-size: 13px;
  color: var(--text);
  font-family: inherit;
  text-transform: none;
  letter-spacing: normal;
  font-weight: normal;
}

.checkbox-row input[type="checkbox"] {
  width: 14px;
  height: 14px;
  cursor: pointer;
  accent-color: var(--accent);
  flex-shrink: 0;
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
