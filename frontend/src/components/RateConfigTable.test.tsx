import { render, screen, fireEvent } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { RateConfigTable } from './RateConfigTable'
import type { ProjectionAccountSetting } from '../api/client'

const mockAccounts: ProjectionAccountSetting[] = [
  {
    account_id: 'acc-checking',
    account_name: 'Chase Checking',
    account_type: 'checking',
    balance: '5000.00',
    apy: '0.01',
    compound: false,
    included: true,
    holdings: [],
  },
  {
    account_id: 'acc-savings',
    account_name: 'Ally Savings',
    account_type: 'savings',
    balance: '10000.00',
    apy: '4.5',
    compound: true,
    included: true,
    holdings: [],
  },
  {
    account_id: 'acc-brokerage',
    account_name: 'Fidelity Brokerage',
    account_type: 'brokerage',
    balance: '25000.00',
    apy: '0',
    compound: true,
    included: true,
    holdings: [
      {
        holding_id: 'h1',
        symbol: 'AAPL',
        description: 'Apple Inc.',
        market_value: '15000.00',
        apy: '7.0',
        compound: true,
        included: true,
        allocation: '60',
      },
      {
        holding_id: 'h2',
        symbol: 'GOOG',
        description: 'Alphabet Inc.',
        market_value: '10000.00',
        apy: '5.0',
        compound: false,
        included: true,
        allocation: '40',
      },
    ],
  },
  {
    account_id: 'acc-retirement',
    account_name: 'Vanguard 401k',
    account_type: 'retirement',
    balance: '50000.00',
    apy: '6.0',
    compound: true,
    included: true,
    holdings: [],
  },
]

const defaultProps = {
  accounts: mockAccounts,
  onApyChange: vi.fn(),
  onHoldingApyChange: vi.fn(),
  onCompoundChange: vi.fn(),
  onHoldingCompoundChange: vi.fn(),
  onIncludeChange: vi.fn(),
  onHoldingIncludeChange: vi.fn(),
  isDark: false,
}

describe('RateConfigTable', () => {
  it('renders grouped accounts under correct panel headers', () => {
    render(<RateConfigTable {...defaultProps} />)
    expect(screen.getByText('Liquid')).toBeInTheDocument()
    expect(screen.getByText('Savings')).toBeInTheDocument()
    expect(screen.getByText('Investments')).toBeInTheDocument()
  })

  it('renders "Projection Settings" heading', () => {
    render(<RateConfigTable {...defaultProps} />)
    expect(screen.getByText('Projection Settings')).toBeInTheDocument()
  })

  it('renders APY inputs with correct aria-label', () => {
    render(<RateConfigTable {...defaultProps} />)
    expect(screen.getAllByLabelText('APY for Chase Checking').length).toBeGreaterThan(0)
    expect(screen.getAllByLabelText('APY for Ally Savings').length).toBeGreaterThan(0)
  })

  it('compound toggle has role="switch" and aria-checked', () => {
    render(<RateConfigTable {...defaultProps} />)
    const switches = screen.getAllByRole('switch')
    expect(switches.length).toBeGreaterThan(0)
    // Ally Savings has compound=true
    const allySavingsToggle = screen.getAllByLabelText('Compound interest for Ally Savings')
    expect(allySavingsToggle.length).toBeGreaterThan(0)
    expect(allySavingsToggle[0]).toHaveAttribute('aria-checked', 'true')
  })

  it('include checkbox has correct aria-label', () => {
    render(<RateConfigTable {...defaultProps} />)
    const checkboxes = screen.getAllByLabelText('Include Chase Checking in projection')
    expect(checkboxes.length).toBeGreaterThan(0)
    expect(checkboxes[0]).toBeChecked()
  })

  it('investment account with holdings shows chevron and expand button', () => {
    render(<RateConfigTable {...defaultProps} />)
    // Fidelity Brokerage has holdings -- should have role="button" with aria-expanded
    const expandButtons = screen.getAllByRole('button', { name: /Fidelity Brokerage/i })
    // The expand button area includes the account name text
    expect(expandButtons.length).toBeGreaterThan(0)
  })

  it('clicking chevron expands holdings rows', () => {
    const { container } = render(<RateConfigTable {...defaultProps} />)
    // Initially collapsed
    const holdingsDiv = container.querySelector('#holdings-acc-brokerage')
    expect(holdingsDiv).not.toBeNull()
    expect(holdingsDiv!.getAttribute('style')).toContain('max-height: 0px')

    // Click expand button
    const expandButtons = screen.getAllByRole('button')
    // Find the expand button for the brokerage account
    const brokerageButton = expandButtons.find(
      (btn) => btn.getAttribute('aria-controls') === 'holdings-acc-brokerage'
    )
    expect(brokerageButton).toBeDefined()
    fireEvent.click(brokerageButton!)

    // Now holdings should be expanded
    expect(holdingsDiv!.getAttribute('style')).not.toContain('max-height: 0px')
  })

  it('investment account without holdings renders as standard row', () => {
    render(<RateConfigTable {...defaultProps} />)
    // Vanguard 401k has no holdings - should have APY input
    const apyInputs = screen.getAllByLabelText('APY for Vanguard 401k')
    expect(apyInputs.length).toBeGreaterThan(0)
    // Should also have compound toggle
    const compoundToggles = screen.getAllByLabelText('Compound interest for Vanguard 401k')
    expect(compoundToggles.length).toBeGreaterThan(0)
  })

  it('empty accounts array shows "No accounts found" message', () => {
    render(<RateConfigTable {...defaultProps} accounts={[]} />)
    expect(screen.getByText('No accounts found. Sync your accounts first.')).toBeInTheDocument()
  })

  it('has grid-cols-[1fr_80px_80px_64px] for desktop layout', () => {
    const { container } = render(<RateConfigTable {...defaultProps} />)
    const gridElements = container.querySelectorAll('.grid-cols-\\[1fr_80px_80px_64px\\]')
    expect(gridElements.length).toBeGreaterThan(0)
  })

  it('master include checkbox for investment account with holdings cascades', () => {
    const onIncludeChange = vi.fn()
    const onHoldingIncludeChange = vi.fn()
    render(
      <RateConfigTable
        {...defaultProps}
        onIncludeChange={onIncludeChange}
        onHoldingIncludeChange={onHoldingIncludeChange}
      />
    )
    // Find the include checkbox for the brokerage account
    const checkboxes = screen.getAllByLabelText('Include Fidelity Brokerage in projection')
    fireEvent.click(checkboxes[0])
    expect(onIncludeChange).toHaveBeenCalledWith('acc-brokerage', false)
    // Should cascade to holdings
    expect(onHoldingIncludeChange).toHaveBeenCalledWith('h1', false)
    expect(onHoldingIncludeChange).toHaveBeenCalledWith('h2', false)
  })
})
