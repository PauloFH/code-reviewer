import { http, API_BASE } from './http'
import type { EventStatus } from '@/domain'

export interface ReviewEvent {
  id: number
  client_name: string
  project_ref: string
  mr_iid: number
  mr_id: number
  mr_title: string
  mr_url: string
  author_username: string
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  cost_usd: number
  duration_ms: number
  status: EventStatus
  error_message: string
  reviewed_at: string
}

export interface ReviewStats {
  total_reviews: number
  total_tokens: number
  total_cost_usd: number
  reviews_last_30d: number
  cost_last_30d: number
  by_client: Array<{
    client_name: string
    review_count: number
    total_tokens: number
    total_cost_usd: number
  }>
}

export interface DailyCostPoint {
  date: string
  cost_usd: number
  reviews: number
}

export interface EventsResponse {
  items: ReviewEvent[]
  total: number
}

export interface EventFilter {
  client?: string
  search?: string
  status?: EventStatus
  since?: string
  until?: string
  limit?: number
  offset?: number
}

export async function listEvents(filter: EventFilter = {}): Promise<EventsResponse> {
  const res = await http.get('/events', { params: filter })
  return res.data
}

export async function getStats(client?: string): Promise<ReviewStats> {
  const res = await http.get('/events/stats', { params: { client } })
  return res.data
}

export async function getDailyCosts(client: string | undefined, days: number): Promise<DailyCostPoint[]> {
  const res = await http.get('/events/daily', { params: { days, ...(client ? { client } : {}) } })
  return res.data
}

// EventSource não envia headers: o token vai na query string.
export function createEventSource(token: string): EventSource {
  return new EventSource(`${API_BASE}/events/stream?token=${encodeURIComponent(token)}`)
}
