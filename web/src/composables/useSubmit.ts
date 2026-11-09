import { ref } from 'vue'
import { apiErrorMessage } from '@/api/http'

// Estado de envio de um formulário: o form aguarda a ação e mostra o erro da API ele mesmo.
export function useSubmit() {
  const saving = ref(false)
  const error = ref('')

  async function submit(action: () => Promise<unknown>) {
    error.value = ''
    saving.value = true
    try {
      await action()
    } catch (e) {
      error.value = apiErrorMessage(e, 'Erro ao salvar')
    } finally {
      saving.value = false
    }
  }

  return { saving, error, submit }
}
