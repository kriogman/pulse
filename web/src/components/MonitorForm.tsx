import { useState } from 'react'
import type { MonitorBody, MonitorOutput } from '../types'

interface Props {
  initial?: MonitorOutput
  onSubmit: (data: MonitorBody) => void
  onCancel: () => void
  loading?: boolean
  error?: string
}

function toForm(m?: MonitorOutput) {
  return {
    name: m?.name ?? '',
    type: m?.type ?? 'http',
    target: m?.target ?? '',
    interval_sec: String(m?.interval_sec ?? 60),
    timeout_ms: String(m?.timeout_ms ?? 5000),
    expected_status: String((m?.config?.expected_status as number | undefined) ?? 200),
    enabled: m?.enabled ?? true,
  }
}

export default function MonitorForm({ initial, onSubmit, onCancel, loading, error }: Props) {
  const [form, setForm] = useState(toForm(initial))

  function set(field: string, value: string | boolean) {
    setForm(f => ({ ...f, [field]: value }))
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    onSubmit({
      name: form.name.trim(),
      type: form.type,
      target: form.target.trim(),
      interval_sec: parseInt(form.interval_sec, 10),
      timeout_ms: parseInt(form.timeout_ms, 10),
      config: { expected_status: parseInt(form.expected_status, 10) },
      enabled: form.enabled,
    })
  }

  const label = 'block text-sm font-medium text-gray-700 mb-1'
  const input =
    'block w-full rounded-md border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-lg rounded-lg bg-white shadow-xl">
        <div className="border-b px-6 py-4">
          <h2 className="text-lg font-semibold text-gray-900">
            {initial ? 'Editar monitor' : 'Nuevo monitor'}
          </h2>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4 px-6 py-5">
          {error && (
            <p className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>
          )}

          <div>
            <label className={label}>Nombre</label>
            <input
              className={input}
              value={form.name}
              onChange={e => set('name', e.target.value)}
              required
              maxLength={100}
              placeholder="mi-api"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className={label}>Tipo</label>
              <select
                className={input}
                value={form.type}
                onChange={e => set('type', e.target.value)}
              >
                <option value="http">HTTP</option>
              </select>
            </div>
            <div>
              <label className={label}>Status esperado</label>
              <input
                className={input}
                type="number"
                value={form.expected_status}
                onChange={e => set('expected_status', e.target.value)}
                min={100}
                max={599}
              />
            </div>
          </div>

          <div>
            <label className={label}>URL objetivo</label>
            <input
              className={input}
              value={form.target}
              onChange={e => set('target', e.target.value)}
              required
              placeholder="https://ejemplo.com/health"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className={label}>Intervalo (segundos)</label>
              <input
                className={input}
                type="number"
                value={form.interval_sec}
                onChange={e => set('interval_sec', e.target.value)}
                min={5}
                max={86400}
              />
            </div>
            <div>
              <label className={label}>Timeout (ms)</label>
              <input
                className={input}
                type="number"
                value={form.timeout_ms}
                onChange={e => set('timeout_ms', e.target.value)}
                min={100}
                max={30000}
              />
            </div>
          </div>

          <div className="flex items-center gap-2">
            <input
              id="enabled"
              type="checkbox"
              checked={form.enabled}
              onChange={e => set('enabled', e.target.checked)}
              className="h-4 w-4 rounded border-gray-300 text-blue-600"
            />
            <label htmlFor="enabled" className="text-sm text-gray-700">
              Activo al crear
            </label>
          </div>

          <div className="flex justify-end gap-3 border-t pt-4">
            <button
              type="button"
              onClick={onCancel}
              disabled={loading}
              className="rounded-md px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100 disabled:opacity-50"
            >
              Cancelar
            </button>
            <button
              type="submit"
              disabled={loading}
              className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            >
              {loading ? 'Guardando…' : initial ? 'Guardar cambios' : 'Crear monitor'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
