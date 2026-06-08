import { http } from './http'

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
  status: 'success' | 'skipped' | 'error'
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
  since?: string
  until?: string
  limit?: number
  offset?: number
}

export async function listEvents(filter: EventFilter = {}): Promise<EventsResponse> {
  const res = await http.get('/events', { params: filter })
  return res.data
}

export async function getStats(client?: string, since?: string, until?: string): Promise<ReviewStats> {
  const res = await http.get('/events/stats', { params: { client, since, until } })
  return res.data
}

export async function getDailyCosts(client?: string, days = 30): Promise<DailyCostPoint[]> {
  const res = await http.get('/events/daily', { params: { days, ...(client ? { client } : {}) } })
  return res.data
}

export function createEventSource(token: string): EventSource {
  return new EventSource(`/api/v1/events/stream?token=${encodeURIComponent(token)}`)
}
