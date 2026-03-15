import { useState } from 'react'
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
} from 'recharts'
import type { TooltipProps } from 'recharts'
import type { BalanceHistoryResponse, HistoryPoint } from '../api/client'
import { PANEL_COLORS } from './panelColors'

type PanelKey = 'liquid' | 'savings' | 'investments'

export interface ChartPoint {
  date: string
  balance: number
  delta: number
}

export function prepareChartData(points: HistoryPoint[]): ChartPoint[] {
  return points.map((point, i) => {
    const balance = parseFloat(point.balance)
    const prev = i > 0 ? parseFloat(points[i - 1].balance) : balance
    const delta = i === 0 ? 0 : balance - prev

    // Parse the date and format as "Mar 15"
    const [year, month, day] = point.date.split('-').map(Number)
    const dateObj = new Date(year, month - 1, day)
    const formattedDate = dateObj.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
    })

    return { date: formattedDate, balance, delta }
  })
}

interface CustomTooltipProps extends TooltipProps<number, string> {
  accentColor: string
}

function CustomTooltip({ active, payload, label, accentColor }: CustomTooltipProps) {
  if (!active || !payload?.length) return null

  const { balance, delta } = payload[0].payload as ChartPoint

  const formattedBalance = balance.toLocaleString('en-US', {
    style: 'currency',
    currency: 'USD',
  })

  let deltaEl: React.ReactNode = null
  if (delta > 0) {
    const formattedDelta = delta.toLocaleString('en-US', {
      style: 'currency',
      currency: 'USD',
    })
    deltaEl = <span className="text-green-500 font-medium">↑{formattedDelta}</span>
  } else if (delta < 0) {
    const formattedDelta = Math.abs(delta).toLocaleString('en-US', {
      style: 'currency',
      currency: 'USD',
    })
    deltaEl = <span className="text-red-500 font-medium">↓{formattedDelta}</span>
  }

  return (
    <div className="bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-md p-3 text-sm">
      <p className="font-semibold text-gray-700 dark:text-gray-200 mb-1">{label}</p>
      <p style={{ color: accentColor }} className="font-bold text-base">{formattedBalance}</p>
      {deltaEl && <p className="mt-1">{deltaEl}</p>}
    </div>
  )
}

export interface BalanceLineChartProps {
  history: BalanceHistoryResponse
  isDark: boolean
}

export function BalanceLineChart({ history, isDark }: BalanceLineChartProps) {
  const panelKeys: PanelKey[] = ['liquid', 'savings', 'investments']
  const panelsWithData = panelKeys.filter((key) => history[key].length > 0)

  const [activePanel, setActivePanel] = useState<PanelKey>(
    panelsWithData.length > 0 ? panelsWithData[0] : 'liquid'
  )

  if (panelsWithData.length === 0) {
    return (
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-6">
        <p className="text-center text-gray-500 dark:text-gray-400">No balance history yet</p>
      </div>
    )
  }

  const accentColor = isDark
    ? PANEL_COLORS[activePanel].darkAccent
    : PANEL_COLORS[activePanel].accent

  const chartData = prepareChartData(history[activePanel])
  const gradientId = `gradient-${activePanel}`

  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-6">
      {/* Tab bar */}
      <div className="flex gap-4 mb-6">
        {panelsWithData.map((key) => {
          const isActive = key === activePanel
          const color = isDark ? PANEL_COLORS[key].darkAccent : PANEL_COLORS[key].accent
          return (
            <button
              key={key}
              type="button"
              onClick={() => setActivePanel(key)}
              className={`pb-2 text-sm font-medium transition-colors ${
                isActive
                  ? 'border-b-2 font-bold'
                  : 'text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-300'
              }`}
              style={
                isActive
                  ? { borderBottomColor: color, color }
                  : undefined
              }
            >
              {PANEL_COLORS[key].label}
            </button>
          )
        })}
      </div>

      {/* Chart */}
      <ResponsiveContainer width="100%" height={280}>
        <AreaChart data={chartData} margin={{ top: 4, right: 4, left: 4, bottom: 4 }}>
          <defs>
            <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={accentColor} stopOpacity={0.25} />
              <stop offset="95%" stopColor={accentColor} stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" strokeOpacity={0.2} />
          <XAxis
            dataKey="date"
            tick={{ fontSize: 11 }}
            stroke={isDark ? '#9ca3af' : '#6b7280'}
          />
          <YAxis
            tick={{ fontSize: 11 }}
            stroke={isDark ? '#9ca3af' : '#6b7280'}
            tickFormatter={(v: number) => {
              if (Math.abs(v) >= 1000) {
                return '$' + (v / 1000).toFixed(0) + 'k'
              }
              return '$' + v.toFixed(0)
            }}
          />
          <Tooltip content={(props) => <CustomTooltip {...props} accentColor={accentColor} />} />
          <Area
            type="monotone"
            dataKey="balance"
            stroke={accentColor}
            strokeWidth={2}
            fill={`url(#${gradientId})`}
            fillOpacity={1}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
}
