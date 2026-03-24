import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import Projections from './Projections'
import * as client from '../api/client'

vi.mock('../api/client')

// Mock child components to isolate page-level logic
vi.mock('../components/ProjectionChart', () => ({
  ProjectionChart: () => <div data-testid="projection-chart">ProjectionChart</div>,
}))

vi.mock('../components/RateConfigTable', () => ({
  RateConfigTable: () => <div data-testid="rate-config-table">RateConfigTable</div>,
}))

vi.mock('../components/IncomeModelingSection', () => ({
  IncomeModelingSection: () => <div data-testid="income-modeling-section">IncomeModelingSection</div>,
}))

vi.mock('../components/HorizonSelector', () => ({
  HorizonSelector: () => <div data-testid="horizon-selector">HorizonSelector</div>,
}))

vi.mock('../hooks/useDarkMode', () => ({
  useDarkMode: () => ({ isDark: false, toggle: vi.fn() }),
}))

const mockGetProjectionSettings = vi.mocked(client.getProjectionSettings)
const mockGetProjectionHistory = vi.mocked(client.getProjectionHistory)

const sampleSettings: client.ProjectionSettingsResponse = {
  accounts: [
    {
      account_id: 'acct-1',
      account_name: 'Checking',
      account_type: 'checking',
      balance: '5000.00',
      apy: '0',
      compound: false,
      included: true,
      holdings: [],
    },
    {
      account_id: 'acct-2',
      account_name: 'Savings',
      account_type: 'savings',
      balance: '20000.00',
      apy: '4.5',
      compound: true,
      included: true,
      holdings: [],
    },
  ],
  income: {
    enabled: false,
    annual_income: '0',
    monthly_savings_pct: '0',
    allocation_json: '{}',
  },
}

const sampleProjectionHistory: client.ProjectionHistoryResponse = {
  points: [
    { date: '2026-01-01', value: '23000.00' },
    { date: '2026-02-01', value: '24100.00' },
    { date: '2026-03-01', value: '25000.00' },
  ],
}

function renderProjections() {
  return render(
    <MemoryRouter>
      <Projections />
    </MemoryRouter>,
  )
}

describe('Projections page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows loading skeleton elements initially', () => {
    // Never resolve the promises to keep loading state
    mockGetProjectionSettings.mockReturnValue(new Promise(() => {}))
    mockGetProjectionHistory.mockReturnValue(new Promise(() => {}))

    renderProjections()

    // Should show page title even while loading
    expect(screen.getByText('Projections')).toBeInTheDocument()
    // Should show animated skeleton elements
    const skeletons = document.querySelectorAll('.animate-pulse')
    expect(skeletons.length).toBeGreaterThanOrEqual(1)
  })

  it('renders page title and all components after fetch resolves', async () => {
    mockGetProjectionSettings.mockResolvedValue(sampleSettings)
    mockGetProjectionHistory.mockResolvedValue(sampleProjectionHistory)

    renderProjections()

    await waitFor(() => {
      expect(screen.getByText('Projections')).toBeInTheDocument()
    })

    expect(screen.getByTestId('horizon-selector')).toBeInTheDocument()
    expect(screen.getByTestId('projection-chart')).toBeInTheDocument()
    expect(screen.getByTestId('rate-config-table')).toBeInTheDocument()
    expect(screen.getByTestId('income-modeling-section')).toBeInTheDocument()
  })

  it('shows empty state when no accounts exist', async () => {
    mockGetProjectionSettings.mockResolvedValue({
      accounts: [],
      income: {
        enabled: false,
        annual_income: '0',
        monthly_savings_pct: '0',
        allocation_json: '{}',
      },
    })
    mockGetProjectionHistory.mockResolvedValue({ points: [] })

    renderProjections()

    await waitFor(() => {
      expect(screen.getByText('No accounts to project')).toBeInTheDocument()
    })
    expect(
      screen.getByText('Sync your financial accounts to start building projections.'),
    ).toBeInTheDocument()
    expect(screen.getByText('Go to Settings')).toBeInTheDocument()
  })

  it('shows error message with retry button on fetch failure', async () => {
    mockGetProjectionSettings.mockRejectedValue(new Error('Network error'))
    mockGetProjectionHistory.mockResolvedValue({ points: [] })

    renderProjections()

    await waitFor(() => {
      expect(
        screen.getByText('Something went wrong loading projection data'),
      ).toBeInTheDocument()
    })
    expect(screen.getByText('Retry')).toBeInTheDocument()
  })

  it('calls getProjectionSettings on mount and getProjectionHistory with included account IDs', async () => {
    mockGetProjectionSettings.mockResolvedValue(sampleSettings)
    mockGetProjectionHistory.mockResolvedValue(sampleProjectionHistory)

    renderProjections()

    await waitFor(() => {
      expect(mockGetProjectionSettings).toHaveBeenCalledTimes(1)
    })

    // getProjectionHistory should be called with 180 days and the included account IDs
    await waitFor(() => {
      expect(mockGetProjectionHistory).toHaveBeenCalledTimes(1)
    })
    expect(mockGetProjectionHistory).toHaveBeenCalledWith(180, ['acct-1', 'acct-2'])
  })

  it('uses only included account IDs for history fetch', async () => {
    const settingsWithOneIncluded: client.ProjectionSettingsResponse = {
      ...sampleSettings,
      accounts: [
        { ...sampleSettings.accounts[0], included: true },
        { ...sampleSettings.accounts[1], included: false },
      ],
    }
    mockGetProjectionSettings.mockResolvedValue(settingsWithOneIncluded)
    mockGetProjectionHistory.mockResolvedValue({ points: [] })

    renderProjections()

    await waitFor(() => {
      expect(mockGetProjectionHistory).toHaveBeenCalledWith(180, ['acct-1'])
    })
  })
})
