<template>
  <Teleport to="body">
    <div class="modal-overlay" @click.self="$emit('close')">
      <div class="modal" :style="{ maxWidth: maxWidth }">
        <div class="modal-header">
          <h2 class="modal-title">{{ title }}</h2>
          <button class="icon-btn" @click="$emit('close')">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">
              <path d="M18 6L6 18M6 6l12 12"/>
            </svg>
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
defineProps<{
  title: string
  maxWidth?: string
}>()

defineEmits<{ close: [] }>()
</script>

<style scoped>
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
  background: var(--panel);
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
  from { opacity: 0; transform: translateY(-8px) scale(0.98); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 18px 22px;
  border-bottom: 1px solid var(--hairline);
  flex-shrink: 0;
}

.modal-title {
  font-size: 15px;
  font-weight: 500;
  color: var(--text);
  margin: 0;
  letter-spacing: -0.01em;
}

.icon-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: transparent;
  color: var(--muted);
  border-radius: var(--radius);
  cursor: pointer;
  transition: color 0.1s, background 0.1s;
}

.icon-btn:hover {
  color: var(--text);
  background: var(--panel-2);
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