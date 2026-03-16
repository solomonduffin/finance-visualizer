import type { NetWorthStatsData } from '../api/client'
import { formatCurrency } from '../utils/format'

interface NetWorthStatsProps {
  stats: NetWorthStatsData
  selectedDays: number
}

function getPeriodLabel(days: number): string {
  switch (days) {
    case 30:
      return '30-Day Change'
    case 90:
      return '90-Day Change'
    case 180:
      return '6-Month Change'
    case 365:
      return '1-Year Change'
    case 0:
      return 'All-Time Change'
    default:
      return `${days}-Day Change`
  }
}

export function NetWorthStats({ stats, selectedDays }: NetWorthStatsProps) {
  const changeDollars = parseFloat(stats.period_change_dollars)
  const isPositive = changeDollars >= 0
  const changeColorClass = isPositive
    ? 'text-green-600 dark:text-green-400'
    : 'text-red-600 dark:text-red-400'

  const prefix = isPositive ? '+' : ''
  const formattedChange = `${prefix}${formatCurrency(stats.period_change_dollars)}`
  const formattedPct = stats.period_change_pct !== null
    ? ` (${prefix}${stats.period_change_pct}%)`
    : ''

  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-6">
      <div className="flex flex-col gap-4 md:flex-row md:gap-8">
        {/* Current Net Worth */}
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400 mb-1">
            Current Net Worth
          </p>
          <p className="text-3xl font-semibold text-gray-900 dark:text-gray-100">
            {formatCurrency(stats.current_net_worth)}
          </p>
        </div>

        {/* Period Change */}
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400 mb-1">
            {getPeriodLabel(selectedDays)}
          </p>
          <p className={`text-2xl font-semibold ${changeColorClass}`}>
            {formattedChange}{formattedPct}
          </p>
        </div>

        {/* All-Time High */}
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400 mb-1">
            All-Time High
          </p>
          <p className="text-2xl font-semibold text-gray-900 dark:text-gray-100">
            {formatCurrency(stats.all_time_high)}
          </p>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            {stats.all_time_high_date}
          </p>
        </div>
      </div>
    </div>
  )
}
