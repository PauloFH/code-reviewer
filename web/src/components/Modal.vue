<template>
  <Teleport to="body">
    <div class="modal-overlay" @click.self="$emit('close')">
      <div class="modal" role="dialog" aria-modal="true" :aria-label="title" :style="{ maxWidth: maxWidth }">
        <div class="modal-header">
          <h2 class="modal-title">{{ title }}</h2>
          <button class="icon-btn" type="button" title="Fechar" aria-label="Fechar" @click="$emit('close')">
            <AppIcon name="close" :size="14" />
          </button>
        </div>
        <div class="modal-body">
          <slot />
        </div>
        <div v-if="$slots.footer" class="modal-footer">
          <slot name="footer" />
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import AppIcon from '@/components/AppIcon.vue'

defineProps<{
  title: string
  maxWidth?: string
}>()

defineEmits<{ close: [] }>()
</script>

<style scoped>
/* A cortina continua translúcida; o conteúdo do modal é sólido. */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.65);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 200;
  padding: 24px 16px;
}

.modal {
  background: var(--panel-solid);
  border: 1px solid var(--hairline);
  border-radius: var(--radius-lg);
  width: 100%;
  max-height: calc(100vh - 48px);
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-soft);
  animation: modal-in 0.15s ease;
}

@keyframes modal-in {
  from { opacity: 0; transform: translateY(-8px); }
  to { opacity: 1; transform: translateY(0); }
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 18px 22px;
  border-bottom: 1px solid var(--hairline);
  flex-shrink: 0;
}

.modal-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--text);
  margin: 0;
  letter-spacing: -0.02em;
  overflow-wrap: anywhere;
}

.modal-body {
  padding: 22px;
  overflow-y: auto;
  flex: 1;
}

.modal-footer {
  padding: 16px 22px;
  border-top: 1px solid var(--hairline);
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  flex-shrink: 0;
}
</style>
