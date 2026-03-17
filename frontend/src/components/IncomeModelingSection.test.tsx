import { render, screen, fireEvent } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { IncomeModelingSection } from './IncomeModelingSection'

const allocationTargets = [
  { id: 'acct:acc-1', name: 'Chase Checking', percentage: '40' },
  { id: 'acct:acc-2', name: 'Ally Savings', percentage: '60' },
]

const defaultProps = {
  enabled: true,
  annualIncome: '120000',
  monthlySavingsPct: '20',
  allocationTargets,
  onToggle: vi.fn(),
  onAnnualIncomeChange: vi.fn(),
  onMonthlySavingsPctChange: vi.fn(),
  onAllocationChange: vi.fn(),
  isDark: false,
}

describe('IncomeModelingSection', () => {
  it('renders heading "Income Modeling"', () => {
    render(<IncomeModelingSection {...defaultProps} />)
    expect(screen.getByText('Income Modeling')).toBeInTheDocument()
  })

  it('toggle has role="switch" and aria-label="Enable income modeling"', () => {
    render(<IncomeModelingSection {...defaultProps} />)
    const toggle = screen.getByRole('switch', { name: 'Enable income modeling' })
    expect(toggle).toBeInTheDocument()
    expect(toggle).toHaveAttribute('aria-checked', 'true')
  })

  it('clicking toggle calls onToggle with inverted value', () => {
    const onToggle = vi.fn()
    render(<IncomeModelingSection {...defaultProps} onToggle={onToggle} />)
    const toggle = screen.getByRole('switch', { name: 'Enable income modeling' })
    fireEvent.click(toggle)
    expect(onToggle).toHaveBeenCalledWith(false)
  })

  it('expanded section shows income fields', () => {
    render(<IncomeModelingSection {...defaultProps} />)
    // Expand the section
    const expandButton = screen.getByRole('button')
    fireEvent.click(expandButton)

    expect(screen.getByText('Annual Income')).toBeInTheDocument()
    expect(screen.getByText('Monthly Savings')).toBeInTheDocument()
    expect(screen.getByText('Savings Allocation')).toBeInTheDocument()
  })

  it('allocation rows render for each target', () => {
    render(<IncomeModelingSection {...defaultProps} />)
    // Expand
    const expandButton = screen.getByRole('button')
    fireEvent.click(expandButton)

    expect(screen.getByText('Chase Checking')).toBeInTheDocument()
    expect(screen.getByText('Ally Savings')).toBeInTheDocument()
  })

  it('allocation sum 100% shows green "Total: 100%"', () => {
    render(<IncomeModelingSection {...defaultProps} />)
    // Expand
    const expandButton = screen.getByRole('button')
    fireEvent.click(expandButton)

    // 40 + 60 = 100
    const totalEl = screen.getByText('Total: 100%')
    expect(totalEl).toBeInTheDocument()
    expect(totalEl.className).toContain('text-green-600')
  })

  it('allocation sum != 100 shows red error message with role="status"', () => {
    const targets = [
      { id: 'acct:acc-1', name: 'Chase Checking', percentage: '30' },
      { id: 'acct:acc-2', name: 'Ally Savings', percentage: '20' },
    ]
    render(
      <IncomeModelingSection {...defaultProps} allocationTargets={targets} />
    )
    // Expand
    const expandButton = screen.getByRole('button')
    fireEvent.click(expandButton)

    // 30 + 20 = 50
    const statusEl = screen.getByRole('status')
    expect(statusEl).toBeInTheDocument()
    const errorText = screen.getByText(/Total: 50%/)
    expect(errorText).toBeInTheDocument()
    expect(errorText.className).toContain('text-red-600')
  })

  it('when enabled=false and expanded, content has opacity-50', () => {
    const { container } = render(
      <IncomeModelingSection {...defaultProps} enabled={false} />
    )
    // Expand
    const expandButton = screen.getByRole('button')
    fireEvent.click(expandButton)

    // The content area should have opacity-50 when disabled
    const contentArea = container.querySelector('.opacity-50')
    expect(contentArea).not.toBeNull()
  })

  it('when enabled=false and collapsed, card has opacity-60', () => {
    const { container } = render(
      <IncomeModelingSection {...defaultProps} enabled={false} />
    )
    // Not expanded by default -- card should be dimmed
    const card = container.firstElementChild as HTMLElement
    expect(card.className).toContain('opacity-60')
  })

  it('displays computed monthly allocation', () => {
    render(<IncomeModelingSection {...defaultProps} />)
    // Expand
    const expandButton = screen.getByRole('button')
    fireEvent.click(expandButton)

    // 120000 / 12 * 20 / 100 = 2000
    expect(screen.getByText(/\$2,000\.00 \/ month to allocate/)).toBeInTheDocument()
  })

  it('has motion-reduce:transition-none on expandable content', () => {
    const { container } = render(<IncomeModelingSection {...defaultProps} />)
    const expandableDiv = container.querySelector('#income-modeling-content')
    expect(expandableDiv).not.toBeNull()
    expect(expandableDiv!.className).toContain('motion-reduce:transition-none')
  })

  it('displays "Must total 100%." helper text', () => {
    render(<IncomeModelingSection {...defaultProps} />)
    // Expand
    const expandButton = screen.getByRole('button')
    fireEvent.click(expandButton)

    expect(screen.getByText(/Must total 100%/)).toBeInTheDocument()
  })
})
