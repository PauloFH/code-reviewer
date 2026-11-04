<template>
  <Modal title="Confirmar exclusão" max-width="420px" @close="$emit('close')">
    <div class="confirm-body">
      <p class="confirm-text">Excluir <strong>{{ name }}</strong>?</p>
      <p v-if="warning" class="warn">{{ warning }}</p>
    </div>
    <template #footer>
      <button class="btn" type="button" @click="$emit('close')">Cancelar</button>
      <button class="btn danger-solid" type="button" :disabled="loading" @click="$emit('confirm')">
        <span v-if="loading" class="spinner" />
        <AppIcon v-else name="trash" :size="13" />
        {{ loading ? 'Excluindo…' : 'Excluir' }}
      </button>
    </template>
  </Modal>
</template>

<script setup lang="ts">
import Modal from '@/components/Modal.vue'
import AppIcon from '@/components/AppIcon.vue'

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
  overflow-wrap: anywhere;
}

.confirm-text {
  margin: 0;
  font-size: 14px;
  color: var(--text);
}

.confirm-text strong {
  font-weight: 600;
}

.warn {
  margin: 0;
  font-size: 13px;
  color: var(--warn);
}

/* Texto e borda em --danger sobre fundo sólido: AA nos dois temas (fundo --danger com --ink falharia no claro). */
.btn.danger-solid,
.btn.danger-solid svg {
  color: var(--danger);
}

.btn.danger-solid {
  border-color: var(--danger);
}

.btn.danger-solid:hover {
  background: var(--panel-2);
  border-color: var(--danger);
}

.btn.danger-solid:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
