<template>
  <Teleport to="body">
    <div class="toast-wrap" role="status" aria-live="polite">
      <TransitionGroup name="toast">
        <div
          v-for="t in toasts.toasts"
          :key="t.id"
          :class="['toast', t.type]"
          @click="toasts.remove(t.id)"
        >
          <!-- Tipo também por sinal, nunca só por cor. -->
          <span class="sign" aria-hidden="true">{{ SIGNS[t.type] }}</span>
          <span class="msg">{{ t.message }}</span>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { useToastStore, type ToastType } from '@/stores/toast'
const toasts = useToastStore()

const SIGNS: Record<ToastType, string> = { success: '+', error: '−', info: '~' }
</script>

<style scoped>
.toast-wrap {
  position: fixed;
  bottom: 1.5rem;
  right: 1.5rem;
  left: 1rem;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.5rem;
  z-index: 9999;
  pointer-events: none;
}
.toast {
  --tone: var(--info);
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 0.7rem 1rem;
  background: var(--panel-solid);
  color: var(--text);
  border: 1px solid var(--hairline-strong);
  border-left: 3px solid var(--tone);
  border-radius: var(--radius);
  font-size: 13px;
  cursor: pointer;
  max-width: 340px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
  pointer-events: auto;
}
.toast.success { --tone: var(--ok); }
.toast.error { --tone: var(--danger); }
.toast .sign {
  color: var(--tone);
  font-family: var(--font-mono);
  font-weight: 500;
  font-size: 15px;
  line-height: 1.3;
  flex-shrink: 0;
}
.toast .msg {
  overflow-wrap: anywhere;
  min-width: 0;
}

.toast-enter-active, .toast-leave-active { transition: all 0.25s ease; }
.toast-enter-from { opacity: 0; transform: translateY(12px); }
.toast-leave-to   { opacity: 0; transform: translateX(100%); }
</style>
