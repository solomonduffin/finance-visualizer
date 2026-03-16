import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts'
import { PANEL_COLORS } from './panelColors'
import { formatCurrency } from '../utils/format'
import type { NetWorthPoint } from '../api/client'

export interface ChartDataPoint {
  date: string
  liquid: number
  savings: number
  investments: number
  total: number
}

/**
 * Converts API response points to chart-ready data.
 * Parses decimal strings to numbers, formats dates to "Mar 15" style,
 * and computes total per point.
 */
export function prepareNetWorthData(points: NetWorthPoint[]): ChartDataPoint[] {
  return points.map((p) => {
    const liquid = parseFloat(p.liquid)
    const savings = parseFloat(p.savings)
    const investments = parseFloat(p.investments)
    // Format date to short month day (e.g., "Mar 15")
    const d = new Date(p.date + 'T00:00:00')
    const dateStr = d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
    return {
      date: dateStr,
      liquid,
      savings,
      investments,
      total: liquid + savings + investments,
    }
  })
}

interface StackedAreaChartProps {
  data: ChartDataPoint[]
  isDark: boolean
}

function formatYAxis(value: number): string {
  if (Math.abs(value) >= 1000) {
    return `$${(value / 1000).toFixed(0)}k`
  }
  return `$${value}`
}

interface TooltipPayloadItem {
  name: string
  value: number
  color: string
  dataKey: string
}

interface NetWorthTooltipProps {
  active?: boolean
  payload?: TooltipPayloadItem[]
  label?: string
}

function NetWorthTooltip({ active, payload, label }: NetWorthTooltipProps) {
  if (!active || !payload || payload.length === 0) return null

  const total = payload.reduce((sum, entry) => sum + entry.value, 0)

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 p-3 text-sm">
      <p className="font-semibold text-gray-900 dark:text-gray-100 mb-1">{label}</p>
      <p className="font-semibold text-base text-gray-900 dark:text-gray-100 mb-2">
        {formatCurrency(total.toFixed(2))}
      </p>
      {payload.map((entry) => (
        <div key={entry.dataKey} className="flex items-center gap-2 py-0.5">
          <span
            className="inline-block w-2.5 h-2.5 rounded-full flex-shrink-0"
            style={{ backgroundColor: entry.color }}
          />
          <span className="text-gray-600 dark:text-gray-400">{entry.name}</span>
          <span className="ml-auto text-gray-900 dark:text-gray-100">
            {formatCurrency(entry.value.toFixed(2))}
          </span>
        </div>
      ))}
    </div>
  )
}

export function StackedAreaChart({ data, isDark }: StackedAreaChartProps) {
  const liquidColor = isDark ? PANEL_COLORS.liquid.darkAccent : PANEL_COLORS.liquid.accent
  const savingsColor = isDark ? PANEL_COLORS.savings.darkAccent : PANEL_COLORS.savings.accent
  const investmentsColor = isDark ? PANEL_COLORS.investments.darkAccent : PANEL_COLORS.investments.accent

  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-6">
      <ResponsiveContainer width="100%" height={400}>
        <AreaChart data={data} margin={{ top: 8, right: 8, left: 8, bottom: 0 }}>
          <defs>
            <linearGradient id="gradLiquid" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={liquidColor} stopOpacity={0.6} />
              <stop offset="95%" stopColor={liquidColor} stopOpacity={0.1} />
            </linearGradient>
            <linearGradient id="gradSavings" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={savingsColor} stopOpacity={0.6} />
              <stop offset="95%" stopColor={savingsColor} stopOpacity={0.1} />
            </linearGradient>
            <linearGradient id="gradInvestments" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={investmentsColor} stopOpacity={0.6} />
              <stop offset="95%" stopColor={investmentsColor} stopOpacity={0.1} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" strokeOpacity={0.2} />
          <XAxis dataKey="date" tick={{ fontSize: 12 }} />
          <YAxis tickFormatter={formatYAxis} tick={{ fontSize: 12 }} />
          <Tooltip content={<NetWorthTooltip />} />
          <Area
            type="monotone"
            dataKey="liquid"
            name={PANEL_COLORS.liquid.label}
            stackId="networth"
            stroke={liquidColor}
            fill="url(#gradLiquid)"
          />
          <Area
            type="monotone"
            dataKey="savings"
            name={PANEL_COLORS.savings.label}
            stackId="networth"
            stroke={savingsColor}
            fill="url(#gradSavings)"
          />
          <Area
            type="monotone"
            dataKey="investments"
            name={PANEL_COLORS.investments.label}
            stackId="networth"
            stroke={investmentsColor}
            fill="url(#gradInvestments)"
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
}
