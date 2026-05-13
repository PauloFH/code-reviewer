import { defineStore } from 'pinia'
import { ref } from 'vue'

export type ToastType = 'success' | 'error' | 'info'

export interface Toast {
  id: number
  message: string
  type: ToastType
}

let _id = 0

export const useToastStore = defineStore('toast', () => {
  const toasts = ref<Toast[]>([])

  function push(message: string, type: ToastType = 'info', ttl = 3500) {
    const id = ++_id
    toasts.value.push({ id, message, type })
    setTimeout(() => remove(id), ttl)
  }

  function remove(id: number) {
    toasts.value = toasts.value.filter((t) => t.id !== id)
  }

  return { toasts, push, remove }
})
