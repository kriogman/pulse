import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { MonitorBody, MonitorOutput } from '../types'
import MonitorForm from '../components/MonitorForm'
import ConfirmDialog from '../components/ConfirmDialog'

type FormMode = { kind: 'create' } | { kind: 'edit'; monitor: MonitorOutput }

export default function MonitorListPage() {
  const qc = useQueryClient()
  const [formMode, setFormMode] = useState<FormMode | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<MonitorOutput | null>(null)
  const [formError, setFormError] = useState<string | undefined>()

  const { data, isLoading, isError } = useQuery({
    queryKey: ['monitors'],
    queryFn: () => api.monitors.list(),
    refetchInterval: 30_000,
  })

  const invalidate = () => qc.invalidateQueries({ queryKey: ['monitors'] })

  const createMutation = useMutation({
    mutationFn: (body: MonitorBody) => api.monitors.create(body),
    onSuccess: () => { invalidate(); setFormMode(null); setFormError(undefined) },
    onError: (e: Error) => setFormError(e.message),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, body }: { id: string; body: MonitorBody }) =>
      api.monitors.update(id, body),
    onSuccess: () => { invalidate(); setFormMode(null); setFormError(undefined) },
    onError: (e: Error) => setFormError(e.message),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.monitors.delete(id),
    onSuccess: () => { invalidate(); setDeleteTarget(null) },
  })

  const toggleMutation = useMutation({
    mutationFn: (m: MonitorOutput) =>
      m.enabled ? api.monitors.pause(m.id) : api.monitors.resume(m.id),
    onSuccess: invalidate,
  })

  function handleSubmit(body: MonitorBody) {
    if (formMode?.kind === 'edit') {
      updateMutation.mutate({ id: formMode.monitor.id, body })
    } else {
      createMutation.mutate(body)
    }
  }

  const isMutating = createMutation.isPending || updateMutation.isPending

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Navbar */}
      <header className="border-b bg-white shadow-sm">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-4 py-4 sm:px-6">
          <div className="flex items-center gap-2">
            <span className="text-xl font-bold text-blue-600">Pulse</span>
            <span className="text-gray-400">—</span>
            <span className="text-sm text-gray-500">Monitor Dashboard</span>
          </div>
          <button
            onClick={() => { setFormMode({ kind: 'create' }); setFormError(undefined) }}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
          >
            + Nuevo monitor
          </button>
        </div>
      </header>

      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6">
        {isLoading && (
          <p className="text-center text-sm text-gray-500">Cargando monitores…</p>
        )}
        {isError && (
          <p className="text-center text-sm text-red-600">Error al cargar los monitores.</p>
        )}

        {data && data.data.length === 0 && (
          <div className="rounded-lg border-2 border-dashed border-gray-200 bg-white py-16 text-center">
            <p className="text-gray-500">No hay monitores configurados.</p>
            <button
              onClick={() => setFormMode({ kind: 'create' })}
              className="mt-4 text-sm font-medium text-blue-600 hover:underline"
            >
              Crear el primero
            </button>
          </div>
        )}

        {data && data.data.length > 0 && (
          <div className="overflow-hidden rounded-lg border bg-white shadow-sm">
            <table className="min-w-full divide-y divide-gray-200 text-sm">
              <thead className="bg-gray-50">
                <tr>
                  {['Nombre', 'Tipo', 'URL objetivo', 'Intervalo', 'Timeout', 'Estado', 'Acciones'].map(h => (
                    <th key={h} className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-gray-500">
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100">
                {data.data.map(m => (
                  <tr key={m.id} className="hover:bg-gray-50">
                    <td className="px-4 py-3 font-medium text-gray-900">{m.name}</td>
                    <td className="px-4 py-3">
                      <span className="rounded bg-gray-100 px-2 py-0.5 text-xs font-mono text-gray-600">
                        {m.type}
                      </span>
                    </td>
                    <td className="max-w-xs truncate px-4 py-3 text-gray-600" title={m.target}>
                      {m.target}
                    </td>
                    <td className="px-4 py-3 text-gray-600">{m.interval_sec}s</td>
                    <td className="px-4 py-3 text-gray-600">{m.timeout_ms}ms</td>
                    <td className="px-4 py-3">
                      <span
                        className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-xs font-medium ${
                          m.enabled
                            ? 'bg-green-100 text-green-800'
                            : 'bg-gray-100 text-gray-600'
                        }`}
                      >
                        <span
                          className={`h-1.5 w-1.5 rounded-full ${m.enabled ? 'bg-green-500' : 'bg-gray-400'}`}
                        />
                        {m.enabled ? 'activo' : 'pausado'}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <Link
                          to={`/monitors/${m.id}/checks`}
                          className="text-xs text-blue-600 hover:underline"
                        >
                          Checks
                        </Link>
                        <button
                          onClick={() => { setFormMode({ kind: 'edit', monitor: m }); setFormError(undefined) }}
                          className="text-xs text-gray-600 hover:text-gray-900"
                        >
                          Editar
                        </button>
                        <button
                          onClick={() => toggleMutation.mutate(m)}
                          disabled={toggleMutation.isPending}
                          className="text-xs text-amber-600 hover:text-amber-800 disabled:opacity-40"
                        >
                          {m.enabled ? 'Pausar' : 'Reanudar'}
                        </button>
                        <button
                          onClick={() => setDeleteTarget(m)}
                          className="text-xs text-red-600 hover:text-red-800"
                        >
                          Eliminar
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <div className="border-t bg-gray-50 px-4 py-2 text-right text-xs text-gray-500">
              {data.total} monitor{data.total !== 1 ? 'es' : ''}
            </div>
          </div>
        )}
      </main>

      {formMode && (
        <MonitorForm
          initial={formMode.kind === 'edit' ? formMode.monitor : undefined}
          onSubmit={handleSubmit}
          onCancel={() => setFormMode(null)}
          loading={isMutating}
          error={formError}
        />
      )}

      {deleteTarget && (
        <ConfirmDialog
          title="Eliminar monitor"
          message={`¿Eliminar "${deleteTarget.name}" y todo su historial de checks? Esta acción no se puede deshacer.`}
          onConfirm={() => deleteMutation.mutate(deleteTarget.id)}
          onCancel={() => setDeleteTarget(null)}
          loading={deleteMutation.isPending}
        />
      )}
    </div>
  )
}
