<template>
  <FormModal :title="initial ? 'Editar prompt' : 'Novo prompt'" max-width="700px" :saving="saving" :error="error" @close="$emit('close')" @submit="submit(() => save(form))">
    <div class="field">
      <label for="prompt-name">Nome</label>
      <input id="prompt-name" v-model="form.name" required placeholder="Ex: Revisão de backend Go" />
    </div>
    <div class="field">
      <label>Prompt de sistema</label>
      <RichTextEditor v-model="form.system_prompt" label="Prompt de sistema" placeholder="Instruções para a IA…" />
    </div>
    <div class="field">
      <label>Template de saída</label>
      <RichTextEditor v-model="form.output_template" label="Template de saída" placeholder="Formato esperado da revisão…" />
    </div>
  </FormModal>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import FormModal from '@/components/FormModal.vue'
import RichTextEditor from '@/components/RichTextEditor.vue'
import { useSubmit } from '@/composables/useSubmit'
import type { Prompt, PromptInput } from '@/api/prompts'

const props = defineProps<{
  initial: Prompt | null
  save: (data: PromptInput) => Promise<void>
}>()
defineEmits<{ close: [] }>()

const { saving, error, submit } = useSubmit()
const form = ref<PromptInput>({
  name: props.initial?.name ?? '',
  system_prompt: props.initial?.system_prompt ?? '',
  output_template: props.initial?.output_template ?? '',
})
</script>
