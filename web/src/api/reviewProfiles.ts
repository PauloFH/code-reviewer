import { libraryApi } from './library'

export interface ReviewProfile {
  id: number
  name: string
  include_related: boolean
  agentic: boolean
  execution: boolean
  max_iterations: number
  created_at: string
  updated_at: string
}

export type ReviewProfileInput = Pick<
  ReviewProfile,
  'name' | 'include_related' | 'agentic' | 'execution' | 'max_iterations'
>

export const reviewProfilesApi = libraryApi<ReviewProfile, ReviewProfileInput>('/review-profiles')
