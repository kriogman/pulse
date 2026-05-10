import { useState, useMemo } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'
import StatusBadge from '../components/StatusBadge'
import UptimeStats from '../components/UptimeStats'
import ResponseTimeChart from '../components/ResponseTimeChart'

type Period = '1h' | '24h' | '7d' | '30d'

const PERIOD_LABEL: Record<Period, string> = {
  '1h': '1 hora',
  '24h': '24 horas',
  '7d': '7 días',
  '30d': '30 días',
}

function periodFrom(p: Period): string {
  const ms: Record<Period, number> = {
    '1h': 60 * 60 * 1000,
    '24h': 24 * 60 * 60 * 1000,
    '7d': 7 * 24 * 60 * 60 * 1000,
    '30d': 30 * 24 * 60 * 60 * 1000,
  }
  return new Date(Date.now() - ms[p]).toISOString()
}

function fmt(iso: string) {
  return new Date(iso).toLocaleString('es-ES', {
    dateStyle: 'short',
    timeStyle: 'medium',
  })
}

export default function CheckHistoryPage() {
  const { id } = useParams<{ id: string }>()
  const [period, setPeriod] = useState<Period>('24h')

  const from = useMemo(() => periodFrom(period), [period])
  const limit = period === '7d' || period === '30d' ? 500 : 100

  const monitorQuery = useQuery({
    queryKey: ['monitors', id],
    queryFn: () => api.monitors.get(id!),
    enabled: !!id,
  })

  const checksQuery = useQuery({
    queryKey: ['checks', id, period],
    queryFn: () => api.checks.list(id!, { from, limit }),
    enabled: !!id,
    refetchInterval: 15_000,
  })

  const statsQuery = useQuery({
    queryKey: ['stats', id, period],
    queryFn: () => api.checks.stats(id!, period),
    enabled: !!id,
    refetchInterval: 30_000,
  })

  const monitor = monitorQuery.data
  const checks = checksQuery.data?.data ?? []

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="border-b bg-white shadow-sm">
        <div className="mx-auto flex max-w-7xl items-center gap-4 px-4 py-4 sm:px-6">
          <Link to="/" className="text-sm text-gray-500 hover:text-gray-900">
            ← Volver
          </Link>
          <div>
            <span className="text-xl font-bold text-blue-600">Pulse</span>
            {monitor && (
              <span className="ml-2 text-sm text-gray-600">
                — Historial de{' '}
                <span className="font-medium text-gray-900">{monitor.name}</span>
              </span>
            )}
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-7xl space-y-5 px-4 py-8 sm:px-6">
        {/* Meta del monitor */}
        {monitor && (
          <div className="flex flex-wrap items-center gap-4 text-sm text-gray-600">
            <span>
              <span className="font-medium">URL:</span>{' '}
              <a href={monitor.target} target="_blank" rel="noreferrer" className="text-blue-600 hover:underline">
                {monitor.target}
              </a>
            </span>
            <span><span className="font-medium">Intervalo:</span> {monitor.interval_sec}s</span>
            <span><span className="font-medium">Timeout:</span> {monitor.timeout_ms}ms</span>
          </div>
        )}

        {/* Selector de período */}
        <div className="flex items-center gap-1">
          {(Object.keys(PERIOD_LABEL) as Period[]).map(p => (
            <button
              key={p}
              onClick={() => setPeriod(p)}
              className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                period === p
                  ? 'bg-blue-600 text-white'
                  : 'bg-white text-gray-600 border hover:bg-gray-50'
              }`}
            >
              {PERIOD_LABEL[p]}
            </button>
          ))}
        </div>

        {/* Stats cards */}
        {statsQuery.data && <UptimeStats stats={statsQuery.data} />}
        {statsQuery.isLoading && (
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="h-20 animate-pulse rounded-lg border bg-gray-100" />
            ))}
          </div>
        )}

        {/* Gráfica de tiempos de respuesta */}
        {checks.length > 0 && (
          <ResponseTimeChart checks={checks} avgDurationMs={statsQuery.data?.avg_duration_ms} />
        )}

        {/* Tabla de checks */}
        {checksQuery.isLoading && (
          <p className="text-center text-sm text-gray-500">Cargando checks…</p>
        )}
        {checksQuery.isError && (
          <p className="text-center text-sm text-red-600">Error al cargar el historial.</p>
        )}

        {!checksQuery.isLoading && checks.length === 0 && (
          <div className="rounded-lg border-2 border-dashed border-gray-200 bg-white py-16 text-center">
            <p className="text-gray-500">Sin checks en este período.</p>
          </div>
        )}

        {checks.length > 0 && (
          <div className="overflow-hidden rounded-lg border bg-white shadow-sm">
            <table className="min-w-full divide-y divide-gray-200 text-sm">
              <thead className="bg-gray-50">
                <tr>
                  {['Inicio', 'Estado', 'Duración', 'HTTP Status', 'Error'].map(h => (
                    <th key={h} className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500">
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {checks.map(c => (
                  <tr key={c.id} className="hover:bg-gray-50">
                    <td className="whitespace-nowrap px-4 py-3 font-mono text-xs text-gray-700">
                      {fmt(c.started_at)}
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={c.status} />
                    </td>
                    <td className="px-4 py-3 text-gray-600">{c.duration_ms}ms</td>
                    <td className="px-4 py-3 text-gray-600">{c.status_code ?? '—'}</td>
                    <td className="max-w-sm truncate px-4 py-3 text-xs text-red-600">
                      {c.error ?? ''}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <div className="border-t bg-gray-50 px-4 py-2 text-right text-xs text-gray-500">
              {checksQuery.data?.total ?? checks.length} check
              {(checksQuery.data?.total ?? checks.length) !== 1 ? 's' : ''}
            </div>
          </div>
        )}
      </main>
    </div>
  )
}
