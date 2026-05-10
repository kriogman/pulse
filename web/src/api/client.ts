import type {
  MonitorBody,
  MonitorOutput,
  MonitorListResponse,
  CheckListResponse,
  MonitorStats,
} from '../types'

const BASE = '/api/v1'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    ...init,
  })
  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText)
    throw new Error(`${res.status}: ${text}`)
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

export const api = {
  monitors: {
    list: () => request<MonitorListResponse>('/monitors'),
    get: (id: string) => request<MonitorOutput>(`/monitors/${id}`),
    create: (data: MonitorBody) =>
      request<MonitorOutput>('/monitors', {
        method: 'POST',
        body: JSON.stringify(data),
      }),
    update: (id: string, data: MonitorBody) =>
      request<MonitorOutput>(`/monitors/${id}`, {
        method: 'PUT',
        body: JSON.stringify(data),
      }),
    delete: (id: string) => request<void>(`/monitors/${id}`, { method: 'DELETE' }),
    pause: (id: string) =>
      request<MonitorOutput>(`/monitors/${id}/pause`, { method: 'POST' }),
    resume: (id: string) =>
      request<MonitorOutput>(`/monitors/${id}/resume`, { method: 'POST' }),
  },
  checks: {
    list: (id: string, params?: { from?: string; limit?: number }) => {
      const q = new URLSearchParams()
      if (params?.from) q.set('from', params.from)
      if (params?.limit) q.set('limit', String(params.limit))
      const qs = q.toString()
      return request<CheckListResponse>(`/monitors/${id}/checks${qs ? `?${qs}` : ''}`)
    },
    stats: (id: string, period = '24h') =>
      request<MonitorStats>(`/monitors/${id}/stats?period=${period}`),
  },
}
