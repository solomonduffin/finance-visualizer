import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import Dashboard from './Dashboard'
import * as client from '../api/client'

vi.mock('../api/client')

vi.mock('../components/BalanceLineChart', () => ({
  BalanceLineChart: (props: any) => (
    <div data-testid="balance-line-chart" data-has-history={!!props.history} />
  ),
}))

vi.mock('../components/NetWorthDonut', () => ({
  NetWorthDonut: (props: any) => (
    <div
      data-testid="net-worth-donut"
      data-liquid={props.liquid}
      data-savings={props.savings}
      data-investments={props.investments}
    />
  ),
}))

vi.mock('../hooks/useDarkMode', () => ({
  useDarkMode: () => ({ isDark: false, toggle: vi.fn() }),
}))

const mockGetSummary = vi.mocked(client.getSummary)
const mockGetAccounts = vi.mocked(client.getAccounts)
const mockGetBalanceHistory = vi.mocked(client.getBalanceHistory)

const summaryWithSync: client.SummaryResponse = {
  liquid: '5000.00',
  savings: '20000.00',
  investments: '50000.00',
  last_synced_at: '2026-03-15T12:00:00Z',
}

const summaryNoSync: client.SummaryResponse = {
  liquid: '0.00',
  savings: '0.00',
  investments: '0.00',
  last_synced_at: null,
}

const accountsResponse: client.AccountsResponse = {
  liquid: [{ id: '1', name: 'Chase Checking', balance: '5000.00', account_type: 'checking' }],
  savings: [{ id: '2', name: 'Ally Savings', balance: '20000.00', account_type: 'savings' }],
  investments: [{ id: '3', name: 'Vanguard 401k', balance: '50000.00', account_type: 'investment' }],
  other: [],
}

const accountsEmptySavings: client.AccountsResponse = {
  liquid: [{ id: '1', name: 'Chase Checking', balance: '5000.00', account_type: 'checking' }],
  savings: [],
  investments: [{ id: '3', name: 'Vanguard 401k', balance: '50000.00', account_type: 'investment' }],
  other: [],
}

const historyResponse: client.BalanceHistoryResponse = {
  liquid: [{ date: '2026-03-14', balance: '4900.00' }, { date: '2026-03-15', balance: '5000.00' }],
  savings: [{ date: '2026-03-14', balance: '19900.00' }, { date: '2026-03-15', balance: '20000.00' }],
  investments: [{ date: '2026-03-14', balance: '49000.00' }, { date: '2026-03-15', balance: '50000.00' }],
}

function renderDashboard() {
  return render(
    <MemoryRouter>
      <Dashboard />
    </MemoryRouter>
  )
}

