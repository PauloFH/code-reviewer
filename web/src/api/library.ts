import { http } from './http'
import { assignmentApi } from './assignments'

// Recursos atribuíveis por escopo (prompts, perfis, modelos de IA): mesmo CRUD, só muda o prefixo.
export type LibraryPrefix = '/prompts' | '/ai-configs' | '/review-profiles'

export type LibraryApi<T, I> = ReturnType<typeof libraryApi<T, I>>

export function libraryApi<T, I>(prefix: LibraryPrefix) {
  return {
    async list(): Promise<T[]> {
      return (await http.get(prefix)).data
    },
    async create(data: I): Promise<T> {
      return (await http.post(prefix, data)).data
    },
    async update(id: number, data: I): Promise<T> {
      return (await http.put(`${prefix}/${id}`, data)).data
    },
    async remove(id: number): Promise<void> {
      await http.delete(`${prefix}/${id}`)
    },
    assignments: assignmentApi(prefix),
  }
}
