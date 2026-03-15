import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BalanceLineChart, prepareChartData } from './BalanceLineChart'
import type { BalanceHistoryResponse } from '../api/client'

vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: any) => <div data-testid="responsive-container">{children}</div>,
  AreaChart: ({ data, children }: any) => <div data-testid="area-chart" data-count={data?.length}>{children}</div>,
  Area: (props: any) => <div data-testid="area" data-stroke={props.stroke} />,
  XAxis: () => <div />,
  YAxis: () => <div />,
  Tooltip: () => <div />,
  CartesianGrid: () => <div />,
  defs: () => <div />,
  linearGradient: () => <div />,
  stop: () => <div />,
}))

const fullHistory: BalanceHistoryResponse = {
  liquid: [
    { date: '2026-03-14', balance: '4950.00' },
    { date: '2026-03-15', balance: '5000.00' },
  ],
  savings: [
    { date: '2026-03-14', balance: '19900.00' },
    { date: '2026-03-15', balance: '20000.00' },
  ],
  investments: [
    { date: '2026-03-14', balance: '49000.00' },
    { date: '2026-03-15', balance: '50000.00' },
  ],
}

const onlyLiquidHistory: BalanceHistoryResponse = {
  liquid: [
    { date: '2026-03-14', balance: '4950.00' },
    { date: '2026-03-15', balance: '5000.00' },
  ],
  savings: [],
  investments: [],
}

const emptyHistory: BalanceHistoryResponse = {
  liquid: [],
  savings: [],
  investments: [],
}

describe('BalanceLineChart', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  it('renders tab buttons for panels with data', () => {
    render(<BalanceLineChart history={fullHistory} isDark={false} />)
    expect(screen.getByRole('button', { name: 'Liquid' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Savings' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Investments' })).toBeInTheDocument()
  })

  it('does NOT render tab for panel with empty history', () => {
    render(<BalanceLineChart history={onlyLiquidHistory} isDark={false} />)
    expect(screen.getByRole('button', { name: 'Liquid' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Savings' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Investments' })).not.toBeInTheDocument()
  })

  it('default active tab is first panel with data', () => {
    render(<BalanceLineChart history={onlyLiquidHistory} isDark={false} />)
    // The area chart should be rendered (not the no-data message)
    expect(screen.getByTestId('area-chart')).toBeInTheDocument()
  })

  it('clicking a tab switches the active panel and updates area stroke color', async () => {
    render(<BalanceLineChart history={fullHistory} isDark={false} />)

    // Initially on Liquid (blue)
    const areaBeforeClick = screen.getByTestId('area')
    expect(areaBeforeClick.getAttribute('data-stroke')).toBe('#3b82f6')

    // Click Savings tab
    await userEvent.click(screen.getByRole('button', { name: 'Savings' }))

    // Area should now use savings color (green)
    const areaAfterClick = screen.getByTestId('area')
    expect(areaAfterClick.getAttribute('data-stroke')).toBe('#22c55e')
  })

  it('shows "No balance history" message when all panels empty', () => {
    render(<BalanceLineChart history={emptyHistory} isDark={false} />)
    expect(screen.getByText(/no balance history/i)).toBeInTheDocument()
    expect(screen.queryByTestId('area-chart')).not.toBeInTheDocument()
  })

  it('uses dark accent color when isDark is true', async () => {
    render(<BalanceLineChart history={onlyLiquidHistory} isDark={true} />)
    const area = screen.getByTestId('area')
    // Liquid dark accent
    expect(area.getAttribute('data-stroke')).toBe('#60a5fa')
  })
})

describe('prepareChartData', () => {
  it('computes delta of 0 for the first point', () => {
    const points = [{ date: '2026-03-14', balance: '5000.00' }]
    const result = prepareChartData(points)
    expect(result[0].delta).toBe(0)
  })

  it('computes positive delta for increasing balance', () => {
    const points = [
      { date: '2026-03-14', balance: '4950.00' },
      { date: '2026-03-15', balance: '5000.00' },
    ]
    const result = prepareChartData(points)
    expect(result[1].delta).toBeCloseTo(50)
  })

  it('computes negative delta for decreasing balance', () => {
    const points = [
      { date: '2026-03-14', balance: '5000.00' },
      { date: '2026-03-15', balance: '4900.00' },
    ]
    const result = prepareChartData(points)
    expect(result[1].delta).toBeCloseTo(-100)
  })

  it('parses balance strings to numbers correctly', () => {
    const points = [{ date: '2026-03-14', balance: '4230.50' }]
    const result = prepareChartData(points)
    expect(result[0].balance).toBe(4230.50)
  })

  it('formats date as short month day string', () => {
    const points = [{ date: '2026-03-15', balance: '5000.00' }]
    const result = prepareChartData(points)
    // Should be something like "Mar 15"
    expect(result[0].date).toMatch(/Mar\s+\d+/)
  })
})
