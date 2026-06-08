<template>
  <Teleport to="body">
    <div class="toast-wrap">
      <TransitionGroup name="toast">
        <div
          v-for="t in toasts.toasts"
          :key="t.id"
          :class="['toast', t.type]"
          @click="toasts.remove(t.id)"
        >
          {{ t.message }}
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { useToastStore } from '@/stores/toast'
const toasts = useToastStore()
</script>

<style scoped>
.toast-wrap {
  position: fixed;
  bottom: 1.5rem;
  right: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  z-index: 9999;
}
.toast {
  padding: 0.75rem 1.25rem;
  border-radius: var(--radius);
  font-size: 13px;
  cursor: pointer;
  max-width: 340px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
}
.toast.success {
  background: rgba(52, 211, 153, 0.12);
  color: var(--ok);
  border: 1px solid rgba(52, 211, 153, 0.25);
}
.toast.error {
  background: rgba(248, 113, 113, 0.12);
  color: var(--danger);
  border: 1px solid rgba(248, 113, 113, 0.25);
}
.toast.info {
  background: rgba(103, 232, 249, 0.12);
  color: var(--info);
  border: 1px solid rgba(103, 232, 249, 0.25);
}

.toast-enter-active, .toast-leave-active { transition: all 0.25s ease; }
.toast-enter-from { opacity: 0; transform: translateY(12px); }
.toast-leave-to   { opacity: 0; transform: translateX(100%); }
</style>
