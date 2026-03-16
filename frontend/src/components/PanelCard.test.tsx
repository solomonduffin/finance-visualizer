import { render, screen, fireEvent } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { PanelCard } from './PanelCard'
import type { GroupItem, GroupGrowthData } from '../api/client'

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

  // --- Group rendering tests ---

  const sampleGroups: GroupItem[] = [
    {
      id: 1,
      name: 'Main Bank',
      panel_type: 'checking',
      total_balance: '4000.00',
      members: [
        { id: 'm1', name: 'Checking', original_name: 'Checking', balance: '2500.00', currency: 'USD', org_name: 'Chase', display_name: null, account_type_override: null },
        { id: 'm2', name: 'Credit', original_name: 'Credit', balance: '1500.00', currency: 'USD', org_name: 'Chase', display_name: null, account_type_override: null },
      ],
    },
  ]

  const sampleGroupGrowth: GroupGrowthData[] = [
    { group_id: 1, name: 'Main Bank', growth: { current_total: '4000.00', prior_total: '3800.00', dollar_change: '200.00', pct_change: '5.26' } },
  ]

  it('renders GroupRow for each group in the groups prop', () => {
    render(
      <PanelCard panelKey="liquid" total="8230.50" accounts={sampleAccounts} groups={sampleGroups} />
    )
    expect(screen.getByText('Main Bank')).toBeInTheDocument()
    expect(screen.getByText('$4,000.00')).toBeInTheDocument()
  })

  it('group rows appear before standalone accounts', () => {
    const { container } = render(
      <PanelCard panelKey="liquid" total="8230.50" accounts={sampleAccounts} groups={sampleGroups} />
    )
    const allText = container.textContent ?? ''
    const groupIdx = allText.indexOf('Main Bank')
    const accountIdx = allText.indexOf('Chase Checking')
    expect(groupIdx).toBeLessThan(accountIdx)
  })

  it('clicking a GroupRow toggles expansion', () => {
    render(
      <PanelCard panelKey="liquid" total="8230.50" accounts={sampleAccounts} groups={sampleGroups} />
    )
    const groupButton = screen.getByRole('button')
    expect(groupButton).toHaveAttribute('aria-expanded', 'false')
    fireEvent.click(groupButton)
    expect(groupButton).toHaveAttribute('aria-expanded', 'true')
  })

  it('renders without groups when groups prop not provided (backward compatible)', () => {
    render(
      <PanelCard panelKey="liquid" total="4230.50" accounts={sampleAccounts} />
    )
    expect(screen.getByText('Liquid')).toBeInTheDocument()
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('passes group growth data to GroupRow', () => {
    render(
      <PanelCard
        panelKey="liquid"
        total="8230.50"
        accounts={sampleAccounts}
        groups={sampleGroups}
        groupGrowth={sampleGroupGrowth}
        growthVisible={true}
      />
    )
    expect(screen.getByText(/5\.3%/)).toBeInTheDocument()
  })
})