describe('Dashboard', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  describe('loading state', () => {
    it('renders SkeletonDashboard while API calls are in flight', () => {
      // Return never-resolving promises to keep loading state
      mockGetSummary.mockReturnValue(new Promise(() => {}))
      mockGetAccounts.mockReturnValue(new Promise(() => {}))
      mockGetBalanceHistory.mockReturnValue(new Promise(() => {}))

      renderDashboard()

      // SkeletonDashboard renders 3 skeleton cards
      expect(screen.getAllByTestId('skeleton-card')).toHaveLength(3)
    })
  })

  describe('empty state', () => {
    it('renders EmptyState when last_synced_at is null', async () => {
      mockGetSummary.mockResolvedValue(summaryNoSync)
      mockGetAccounts.mockResolvedValue(accountsResponse)
      mockGetBalanceHistory.mockResolvedValue(historyResponse)

      renderDashboard()

      await waitFor(() => {
        expect(screen.getByText(/connect your accounts/i)).toBeInTheDocument()
      })
    })
  })

  describe('error state', () => {
    it('renders error message when API call fails', async () => {
      mockGetSummary.mockRejectedValue(new Error('Network error'))
      mockGetAccounts.mockResolvedValue(accountsResponse)
      mockGetBalanceHistory.mockResolvedValue(historyResponse)

      renderDashboard()

      await waitFor(() => {
        expect(screen.getByText(/something went wrong/i)).toBeInTheDocument()
      })
    })

    it('renders a Retry button on error', async () => {
      mockGetSummary.mockRejectedValue(new Error('Network error'))
      mockGetAccounts.mockResolvedValue(accountsResponse)
      mockGetBalanceHistory.mockResolvedValue(historyResponse)

      renderDashboard()

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /retry/i })).toBeInTheDocument()
      })
    })

    it('re-fetches data when Retry is clicked', async () => {
      mockGetSummary
        .mockRejectedValueOnce(new Error('Network error'))
        .mockResolvedValueOnce(summaryWithSync)
      mockGetAccounts.mockResolvedValue(accountsResponse)
      mockGetBalanceHistory.mockResolvedValue(historyResponse)

      renderDashboard()

      // Wait for error state
      const retryButton = await screen.findByRole('button', { name: /retry/i })

      // Click retry
      await userEvent.click(retryButton)

      // Should now show dashboard data
      await waitFor(() => {
        expect(screen.getByText('Dashboard')).toBeInTheDocument()
      })
    })
  })

  describe('data state', () => {
    it('renders PanelCards for each non-empty panel', async () => {
      mockGetSummary.mockResolvedValue(summaryWithSync)
      mockGetAccounts.mockResolvedValue(accountsResponse)
      mockGetBalanceHistory.mockResolvedValue(historyResponse)

      renderDashboard()

      await waitFor(() => {
        expect(screen.getByText('Liquid')).toBeInTheDocument()
        expect(screen.getByText('Savings')).toBeInTheDocument()
        expect(screen.getByText('Investments')).toBeInTheDocument()
      })
    })

    it('shows freshness indicator "Last updated" when data is loaded', async () => {
      mockGetSummary.mockResolvedValue(summaryWithSync)
      mockGetAccounts.mockResolvedValue(accountsResponse)
      mockGetBalanceHistory.mockResolvedValue(historyResponse)

      renderDashboard()

      await waitFor(() => {
        expect(screen.getByText(/last updated/i)).toBeInTheDocument()
      })
    })

    it('hides panel when accounts array is empty (no savings panel)', async () => {
      mockGetSummary.mockResolvedValue(summaryWithSync)
      mockGetAccounts.mockResolvedValue(accountsEmptySavings)
      mockGetBalanceHistory.mockResolvedValue(historyResponse)

      renderDashboard()

      await waitFor(() => {
        expect(screen.getByText('Liquid')).toBeInTheDocument()
        expect(screen.getByText('Investments')).toBeInTheDocument()
        expect(screen.queryByText('Savings')).not.toBeInTheDocument()
      })
    })

    it('passes correct total and accounts to each PanelCard', async () => {
      mockGetSummary.mockResolvedValue(summaryWithSync)
      mockGetAccounts.mockResolvedValue(accountsResponse)
      mockGetBalanceHistory.mockResolvedValue(historyResponse)

      renderDashboard()

      await waitFor(() => {
        // Liquid panel: account name rendered
        expect(screen.getByText('Chase Checking')).toBeInTheDocument()
        // Savings panel: account name rendered
        expect(screen.getByText('Ally Savings')).toBeInTheDocument()
        // Investments panel: account name rendered
        expect(screen.getByText('Vanguard 401k')).toBeInTheDocument()
        // All panel labels visible
        expect(screen.getByText('Liquid')).toBeInTheDocument()
        expect(screen.getByText('Savings')).toBeInTheDocument()
        expect(screen.getByText('Investments')).toBeInTheDocument()
      })
    })
  })

  describe('charts section', () => {
    it('renders BalanceLineChart when data is loaded', async () => {
      mockGetSummary.mockResolvedValue(summaryWithSync)
      mockGetAccounts.mockResolvedValue(accountsResponse)
      mockGetBalanceHistory.mockResolvedValue(historyResponse)

      renderDashboard()

      await waitFor(() => {
        expect(screen.getByTestId('balance-line-chart')).toBeInTheDocument()
      })
    })

    it('renders NetWorthDonut when data is loaded', async () => {
      mockGetSummary.mockResolvedValue(summaryWithSync)
      mockGetAccounts.mockResolvedValue(accountsResponse)
      mockGetBalanceHistory.mockResolvedValue(historyResponse)

      renderDashboard()

      await waitFor(() => {
        expect(screen.getByTestId('net-worth-donut')).toBeInTheDocument()
      })
    })

    it('passes summary values to NetWorthDonut', async () => {
      mockGetSummary.mockResolvedValue(summaryWithSync)
      mockGetAccounts.mockResolvedValue(accountsResponse)
      mockGetBalanceHistory.mockResolvedValue(historyResponse)

      renderDashboard()

      await waitFor(() => {
        const donut = screen.getByTestId('net-worth-donut')
        expect(donut.getAttribute('data-liquid')).toBe('5000.00')
        expect(donut.getAttribute('data-savings')).toBe('20000.00')
        expect(donut.getAttribute('data-investments')).toBe('50000.00')
      })
    })

    it('passes history data to BalanceLineChart', async () => {
      mockGetSummary.mockResolvedValue(summaryWithSync)
      mockGetAccounts.mockResolvedValue(accountsResponse)
      mockGetBalanceHistory.mockResolvedValue(historyResponse)

      renderDashboard()

      await waitFor(() => {
        const lineChart = screen.getByTestId('balance-line-chart')
        expect(lineChart.getAttribute('data-has-history')).toBe('true')
      })
    })

    it('does not render charts in loading state', () => {
      mockGetSummary.mockReturnValue(new Promise(() => {}))
      mockGetAccounts.mockReturnValue(new Promise(() => {}))
      mockGetBalanceHistory.mockReturnValue(new Promise(() => {}))

      renderDashboard()

      expect(screen.queryByTestId('balance-line-chart')).not.toBeInTheDocument()
      expect(screen.queryByTestId('net-worth-donut')).not.toBeInTheDocument()
    })
  })
})
