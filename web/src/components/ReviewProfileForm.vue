<template>
  <FormModal :title="initial ? 'Editar perfil de revisão' : 'Novo perfil de revisão'" max-width="500px" :saving="saving" :error="error" @close="$emit('close')" @submit="submit(() => save(form))">
    <div class="field">
      <label for="profile-name">Nome</label>
      <input id="profile-name" v-model="form.name" required placeholder="Ex: Revisão Completa" />
    </div>
    <fieldset class="field group">
      <legend>Camadas de revisão</legend>
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
    </fieldset>
    <div class="field">
      <label for="profile-max-iter">Máximo de iterações agênticas (0 = padrão)</label>
      <input id="profile-max-iter" type="number" v-model.number="form.max_iterations" min="0" placeholder="0" />
    </div>
  </FormModal>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import FormModal from '@/components/FormModal.vue'
import { useSubmit } from '@/composables/useSubmit'
import type { ReviewProfile, ReviewProfileInput } from '@/api/reviewProfiles'

const props = defineProps<{
  initial: ReviewProfile | null
  save: (data: ReviewProfileInput) => Promise<void>
}>()
defineEmits<{ close: [] }>()

const { saving, error, submit } = useSubmit()
const form = ref<ReviewProfileInput>({
  name: props.initial?.name ?? '',
  include_related: props.initial?.include_related ?? false,
  agentic: props.initial?.agentic ?? false,
  execution: props.initial?.execution ?? false,
  max_iterations: props.initial?.max_iterations ?? 0,
})
</script>

<style scoped>
.group {
  border: 0;
  min-width: 0;
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
}

.form .checkbox-row {
  font-size: 13px;
  font-weight: normal;
  color: var(--text);
}

.checkbox-row input {
  width: 14px;
  height: 14px;
  cursor: pointer;
  flex-shrink: 0;
}
</style>
