import {
  PieChart,
  Pie,
  Cell,
  Label,
  Tooltip,
  ResponsiveContainer,
} from 'recharts'
import { PANEL_COLORS } from './panelColors'
import { formatCurrency } from '../utils/format'

type PanelKey = 'liquid' | 'savings' | 'investments'

const PANEL_KEYS: PanelKey[] = ['liquid', 'savings', 'investments']

export interface NetWorthDonutProps {
  liquid: string
  savings: string
  investments: string
  isDark: boolean
}

interface Segment {
  name: string
  value: number
  color: string
  key: PanelKey
}

export function NetWorthDonut({ liquid, savings, investments, isDark }: NetWorthDonutProps) {
  const rawValues: Record<PanelKey, number> = {
    liquid: parseFloat(liquid),
    savings: parseFloat(savings),
    investments: parseFloat(investments),
  }

  const total = rawValues.liquid + rawValues.savings + rawValues.investments

  const segments: Segment[] = PANEL_KEYS
    .filter((key) => rawValues[key] > 0)
    .map((key) => ({
      key,
      name: PANEL_COLORS[key].label,
      value: rawValues[key],
      color: isDark ? PANEL_COLORS[key].darkAccent : PANEL_COLORS[key].accent,
    }))

  if (total === 0) {
    return (
      <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-6 flex items-center justify-center h-60">
        <p className="text-gray-500 dark:text-gray-400 text-sm">No data</p>
      </div>
    )
  }

  const formattedTotal = total.toLocaleString('en-US', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 0,
  })

  return (
    <div className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-6">
      <h3 className="text-sm font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-4">
        Net Worth
      </h3>

      {/* Donut chart */}
      <div className="h-60">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={segments}
              cx="50%"
              cy="50%"
              innerRadius={65}
              outerRadius={95}
              paddingAngle={3}
              dataKey="value"
              stroke="none"
            >
              {segments.map((seg) => (
                <Cell key={seg.key} fill={seg.color} />
              ))}
              <Label value={formattedTotal} position="center" />
            </Pie>
            <Tooltip
              formatter={(v: number) => formatCurrency(v.toFixed(2))}
            />
          </PieChart>
        </ResponsiveContainer>
      </div>

      {/* Custom legend */}
      <div className="mt-4 space-y-2">
        {PANEL_KEYS.map((key) => {
          const value = rawValues[key]
          const color = isDark ? PANEL_COLORS[key].darkAccent : PANEL_COLORS[key].accent
          return (
            <div key={key} className="flex items-center justify-between text-sm">
              <div className="flex items-center gap-2">
                <span
                  className="inline-block w-2.5 h-2.5 rounded-full flex-shrink-0"
                  style={{ backgroundColor: color }}
                />
                <span className="text-gray-700 dark:text-gray-300">{PANEL_COLORS[key].label}</span>
              </div>
              <span className="font-medium text-gray-900 dark:text-gray-100">
                {formatCurrency(value.toFixed(2))}
              </span>
            </div>
          )
        })}
      </div>
    </div>
  )
}
