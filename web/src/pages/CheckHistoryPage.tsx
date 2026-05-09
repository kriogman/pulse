import { useParams, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api } from '../api/client'
import StatusBadge from '../components/StatusBadge'

function fmt(iso: string) {
  return new Date(iso).toLocaleString('es-ES', {
    dateStyle: 'short',
    timeStyle: 'medium',
  })
}

export default function CheckHistoryPage() {
  const { id } = useParams<{ id: string }>()

  const monitorQuery = useQuery({
    queryKey: ['monitors', id],
    queryFn: () => api.monitors.get(id!),
    enabled: !!id,
  })

  const checksQuery = useQuery({
    queryKey: ['checks', id],
    queryFn: () => api.checks.list(id!),
    enabled: !!id,
    refetchInterval: 15_000,
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

      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6">
        {monitor && (
          <div className="mb-4 flex items-center gap-4 text-sm text-gray-600">
            <span>
              <span className="font-medium">URL:</span>{' '}
              <a
                href={monitor.target}
                target="_blank"
                rel="noreferrer"
                className="text-blue-600 hover:underline"
              >
                {monitor.target}
              </a>
            </span>
            <span>
              <span className="font-medium">Intervalo:</span> {monitor.interval_sec}s
            </span>
            <span>
              <span className="font-medium">Timeout:</span> {monitor.timeout_ms}ms
            </span>
          </div>
        )}

        {checksQuery.isLoading && (
          <p className="text-center text-sm text-gray-500">Cargando checks…</p>
        )}
        {checksQuery.isError && (
          <p className="text-center text-sm text-red-600">Error al cargar el historial.</p>
        )}

        {!checksQuery.isLoading && checks.length === 0 && (
          <div className="rounded-lg border-2 border-dashed border-gray-200 bg-white py-16 text-center">
            <p className="text-gray-500">Aún no hay checks registrados para este monitor.</p>
          </div>
        )}

        {checks.length > 0 && (
          <div className="overflow-hidden rounded-lg border bg-white shadow-sm">
            <table className="min-w-full divide-y divide-gray-200 text-sm">
              <thead className="bg-gray-50">
                <tr>
                  {['Inicio', 'Estado', 'Duración', 'HTTP Status', 'Error'].map(h => (
                    <th
                      key={h}
                      className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500"
                    >
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {checks.map(c => (
                  <tr key={c.id} className="hover:bg-gray-50">
                    <td className="whitespace-nowrap px-4 py-3 text-gray-700 font-mono text-xs">
                      {fmt(c.started_at)}
                    </td>
                    <td className="px-4 py-3">
                      <StatusBadge status={c.status} />
                    </td>
                    <td className="px-4 py-3 text-gray-600">{c.duration_ms}ms</td>
                    <td className="px-4 py-3 text-gray-600">
                      {c.status_code ?? '—'}
                    </td>
                    <td className="max-w-sm truncate px-4 py-3 text-red-600 text-xs">
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
