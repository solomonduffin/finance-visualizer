import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { StackedAreaChart, prepareNetWorthData } from './StackedAreaChart'
import type { NetWorthPoint } from '../api/client'

vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: any) => <div data-testid="responsive-container">{children}</div>,
  AreaChart: ({ data, children }: any) => <div data-testid="area-chart" data-count={data?.length}>{children}</div>,
  Area: (props: any) => <div data-testid="stacked-area" data-datakey={props.dataKey} data-stroke={props.stroke} data-stackid={props.stackId} />,
  XAxis: () => <div />,
  YAxis: () => <div />,
  Tooltip: () => <div />,
  CartesianGrid: () => <div />,
  defs: () => <div />,
  linearGradient: () => <div />,
  stop: () => <div />,
}))

const samplePoints: NetWorthPoint[] = [
  { date: '2026-03-14', liquid: '5000.00', savings: '20000.00', investments: '50000.00' },
  { date: '2026-03-15', liquid: '5100.00', savings: '20200.00', investments: '50500.00' },
]

describe('StackedAreaChart', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  it('renders 3 Area elements for liquid, savings, investments', () => {
    const data = prepareNetWorthData(samplePoints)
    render(<StackedAreaChart data={data} isDark={false} />)
    const areas = screen.getAllByTestId('stacked-area')
    expect(areas).toHaveLength(3)
  })

  it('Area elements have correct dataKey attributes', () => {
    const data = prepareNetWorthData(samplePoints)
    render(<StackedAreaChart data={data} isDark={false} />)
    const areas = screen.getAllByTestId('stacked-area')
    const dataKeys = areas.map(a => a.getAttribute('data-datakey'))
    expect(dataKeys).toContain('liquid')
    expect(dataKeys).toContain('savings')
    expect(dataKeys).toContain('investments')
  })

  it('uses panelColors accent for stroke in light mode', () => {
    const data = prepareNetWorthData(samplePoints)
    render(<StackedAreaChart data={data} isDark={false} />)
    const areas = screen.getAllByTestId('stacked-area')
    const strokes = areas.map(a => a.getAttribute('data-stroke'))
    expect(strokes).toContain('#3b82f6')  // liquid accent
    expect(strokes).toContain('#22c55e')  // savings accent
    expect(strokes).toContain('#a855f7')  // investments accent
  })

  it('uses panelColors darkAccent for stroke in dark mode', () => {
    const data = prepareNetWorthData(samplePoints)
    render(<StackedAreaChart data={data} isDark={true} />)
    const areas = screen.getAllByTestId('stacked-area')
    const strokes = areas.map(a => a.getAttribute('data-stroke'))
    expect(strokes).toContain('#60a5fa')  // liquid darkAccent
    expect(strokes).toContain('#4ade80')  // savings darkAccent
    expect(strokes).toContain('#c084fc')  // investments darkAccent
  })

  it('all Area elements share the same stackId', () => {
    const data = prepareNetWorthData(samplePoints)
    render(<StackedAreaChart data={data} isDark={false} />)
    const areas = screen.getAllByTestId('stacked-area')
    const stackIds = areas.map(a => a.getAttribute('data-stackid'))
    expect(stackIds.every(id => id === 'networth')).toBe(true)
  })
})

describe('prepareNetWorthData', () => {
  it('converts API points to chart data with numeric values', () => {
    const result = prepareNetWorthData(samplePoints)
    expect(result).toHaveLength(2)
    expect(result[0].liquid).toBe(5000)
    expect(result[0].savings).toBe(20000)
    expect(result[0].investments).toBe(50000)
  })

  it('computes total as sum of all three panels', () => {
    const result = prepareNetWorthData(samplePoints)
    expect(result[0].total).toBe(75000)
  })

  it('formats date as short month day string', () => {
    const result = prepareNetWorthData(samplePoints)
    expect(result[0].date).toMatch(/Mar\s+\d+/)
  })

  it('handles empty array', () => {
    const result = prepareNetWorthData([])
    expect(result).toHaveLength(0)
  })
})
