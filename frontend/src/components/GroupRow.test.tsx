import { render, screen, fireEvent } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { GroupRow } from './GroupRow'
import type { GroupItem } from '../api/client'

const mockGroup: GroupItem = {
  id: 1,
  name: 'Main Bank',
  panel_type: 'checking',
  total_balance: '5500.00',
  members: [
    {
      id: 'acc-1',
      name: 'Checking',
      original_name: 'Checking',
      balance: '3000.00',
      currency: 'USD',
      org_name: 'Chase',
      display_name: null,
      account_type_override: null,
    },
    {
      id: 'acc-2',
      name: 'Savings',
      original_name: 'Savings',
      balance: '2500.00',
      currency: 'USD',
      org_name: 'Chase',
      display_name: null,
      account_type_override: null,
    },
  ],
}

describe('GroupRow', () => {
  it('renders group name in font-semibold and summed balance', () => {
    render(<GroupRow group={mockGroup} expanded={false} onToggle={() => {}} />)
    const name = screen.getByText('Main Bank')
    expect(name.className).toContain('font-semibold')
    expect(screen.getByText('$5,500.00')).toBeInTheDocument()
  })

  it('collapsed by default — member accounts not visible (maxHeight 0)', () => {
    const { container } = render(<GroupRow group={mockGroup} expanded={false} onToggle={() => {}} />)
    const membersDiv = container.querySelector('#group-members-1')
    expect(membersDiv).not.toBeNull()
    expect(membersDiv!.getAttribute('style')).toContain('max-height: 0px')
  })

  it('click toggles expansion — onToggle callback called', () => {
    const onToggle = vi.fn()
    render(<GroupRow group={mockGroup} expanded={false} onToggle={onToggle} />)
    const button = screen.getByRole('button')
    fireEvent.click(button)
    expect(onToggle).toHaveBeenCalledTimes(1)
  })

  it('expanded state shows member accounts indented with pl-6', () => {
    const { container } = render(<GroupRow group={mockGroup} expanded={true} onToggle={() => {}} />)
    const membersDiv = container.querySelector('#group-members-1')
    expect(membersDiv!.getAttribute('style')).not.toContain('max-height: 0px')
    expect(screen.getByText('Chase \u2013 Checking')).toBeInTheDocument()
    expect(screen.getByText('Chase \u2013 Savings')).toBeInTheDocument()
    const indented = container.querySelector('.pl-6')
    expect(indented).not.toBeNull()
  })

  it('shows GrowthBadge when growth data provided and visible=true', () => {
    render(
      <GroupRow
        group={mockGroup}
        expanded={false}
        onToggle={() => {}}
        pctChange="3.50"
        dollarChange="200.00"
        growthVisible={true}
      />
    )
    expect(screen.getByText(/3\.5%/)).toBeInTheDocument()
  })

  it('has role="button", tabIndex=0, and aria-expanded attribute', () => {
    render(<GroupRow group={mockGroup} expanded={false} onToggle={() => {}} />)
    const button = screen.getByRole('button')
    expect(button).toHaveAttribute('tabindex', '0')
    expect(button).toHaveAttribute('aria-expanded', 'false')
  })

  it('chevron rotates 90 degrees when expanded (rotate-90 class present)', () => {
    const { container } = render(<GroupRow group={mockGroup} expanded={true} onToggle={() => {}} />)
    const svg = container.querySelector('svg')
    expect(svg!.className.baseVal).toContain('rotate-90')
  })
})
