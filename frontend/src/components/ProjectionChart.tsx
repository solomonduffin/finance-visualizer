import {
  ComposedChart,
  Area,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ReferenceLine,
  ResponsiveContainer,
  Label,
} from 'recharts'
import { formatCurrency } from '../utils/format'
import type { ProjectionPoint } from '../utils/projection'

interface ProjectionChartProps {
  historicalData: Array<{ date: string; value: number }>
  projectionData: ProjectionPoint[]
  isDark: boolean
}

interface CombinedPoint {
  date: string
  historical: number | null
  projected: number | null
}

function formatYAxis(value: number): string {
  if (Math.abs(value) >= 1_000_000) {
    return `$${(value / 1_000_000).toFixed(1)}M`
  }
  if (Math.abs(value) >= 1000) {
    return `$${(value / 1000).toFixed(0)}k`
  }
  return `$${value}`
}

function formatXTick(dateStr: string, dataLength: number): string {
  const d = new Date(dateStr + 'T00:00:00')
  if (dataLength > 60) {
    // Long projections: show year
    return d.toLocaleDateString('en-US', { year: 'numeric' })
  }
  return d.toLocaleDateString('en-US', { month: 'short', year: '2-digit' })
}

interface TooltipPayloadEntry {
  dataKey: string
  value: number | null
  payload: CombinedPoint
}

interface ProjectionTooltipProps {
  active?: boolean
  payload?: TooltipPayloadEntry[]
  label?: string
  isDark: boolean
}

function ProjectionTooltip({ active, payload, isDark }: ProjectionTooltipProps) {
  if (!active || !payload || payload.length === 0) return null

  const point = payload[0].payload
  const isProjected = point.projected !== null && point.historical === null
  const value = isProjected ? point.projected : point.historical

  if (value === null) return null

  const d = new Date(point.date + 'T00:00:00')
  const dateLabel = d.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  })

  return (
    <div
      className={`rounded-lg shadow-lg p-3 ${
        isDark
          ? 'bg-gray-800 border border-gray-700'
          : 'bg-white border border-gray-200'
      }`}
    >
      <p
        className={`text-xs font-semibold mb-1 ${
          isDark ? 'text-gray-400' : 'text-gray-500'
        }`}
      >
        {dateLabel}
      </p>
      <p className={`text-xs ${isDark ? 'text-gray-400' : 'text-gray-500'}`}>
        {isProjected ? 'Projected' : 'Historical'}
      </p>
      <p
        className={`text-sm font-semibold ${
          isDark ? 'text-gray-100' : 'text-gray-900'
        }`}
      >
        {formatCurrency(value.toFixed(2))}
      </p>
    </div>
  )
}

export function ProjectionChart({
  historicalData,
  projectionData,
  isDark,
}: ProjectionChartProps) {
  if (
    (!historicalData || historicalData.length === 0) &&
    (!projectionData || projectionData.length === 0)
  ) {
    return (
      <div
        className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-6"
        role="img"
        aria-label="Net worth projection chart showing historical and projected values"
      >
        <div className="flex items-center justify-center h-[400px] sm:h-[400px]">
          <p className="text-gray-500 dark:text-gray-400 text-sm">
            No data to project. Sync your accounts to see projections.
          </p>
        </div>
      </div>
    )
  }

  // Build combined data array
  const combined: CombinedPoint[] = []

  // Historical points
  for (const h of historicalData) {
    combined.push({ date: h.date, historical: h.value, projected: null })
  }

  // Bridge point: last historical connects to first projection
  const todayStr =
    historicalData.length > 0
      ? historicalData[historicalData.length - 1].date
      : projectionData.length > 0
        ? projectionData[0].date
        : ''

  // If we have both datasets, add a bridge point where both values overlap
  if (historicalData.length > 0 && projectionData.length > 0) {
    const lastHistorical = historicalData[historicalData.length - 1]
    // Replace the last historical point with a bridge point
    combined[combined.length - 1] = {
      date: lastHistorical.date,
      historical: lastHistorical.value,
      projected: lastHistorical.value,
    }
  }

  // Projection points (skip the first one if it's the bridge date)
  for (const p of projectionData) {
    if (p.date === todayStr) continue
    combined.push({ date: p.date, historical: null, projected: p.value })
  }

  const gridStroke = isDark ? '#374151' : '#e5e7eb'
  const historicalStroke = isDark ? '#d1d5db' : '#374151'
  const projectedStroke = isDark ? '#60a5fa' : '#3b82f6'
  const refLineStroke = isDark ? '#6b7280' : '#9ca3af'
  const gradientStart = isDark ? 'rgba(96,165,250,0.15)' : 'rgba(59,130,246,0.2)'

  return (
    <div
      className="bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-100 dark:border-gray-700 p-6"
      role="img"
      aria-label="Net worth projection chart showing historical and projected values"
    >
      <ResponsiveContainer width="100%" height={400}>
        <ComposedChart
          data={combined}
          margin={{ top: 16, right: 8, left: 8, bottom: 0 }}
        >
          <defs>
            <linearGradient id="projGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={gradientStart} stopOpacity={1} />
              <stop offset="95%" stopColor={gradientStart} stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid
            strokeDasharray="3 3"
            strokeOpacity={0.2}
            stroke={gridStroke}
          />
          <XAxis
            dataKey="date"
            tick={{ fontSize: 12 }}
            tickFormatter={(d: string) => formatXTick(d, combined.length)}
          />
          <YAxis tickFormatter={formatYAxis} tick={{ fontSize: 12 }} />
          <Tooltip
            content={<ProjectionTooltip isDark={isDark} />}
          />
          <Area
            type="monotone"
            dataKey="projected"
            fill="url(#projGradient)"
            stroke="none"
            connectNulls={false}
          />
          <Line
            type="monotone"
            dataKey="historical"
            stroke={historicalStroke}
            strokeWidth={2}
            dot={false}
            connectNulls={false}
          />
          <Line
            type="monotone"
            dataKey="projected"
            stroke={projectedStroke}
            strokeWidth={2}
            strokeDasharray="8 4"
            dot={false}
            connectNulls={false}
          />
          {todayStr && (
            <ReferenceLine
              x={todayStr}
              stroke={refLineStroke}
              strokeDasharray="4 4"
            >
              <Label
                value="Now"
                position="top"
                fill={isDark ? '#9ca3af' : '#6b7280'}
                fontSize={12}
                fontWeight={600}
              />
            </ReferenceLine>
          )}
        </ComposedChart>
      </ResponsiveContainer>
    </div>
  )
}
