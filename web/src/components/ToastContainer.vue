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
  gap: .5rem;
  z-index: 9999;
}
.toast {
  padding: .75rem 1.25rem;
  border-radius: 8px;
  font-size: .875rem;
  cursor: pointer;
  max-width: 340px;
  box-shadow: 0 4px 16px rgba(0,0,0,.4);
}
.toast.success { background: #052e16; color: #4ade80; border: 1px solid #166534; }
.toast.error   { background: #450a0a; color: #fca5a5; border: 1px solid #7f1d1d; }
.toast.info    { background: #1e3a5f; color: #93c5fd; border: 1px solid #1e40af; }

.toast-enter-active, .toast-leave-active { transition: all .25s ease; }
.toast-enter-from { opacity: 0; transform: translateY(12px); }
.toast-leave-to   { opacity: 0; transform: translateX(100%); }
</style>
