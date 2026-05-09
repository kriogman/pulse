import type { CheckStatus } from '../types'

const styles: Record<CheckStatus, string> = {
  up: 'bg-green-100 text-green-800',
  down: 'bg-red-100 text-red-800',
  degraded: 'bg-yellow-100 text-yellow-800',
}

export default function StatusBadge({ status }: { status: CheckStatus | string }) {
  const cls = styles[status as CheckStatus] ?? 'bg-gray-100 text-gray-700'
  return (
    <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${cls}`}>
      {status}
    </span>
  )
}
