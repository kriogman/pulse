import type {
  MonitorBody,
  MonitorOutput,
  MonitorListResponse,
  CheckListResponse,
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
    list: (id: string) => request<CheckListResponse>(`/monitors/${id}/checks`),
  },
}
