import { ref, onMounted } from 'vue'
import { listClients, type Client } from '@/api/clients'
import { apiErrorMessage } from '@/api/http'
import { useAuthStore } from '@/stores/auth'
import { useToastStore } from '@/stores/toast'

// GET /clients é só de gestor/super: para os demais a lista fica vazia, sem requisição nem erro.
export function useClients() {
  const clients = ref<Client[]>([])
  const auth = useAuthStore()
  const toast = useToastStore()
  onMounted(async () => {
    await auth.initPromise
    if (!auth.can('canManageResources')) return
    try {
      clients.value = await listClients()
    } catch (e) {
      toast.push(apiErrorMessage(e, 'Erro ao carregar clientes'), 'error')
    }
  })
  return clients
}
