import { http } from './http'

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

export interface ReviewProfileAssignment {
  id: number
  profile_id: number
  profile_name: string
  scope: 'global' | 'client' | 'project'
  client_name: string
  project_ref: string
}

export interface AssignmentInput {
  profile_id: number
  scope: 'global' | 'client' | 'project'
  client_name: string
  project_ref: string
}

export async function listReviewProfiles(): Promise<ReviewProfile[]> {
  const res = await http.get('/review-profiles')
  return res.data
}

export async function getReviewProfile(id: number): Promise<ReviewProfile> {
  const res = await http.get(`/review-profiles/${id}`)
  return res.data
}

export async function createReviewProfile(data: ReviewProfileInput): Promise<ReviewProfile> {
  const res = await http.post('/review-profiles', data)
  return res.data
}

export async function updateReviewProfile(id: number, data: ReviewProfileInput): Promise<ReviewProfile> {
  const res = await http.put(`/review-profiles/${id}`, data)
  return res.data
}

export async function deleteReviewProfile(id: number): Promise<void> {
  await http.delete(`/review-profiles/${id}`)
}

export async function listAssignments(): Promise<ReviewProfileAssignment[]> {
  const res = await http.get('/review-profiles/assignments')
  return res.data
}

export async function setAssignment(data: AssignmentInput): Promise<void> {
  await http.put('/review-profiles/assignments', data)
}

export async function removeAssignment(scope: string, clientName: string, projectRef: string): Promise<void> {
  await http.delete('/review-profiles/assignments', { params: { scope, client_name: clientName, project_ref: projectRef } })
}
