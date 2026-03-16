import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { PanelCard } from './PanelCard'

const sampleAccounts = [
  { id: '1', name: 'Chase Checking', balance: '1230.50', org_name: '' },
  { id: '2', name: 'Wells Fargo', balance: '3000.00', org_name: '' },
]

describe('PanelCard', () => {
  it('renders the panel label', () => {
    render(
      <PanelCard panelKey="liquid" total="4230.50" accounts={sampleAccounts} />
    )
    expect(screen.getByText('Liquid')).toBeInTheDocument()
  })

  it('renders the formatted total balance', () => {
    render(
      <PanelCard panelKey="liquid" total="4230.50" accounts={sampleAccounts} />
    )
    expect(screen.getByText('$4,230.50')).toBeInTheDocument()
  })

  it('renders all account names', () => {
    render(
      <PanelCard panelKey="liquid" total="4230.50" accounts={sampleAccounts} />
    )
    expect(screen.getByText('Chase Checking')).toBeInTheDocument()
    expect(screen.getByText('Wells Fargo')).toBeInTheDocument()
  })

  it('renders all account balances formatted', () => {
    render(
      <PanelCard panelKey="liquid" total="4230.50" accounts={sampleAccounts} />
    )
    expect(screen.getByText('$1,230.50')).toBeInTheDocument()
    expect(screen.getByText('$3,000.00')).toBeInTheDocument()
  })

  it('renders correctly with empty accounts array', () => {
    render(<PanelCard panelKey="savings" total="15000.00" accounts={[]} />)
    expect(screen.getByText('Savings')).toBeInTheDocument()
    expect(screen.getByText('$15,000.00')).toBeInTheDocument()
  })

  it('renders display_name when set, ignoring org_name', () => {
    const accounts = [
      { id: '1', name: 'Checking 1234', balance: '500.00', org_name: 'Chase', display_name: 'My Checking' },
    ]
    render(
      <PanelCard panelKey="liquid" total="500.00" accounts={accounts} />
    )
    expect(screen.getByText('My Checking')).toBeInTheDocument()
    // Should NOT show "Chase - Checking 1234" format
    expect(screen.queryByText(/Chase/)).not.toBeInTheDocument()
  })

  it('renders "OrgName - Name" when display_name is null', () => {
    const accounts = [
      { id: '1', name: 'Checking 1234', balance: '500.00', org_name: 'Chase', display_name: null },
    ]
    render(
      <PanelCard panelKey="liquid" total="500.00" accounts={accounts} />
    )
    expect(screen.getByText('Chase \u2013 Checking 1234')).toBeInTheDocument()
  })

  // --- Growth Badge tests ---

  it('renders GrowthBadge with green color when growth is positive', () => {
    render(
      <PanelCard
        panelKey="liquid"
        total="4230.50"
        accounts={sampleAccounts}
        pctChange="5.20"
        dollarChange="520.00"
        growthVisible={true}
      />
    )
    const badge = screen.getByText(/5\.2%/)
    expect(badge.className).toContain('text-green-600')
    expect(badge.textContent).toContain('+5.2%')
  })

  it('renders invisible placeholder when growthVisible is false', () => {
    const { container } = render(
      <PanelCard
        panelKey="liquid"
        total="4230.50"
        accounts={sampleAccounts}
        pctChange="5.20"
        dollarChange="520.00"
        growthVisible={false}
      />
    )
    // The GrowthBadge span should be invisible
    const badges = container.querySelectorAll('span.invisible')
    expect(badges.length).toBeGreaterThan(0)
  })

  it('renders invisible placeholder when no growth props provided (backwards compat)', () => {
    const { container } = render(
      <PanelCard panelKey="liquid" total="4230.50" accounts={sampleAccounts} />
    )
    // Without growth props, badge should be invisible placeholder
    const badges = container.querySelectorAll('span.invisible')
    expect(badges.length).toBeGreaterThan(0)
  })

  it('total line uses flex items-baseline layout', () => {
    const { container } = render(
      <PanelCard panelKey="liquid" total="4230.50" accounts={sampleAccounts} />
    )
    const totalLine = container.querySelector('.flex.items-baseline')
    expect(totalLine).not.toBeNull()
  })
})
