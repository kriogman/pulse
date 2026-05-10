import type { MonitorStats } from '../types'

interface Props {
  stats: MonitorStats
}

function StatCard({
  label,
  value,
  sub,
  color,
}: {
  label: string
  value: string
  sub?: string
  color?: string
}) {
  return (
    <div className="rounded-lg border bg-white px-5 py-4 shadow-sm">
      <p className="text-xs font-medium uppercase tracking-wide text-gray-500">{label}</p>
      <p className={`mt-1 text-2xl font-bold ${color ?? 'text-gray-900'}`}>{value}</p>
      {sub && <p className="mt-0.5 text-xs text-gray-400">{sub}</p>}
    </div>
  )
}

export default function UptimeStats({ stats }: Props) {
  const uptimeColor =
    stats.uptime_pct >= 99
      ? 'text-green-600'
      : stats.uptime_pct >= 95
        ? 'text-yellow-600'
        : 'text-red-600'

  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
      <StatCard
        label={`Uptime (${stats.period})`}
        value={`${stats.uptime_pct.toFixed(2)}%`}
        sub={`${stats.up_count} / ${stats.total_checks} checks OK`}
        color={uptimeColor}
      />
      <StatCard
        label="Total checks"
        value={String(stats.total_checks)}
        sub={`${stats.down_count} caídas · ${stats.degraded_count} degradados`}
      />
      <StatCard
        label="Respuesta media"
        value={`${stats.avg_duration_ms}ms`}
      />
      <StatCard
        label="Respuesta máx."
        value={`${stats.max_duration_ms}ms`}
      />
    </div>
  )
}
