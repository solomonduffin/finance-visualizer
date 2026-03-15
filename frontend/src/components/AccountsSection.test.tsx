import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import AccountsSection from './AccountsSection'
import * as client from '../api/client'

vi.mock('../api/client')

const mockGetAccounts = vi.mocked(client.getAccounts)
const mockUpdateAccount = vi.mocked(client.updateAccount)

function makeAccount(overrides: Partial<client.AccountItem> = {}): client.AccountItem {
  return {
    id: 'acc-1',
    name: 'Checking Account',
    original_name: 'Checking Account',
    balance: '1500.00',
    account_type: 'checking',
    org_name: 'Chase',
    display_name: null,
    hidden_at: null,
    account_type_override: null,
    ...overrides,
  }
}

const defaultResponse: client.AccountsResponse = {
  liquid: [
    makeAccount({ id: 'acc-1', name: 'Checking', original_name: 'Checking', org_name: 'Chase', balance: '1500.00' }),
    makeAccount({ id: 'acc-2', name: 'Savings Acct', original_name: 'Savings Acct', org_name: 'Chase', balance: '5000.00' }),
  ],
  savings: [
    makeAccount({ id: 'acc-3', name: 'High Yield', original_name: 'High Yield', org_name: 'Ally', balance: '10000.00', account_type: 'savings' }),
  ],
  investments: [
    makeAccount({ id: 'acc-4', name: '401k', original_name: '401k', org_name: 'Fidelity', balance: '50000.00', account_type: 'investment' }),
  ],
  other: [],
}

const responseWithHidden: client.AccountsResponse = {
  liquid: [
    makeAccount({ id: 'acc-1', name: 'Checking', original_name: 'Checking', org_name: 'Chase' }),
    makeAccount({ id: 'acc-hidden', name: 'Old Account', original_name: 'Old Account', org_name: 'Bank', hidden_at: '2026-03-10T00:00:00Z', balance: '0.00' }),
  ],
  savings: [],
  investments: [],
  other: [],
}

const responseWithDisplayName: client.AccountsResponse = {
  liquid: [
    makeAccount({
      id: 'acc-1',
      name: 'My Checking',
      original_name: 'Checking Account #1234',
      org_name: 'Chase',
      display_name: 'My Checking',
    }),
  ],
  savings: [],
  investments: [],
  other: [],
}

