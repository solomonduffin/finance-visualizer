import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import AlertRuleCard from './AlertRuleCard'
import type { AlertRule } from '../api/client'

const baseRule: AlertRule = {
  id: 1,
  name: 'Low cash warning',
  operands: [
    { id: 'op1', type: 'bucket', ref: 'liquid', label: 'Liquid Balance', operator: '+' },
  ],
  expression: 'liquid',
  comparison: '<',
  threshold: '5000',
  notify_on_recovery: true,
  enabled: true,
  last_state: 'normal',
  last_eval_at: new Date().toISOString(),
  last_value: '4850.50',
  created_at: '2026-03-15T00:00:00Z',
  updated_at: '2026-03-15T00:00:00Z',
  history: [],
}

describe('AlertRuleCard', () => {
  let mockOnToggle: ReturnType<typeof vi.fn>
  let mockOnEdit: ReturnType<typeof vi.fn>
  let mockOnDelete: ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.clearAllMocks()
    mockOnToggle = vi.fn()
    mockOnEdit = vi.fn()
    mockOnDelete = vi.fn()
  })

  it('renders rule name and expression summary', () => {
    render(
      <AlertRuleCard rule={baseRule} onToggle={mockOnToggle} onEdit={mockOnEdit} onDelete={mockOnDelete} />
    )
    expect(screen.getByText('Low cash warning')).toBeInTheDocument()
    expect(screen.getByText('Liquid Balance < $5,000')).toBeInTheDocument()
  })

  it('renders Normal status badge for normal state', () => {
    render(
      <AlertRuleCard rule={baseRule} onToggle={mockOnToggle} onEdit={mockOnEdit} onDelete={mockOnDelete} />
    )
    expect(screen.getByText('Normal')).toBeInTheDocument()
  })

  it('renders Triggered status badge for triggered state', () => {
    const triggered = { ...baseRule, last_state: 'triggered' as const }
    render(
      <AlertRuleCard rule={triggered} onToggle={mockOnToggle} onEdit={mockOnEdit} onDelete={mockOnDelete} />
    )
    expect(screen.getByText('Triggered')).toBeInTheDocument()
  })

  it('renders Disabled badge when not enabled', () => {
    const disabled = { ...baseRule, enabled: false }
    render(
      <AlertRuleCard rule={disabled} onToggle={mockOnToggle} onEdit={mockOnEdit} onDelete={mockOnDelete} />
    )
    expect(screen.getByText('Disabled')).toBeInTheDocument()
  })

  it('clicking chevron expands history section', async () => {
    const user = userEvent.setup()
    render(
      <AlertRuleCard rule={baseRule} onToggle={mockOnToggle} onEdit={mockOnEdit} onDelete={mockOnDelete} />
    )

    const expandButton = screen.getByRole('button', { name: 'Show alert history' })
    expect(expandButton).toHaveAttribute('aria-expanded', 'false')

    await user.click(expandButton)
    expect(expandButton).toHaveAttribute('aria-expanded', 'true')
    expect(expandButton).toHaveAttribute('aria-label', 'Hide alert history')
  })

  it('shows "No events yet" when history is empty', async () => {
    const user = userEvent.setup()
    render(
      <AlertRuleCard rule={baseRule} onToggle={mockOnToggle} onEdit={mockOnEdit} onDelete={mockOnDelete} />
    )

    await user.click(screen.getByRole('button', { name: 'Show alert history' }))
    expect(screen.getByText('No events yet')).toBeInTheDocument()
  })

  it('calls onToggle when toggle switch clicked', async () => {
    const user = userEvent.setup()
    render(
      <AlertRuleCard rule={baseRule} onToggle={mockOnToggle} onEdit={mockOnEdit} onDelete={mockOnDelete} />
    )

    const toggle = screen.getByRole('switch', { name: 'Disable alert rule' })
    await user.click(toggle)
    expect(mockOnToggle).toHaveBeenCalledWith(1, false)
  })
})
