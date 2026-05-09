export interface MonitorOutput {
  id: string
  name: string
  type: string
  target: string
  interval_sec: number
  timeout_ms: number
  config: Record<string, unknown>
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface MonitorBody {
  name: string
  type: string
  target: string
  interval_sec: number
  timeout_ms: number
  config: Record<string, unknown>
  enabled?: boolean
}

export interface MonitorListResponse {
  data: MonitorOutput[]
  total: number
  page: number
  per_page: number
  pages: number
}

export type CheckStatus = 'up' | 'down' | 'degraded'

export interface CheckOutput {
  id: number
  monitor_id: string
  started_at: string
  duration_ms: number
  status: CheckStatus
  status_code?: number
  error?: string
}

export interface CheckListResponse {
  data: CheckOutput[]
  total: number
}