describe('AccountsSection', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    // Default matchMedia mock for desktop
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: vi.fn().mockImplementation((query: string) => ({
        matches: false, // desktop by default
        media: query,
        onchange: null,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
        addListener: vi.fn(),
        removeListener: vi.fn(),
      })),
    })
  })

  it('renders account groups with correct headings', async () => {
    mockGetAccounts.mockResolvedValue(defaultResponse)
    render(<AccountsSection />)

    await waitFor(() => {
      expect(screen.getByText('Liquid')).toBeInTheDocument()
      expect(screen.getByText('Savings')).toBeInTheDocument()
      expect(screen.getByText('Investments')).toBeInTheDocument()
    })
  })

  it('renders account names using getAccountDisplayName', async () => {
    mockGetAccounts.mockResolvedValue(defaultResponse)
    render(<AccountsSection />)

    await waitFor(() => {
      // org_name + name pattern: "Chase - Checking"
      expect(screen.getByText('Chase \u2013 Checking')).toBeInTheDocument()
      expect(screen.getByText('Ally \u2013 High Yield')).toBeInTheDocument()
      expect(screen.getByText('Fidelity \u2013 401k')).toBeInTheDocument()
    })
  })

  it('shows account counts per group', async () => {
    mockGetAccounts.mockResolvedValue(defaultResponse)
    render(<AccountsSection />)

    await waitFor(() => {
      expect(screen.getByText('(2)')).toBeInTheDocument() // liquid
      expect(screen.getAllByText('(1)')).toHaveLength(2)   // savings + investments
    })
  })

  it('clicking pencil icon shows edit input field', async () => {
    mockGetAccounts.mockResolvedValue(defaultResponse)
    const user = userEvent.setup()
    render(<AccountsSection />)

    await waitFor(() => screen.getByText('Chase \u2013 Checking'))

    const editButtons = screen.getAllByRole('button', { name: /edit name/i })
    await user.click(editButtons[0])

    expect(screen.getByLabelText('Edit account name')).toBeInTheDocument()
  })

  it('pressing Enter in edit input calls updateAccount with display_name', async () => {
    mockGetAccounts.mockResolvedValue(defaultResponse)
    mockUpdateAccount.mockResolvedValue(
      makeAccount({ id: 'acc-1', display_name: 'My Chase Checking', name: 'My Chase Checking' })
    )
    const user = userEvent.setup()
    render(<AccountsSection />)

    await waitFor(() => screen.getByText('Chase \u2013 Checking'))

    const editButtons = screen.getAllByRole('button', { name: /edit name/i })
    await user.click(editButtons[0])

    const input = screen.getByLabelText('Edit account name')
    await user.type(input, 'My Chase Checking')
    await user.keyboard('{Enter}')

    await waitFor(() => {
      expect(mockUpdateAccount).toHaveBeenCalledWith('acc-1', {
        display_name: 'My Chase Checking',
      })
    })
  })

  it('pressing Escape in edit input reverts without API call', async () => {
    mockGetAccounts.mockResolvedValue(defaultResponse)
    const user = userEvent.setup()
    render(<AccountsSection />)

    await waitFor(() => screen.getByText('Chase \u2013 Checking'))

    const editButtons = screen.getAllByRole('button', { name: /edit name/i })
    await user.click(editButtons[0])

    const input = screen.getByLabelText('Edit account name')
    await user.type(input, 'Some name')
    await user.keyboard('{Escape}')

    expect(mockUpdateAccount).not.toHaveBeenCalled()
    // Input should be gone
    expect(screen.queryByLabelText('Edit account name')).not.toBeInTheDocument()
  })

  it('clicking hide button calls updateAccount with hidden=true', async () => {
    mockGetAccounts.mockResolvedValue(defaultResponse)
    mockUpdateAccount.mockResolvedValue(
      makeAccount({ id: 'acc-1', hidden_at: '2026-03-15T00:00:00Z' })
    )
    const user = userEvent.setup()
    render(<AccountsSection />)

    await waitFor(() => screen.getByText('Chase \u2013 Checking'))

    const hideButtons = screen.getAllByRole('button', { name: /hide account/i })
    await user.click(hideButtons[0])

    await waitFor(() => {
      expect(mockUpdateAccount).toHaveBeenCalledWith('acc-1', { hidden: true })
    })
  })

  it('hidden accounts appear in collapsible section', async () => {
    mockGetAccounts.mockResolvedValue(responseWithHidden)
    const user = userEvent.setup()
    render(<AccountsSection />)

    await waitFor(() => screen.getByText(/hidden accounts/i))

    // Section should be collapsed by default
    expect(screen.queryByTestId('hidden-accounts')).not.toBeInTheDocument()

    // Expand hidden accounts section
    await user.click(screen.getByText(/hidden accounts/i))

    await waitFor(() => {
      expect(screen.getByTestId('hidden-accounts')).toBeInTheDocument()
      expect(screen.getByTestId('hidden-account-acc-hidden')).toBeInTheDocument()
    })
  })

  it('clicking Unhide calls updateAccount with hidden=false', async () => {
    mockGetAccounts.mockResolvedValue(responseWithHidden)
    mockUpdateAccount.mockResolvedValue(
      makeAccount({ id: 'acc-hidden', hidden_at: null })
    )
    const user = userEvent.setup()
    render(<AccountsSection />)

    await waitFor(() => screen.getByText(/hidden accounts/i))

    // Expand hidden section
    await user.click(screen.getByText(/hidden accounts/i))

    await waitFor(() => screen.getByTestId('hidden-accounts'))

    const unhideBtn = screen.getByRole('button', { name: /unhide account/i })
    await user.click(unhideBtn)

    await waitFor(() => {
      expect(mockUpdateAccount).toHaveBeenCalledWith('acc-hidden', { hidden: false })
    })
  })

  it('reset button calls updateAccount with display_name=null', async () => {
    mockGetAccounts.mockResolvedValue(responseWithDisplayName)
    mockUpdateAccount.mockResolvedValue(
      makeAccount({ id: 'acc-1', display_name: null, name: 'Checking Account #1234' })
    )
    const user = userEvent.setup()
    render(<AccountsSection />)

    await waitFor(() => screen.getByText('My Checking'))

    const resetBtn = screen.getByRole('button', { name: /reset name/i })
    await user.click(resetBtn)

    await waitFor(() => {
      expect(mockUpdateAccount).toHaveBeenCalledWith('acc-1', { display_name: null })
    })
  })

  it('shows original SimpleFIN name as subtitle when display_name is set', async () => {
    mockGetAccounts.mockResolvedValue(responseWithDisplayName)
    render(<AccountsSection />)

    await waitFor(() => {
      expect(screen.getByText('My Checking')).toBeInTheDocument()
      expect(screen.getByText('Checking Account #1234')).toBeInTheDocument()
    })
  })

  it('shows mobile type dropdown when viewport is narrow', async () => {
    // Mock matchMedia for mobile
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: vi.fn().mockImplementation((query: string) => ({
        matches: true, // mobile
        media: query,
        onchange: null,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
        addListener: vi.fn(),
        removeListener: vi.fn(),
      })),
    })

    mockGetAccounts.mockResolvedValue(defaultResponse)
    render(<AccountsSection />)

    await waitFor(() => {
      const dropdowns = screen.getAllByRole('combobox', { name: /account type/i })
      expect(dropdowns.length).toBeGreaterThan(0)
    })
  })

  it('returns null when no accounts exist', async () => {
    mockGetAccounts.mockResolvedValue({ liquid: [], savings: [], investments: [], other: [] })
    const { container } = render(<AccountsSection />)

    await waitFor(() => {
      expect(container.querySelector('[data-testid="accounts-section"]')).not.toBeInTheDocument()
    })
  })
})
