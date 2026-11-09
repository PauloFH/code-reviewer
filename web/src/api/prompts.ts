import { libraryApi } from './library'

export interface Prompt {
  id: number
  name: string
  system_prompt: string
  output_template: string
  created_at: string
  updated_at: string
}

export type PromptInput = Pick<Prompt, 'name' | 'system_prompt' | 'output_template'>

export const promptsApi = libraryApi<Prompt, PromptInput>('/prompts')
