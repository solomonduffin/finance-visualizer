import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import NetWorth from './NetWorth'
import * as client from '../api/client'

vi.mock('../api/client')

vi.mock('../components/StackedAreaChart', () => ({
  StackedAreaChart: (props: any) => (
    <div data-testid="stacked-area-chart" data-count={props.data?.length} />
  ),
  prepareNetWorthData: (points: any[]) => points.map((p: any) => ({
    date: p.date,
    liquid: parseFloat(p.liquid),
    savings: parseFloat(p.savings),
    investments: parseFloat(p.investments),
    total: parseFloat(p.liquid) + parseFloat(p.savings) + parseFloat(p.investments),
  })),
}))

vi.mock('../components/NetWorthStats', () => ({
  NetWorthStats: (props: any) => (
    <div data-testid="net-worth-stats" data-current={props.stats?.current_net_worth} />
  ),
}))

vi.mock('../components/TimeRangeSelector', () => ({
  TimeRangeSelector: (props: any) => (
    <div data-testid="time-range-selector">
      <button onClick={() => props.onChange(30)}>30d</button>
    </div>
  ),
}))

vi.mock('../hooks/useDarkMode', () => ({
  useDarkMode: () => ({ isDark: false, toggle: vi.fn() }),
}))

const mockGetNetWorth = vi.mocked(client.getNetWorth)

const netWorthResponse: client.NetWorthResponse = {
  points: [
    { date: '2026-03-14', liquid: '5000.00', savings: '20000.00', investments: '50000.00' },
    { date: '2026-03-15', liquid: '5100.00', savings: '20200.00', investments: '50500.00' },
  ],
  stats: {
    current_net_worth: '75800.00',
    period_change_dollars: '800.00',
    period_change_pct: '1.07',
    all_time_high: '75800.00',
    all_time_high_date: '2026-03-15',
  },
}

const emptyResponse: client.NetWorthResponse = {
  points: [],
  stats: null,
}

function renderNetWorth() {
  return render(
    <MemoryRouter>
      <NetWorth />
    </MemoryRouter>
  )
}

describe('NetWorth page', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  it('shows loading skeletons initially', () => {
    mockGetNetWorth.mockReturnValue(new Promise(() => {})) // never resolves
    const { container } = renderNetWorth()
    const skeletons = container.querySelectorAll('.animate-pulse')
    expect(skeletons.length).toBeGreaterThan(0)
  })

  it('renders stats and chart after data loads', async () => {
    mockGetNetWorth.mockResolvedValue(netWorthResponse)
    renderNetWorth()

    await waitFor(() => {
      expect(screen.getByTestId('net-worth-stats')).toBeInTheDocument()
    })
    expect(screen.getByTestId('stacked-area-chart')).toBeInTheDocument()
    expect(screen.getByTestId('time-range-selector')).toBeInTheDocument()
  })

  it('shows empty state when no data', async () => {
    mockGetNetWorth.mockResolvedValue(emptyResponse)
    renderNetWorth()

    await waitFor(() => {
      expect(screen.getByText(/no balance data/i)).toBeInTheDocument()
    })
  })

  it('shows error state with retry button on fetch failure', async () => {
    mockGetNetWorth.mockRejectedValue(new Error('fail'))
    renderNetWorth()

    await waitFor(() => {
      expect(screen.getByText(/something went wrong/i)).toBeInTheDocument()
    })
    expect(screen.getByRole('button', { name: /retry/i })).toBeInTheDocument()
  })

  it('refetches when time range changes', async () => {
    mockGetNetWorth.mockResolvedValue(netWorthResponse)
    renderNetWorth()

    await waitFor(() => {
      expect(screen.getByTestId('net-worth-stats')).toBeInTheDocument()
    })

    // Click 30d in the mocked selector
    await userEvent.click(screen.getByText('30d'))

    // Should have been called again with new days value
    await waitFor(() => {
      expect(mockGetNetWorth).toHaveBeenCalledWith(30)
    })
  })

  it('displays page heading "Net Worth"', async () => {
    mockGetNetWorth.mockResolvedValue(netWorthResponse)
    renderNetWorth()

    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Net Worth' })).toBeInTheDocument()
    })
  })
})
