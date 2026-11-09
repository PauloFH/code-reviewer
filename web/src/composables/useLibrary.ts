import { ref, onMounted, type Ref } from 'vue'
import type { LibraryApi } from '@/api/library'
import type { Assignment } from '@/api/assignments'
import { apiErrorMessage } from '@/api/http'
import { useToastStore } from '@/stores/toast'

interface Messages {
  loadError: string
  created: string
  updated: string
  deleted: (name: string) => string
}

// Estado e ações de uma página de biblioteca: lista + atribuições, modal de edição e exclusão.
export function useLibrary<T extends { id: number; name: string }, I>(
  api: LibraryApi<T, I>,
  msg: Messages,
  // Após criar, devolve true para manter o modal aberto editando o item criado.
  keepOpenAfterCreate?: (created: T) => boolean,
) {
  const toast = useToastStore()
  const items = ref([]) as Ref<T[]>
  const assignments = ref<Assignment[]>([])
  const loading = ref(false)
  const deleting = ref(false)
  const showForm = ref(false)
  const editing = ref(null) as Ref<T | null>
  const deleteTarget = ref(null) as Ref<T | null>

  async function load() {
    loading.value = true
    try {
      const [list, a] = await Promise.all([api.list(), api.assignments.list()])
      items.value = list
      assignments.value = a
    } catch (e) {
      toast.push(apiErrorMessage(e, msg.loadError), 'error')
    } finally {
      loading.value = false
    }
  }

  function openForm(item: T | null) {
    editing.value = item
    showForm.value = true
  }

  async function save(data: I) {
    const current = editing.value
    const saved = current ? await api.update(current.id, data) : await api.create(data)
    toast.push(current ? msg.updated : msg.created, 'success')
    if (!current && keepOpenAfterCreate?.(saved)) {
      await load()
      editing.value = items.value.find((i) => i.id === saved.id) ?? saved
      return
    }
    showForm.value = false
    await load()
  }

  async function doDelete() {
    const target = deleteTarget.value
    if (!target) return
    deleting.value = true
    try {
      await api.remove(target.id)
      toast.push(msg.deleted(target.name), 'success')
      deleteTarget.value = null
      await load()
    } catch (e) {
      toast.push(apiErrorMessage(e, 'Erro ao excluir'), 'error')
    } finally {
      deleting.value = false
    }
  }

  onMounted(load)

  return { items, assignments, loading, deleting, showForm, editing, deleteTarget, load, openForm, save, doDelete }
}
