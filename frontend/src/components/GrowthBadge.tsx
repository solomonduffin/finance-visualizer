import { formatCurrency } from '../utils/format'

interface GrowthBadgeProps {
  pctChange: string | null    // e.g., "2.30" or "-1.50", null = no data
  dollarChange: string | null // e.g., "280.00" or "-150.00"
  visible: boolean            // controlled by settings toggle
}

export function GrowthBadge({ pctChange, dollarChange, visible }: GrowthBadgeProps) {
  const hasData = pctChange !== null && pctChange !== '0.00'
  const isPositive = hasData && !pctChange.startsWith('-')
  const pct = hasData ? Number(pctChange).toFixed(1) : '0.0'

  // Invisible placeholder for layout consistency
  if (!visible || !hasData) {
    return (
      <span className="invisible text-sm font-semibold ml-2">
        {'\u25B2'} +0.0%
      </span>
    )
  }

  const tooltipText = dollarChange
    ? `${isPositive ? '+' : ''}${formatCurrency(dollarChange)} over 30 days`
    : ''

  return (
    <span
      className={`text-sm font-semibold ml-2 ${
        isPositive
          ? 'text-green-600 dark:text-green-400'
          : 'text-red-600 dark:text-red-400'
      }`}
      title={tooltipText}
    >
      {isPositive ? '\u25B2' : '\u25BC'} {isPositive ? '+' : ''}{pct}%
    </span>
  )
}
