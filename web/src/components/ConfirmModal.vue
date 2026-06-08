<template>
  <Modal title="Confirmar exclusão" max-width="420px" @close="$emit('close')">
    <div class="confirm-body">
      <p class="confirm-text">Excluir <strong>{{ name }}</strong>?</p>
      <p v-if="warning" class="warn">{{ warning }}</p>
    </div>
    <template #footer>
      <button class="btn" type="button" @click="$emit('close')">Cancelar</button>
      <button class="btn danger" type="button" :disabled="loading" @click="$emit('confirm')">
        <span v-if="loading" class="spinner" />
        {{ loading ? 'Excluindo…' : 'Excluir' }}
      </button>
    </template>
  </Modal>
</template>

<script setup lang="ts">
import Modal from '@/components/Modal.vue'

defineProps<{
  name: string
  warning?: string
  loading?: boolean
}>()

defineEmits<{ close: []; confirm: [] }>()
</script>

<style scoped>
.confirm-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.confirm-text {
  margin: 0;
  font-size: 13.5px;
  color: var(--text);
}

.confirm-text strong {
  color: var(--text);
  font-weight: 500;
}

.warn {
  margin: 0;
  font-size: 12px;
  color: var(--warn);
  font-family: var(--font-mono);
}
</style>