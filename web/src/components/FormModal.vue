<template>
  <Modal :title="title" :max-width="maxWidth" @close="$emit('close')">
    <form :id="id" class="form" @submit.prevent="$emit('submit')">
      <p v-if="error" class="error" role="alert">{{ error }}</p>
      <slot />
    </form>
    <template #footer>
      <button class="btn" type="button" @click="$emit('close')">Cancelar</button>
      <!-- O botão fica no rodapé, fora do <form>: `form` liga os dois para o navegador validar `required`. -->
      <button class="btn primary" type="submit" :form="id" :disabled="saving">
        <span v-if="saving" class="spinner" />
        {{ saving ? 'Salvando…' : 'Salvar' }}
      </button>
    </template>
  </Modal>
</template>

<script setup lang="ts">
import { useId } from 'vue'
import Modal from '@/components/Modal.vue'

defineProps<{ title: string; maxWidth?: string; saving: boolean; error: string }>()
defineEmits<{ close: []; submit: [] }>()

const id = useId()
</script>
