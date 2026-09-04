import { http } from '@/lib/http'

export interface AlertEvent {
  id: string
  service_id: string
  env_id: string
  alert_source: string
  alert_fingerprint: string
  severity: string
  title: string
  content: string
  status: string
  first_seen_at: string
  last_seen_at: string
  assignee_user_id: string
  created_at: string
}

export interface AlertStats {
  total_open: number
  total_acked: number
  total_closed: number
  p1_open: number
  p2_open: number
}

export interface AlertListResponse {
  list: AlertEvent[]
  total: number
  page: number
  page_size: number
}

export const alertApi = {
  list: (params: Record<string, string | number>) =>
    http.get<AlertListResponse>('/api/v1/alerts', params),

  stats: () => http.get<AlertStats>('/api/v1/alerts/stats'),

  get: (id: string) => http.get<AlertEvent>(`/api/v1/alerts/${id}`),

  create: (data: { service_id?: string; env_id?: string; severity: string; title: string; content?: string }) =>
    http.post<AlertEvent>('/api/v1/alerts', data),

  ack: (id: string, userId?: string) =>
    http.post(`/api/v1/alerts/${id}/ack`, { user_id: userId || 'system' }),

  close: (id: string) =>
    http.post(`/api/v1/alerts/${id}/close`, {}),
}
