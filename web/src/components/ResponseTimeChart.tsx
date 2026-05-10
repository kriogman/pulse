import {
  ResponsiveContainer,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ReferenceLine,
} from 'recharts'
import type { CheckOutput } from '../types'

interface Props {
  checks: CheckOutput[]
  avgDurationMs?: number
}

const STATUS_COLOR: Record<string, string> = {
  up: '#16a34a',
  down: '#dc2626',
  degraded: '#d97706',
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function CustomDot(props: any) {
  const { cx, cy, payload } = props
  const color = STATUS_COLOR[payload.status] ?? '#6b7280'
  return <circle key={`dot-${payload.index}`} cx={cx} cy={cy} r={3} fill={color} stroke="none" />
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function CustomTooltip({ active, payload }: any) {
  if (!active || !payload?.length) return null
  const d = payload[0].payload
  const color = STATUS_COLOR[d.status] ?? '#6b7280'
  return (
    <div className="rounded border bg-white px-3 py-2 text-xs shadow-lg">
      <p className="font-medium text-gray-700">{d.time}</p>
      <p style={{ color }}>{d.status} — {d.duration_ms}ms</p>
    </div>
  )
}

export default function ResponseTimeChart({ checks, avgDurationMs }: Props) {
  const data = [...checks]
    .reverse()
    .map((c, i) => ({
      index: i,
      duration_ms: c.duration_ms,
      status: c.status,
      time: new Date(c.started_at).toLocaleString('es-ES', {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      }),
    }))

  if (data.length === 0) return null

  return (
    <div className="rounded-lg border bg-white p-4 shadow-sm">
      <p className="mb-3 text-sm font-medium text-gray-700">Tiempo de respuesta</p>
      <ResponsiveContainer width="100%" height={180}>
        <LineChart data={data} margin={{ top: 4, right: 8, left: 0, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
          <XAxis
            dataKey="index"
            tick={false}
            axisLine={false}
            tickLine={false}
          />
          <YAxis
            tickFormatter={v => `${v}ms`}
            tick={{ fontSize: 11, fill: '#9ca3af' }}
            axisLine={false}
            tickLine={false}
            width={55}
          />
          <Tooltip content={<CustomTooltip />} />
          {avgDurationMs !== undefined && avgDurationMs > 0 && (
            <ReferenceLine
              y={avgDurationMs}
              stroke="#93c5fd"
              strokeDasharray="4 2"
              label={{ value: 'media', position: 'insideTopRight', fontSize: 10, fill: '#93c5fd' }}
            />
          )}
          <Line
            type="monotone"
            dataKey="duration_ms"
            stroke="#3b82f6"
            strokeWidth={1.5}
            dot={<CustomDot />}
            activeDot={{ r: 5 }}
            isAnimationActive={false}
          />
        </LineChart>
      </ResponsiveContainer>
      <p className="mt-1 text-right text-xs text-gray-400">
        {data.length} punto{data.length !== 1 ? 's' : ''} · antiguo → reciente
      </p>
    </div>
  )
}
