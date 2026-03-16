import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import DashboardPreferences from './DashboardPreferences'

describe('DashboardPreferences', () => {
  let mockOnToggle: ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.clearAllMocks()
    mockOnToggle = vi.fn().mockResolvedValue(undefined)
  })

  it('renders toggle with label "Show growth badges"', () => {
    render(<DashboardPreferences initialValue={true} onToggle={mockOnToggle} />)
    expect(screen.getByText('Show growth badges')).toBeInTheDocument()
  })

  it('renders "Dashboard Preferences" heading', () => {
    render(<DashboardPreferences initialValue={true} onToggle={mockOnToggle} />)
    expect(screen.getByText('Dashboard Preferences')).toBeInTheDocument()
  })

  it('toggle is ON (checked) when initialValue is true', () => {
    render(<DashboardPreferences initialValue={true} onToggle={mockOnToggle} />)
    const toggle = screen.getByRole('switch', { name: 'Show growth badges' })
    expect(toggle).toHaveAttribute('aria-checked', 'true')
  })

  it('toggle is OFF (unchecked) when initialValue is false', () => {
    render(<DashboardPreferences initialValue={false} onToggle={mockOnToggle} />)
    const toggle = screen.getByRole('switch', { name: 'Show growth badges' })
    expect(toggle).toHaveAttribute('aria-checked', 'false')
  })

  it('clicking toggle calls onToggle callback with new value', async () => {
    const user = userEvent.setup()
    render(<DashboardPreferences initialValue={true} onToggle={mockOnToggle} />)
    const toggle = screen.getByRole('switch', { name: 'Show growth badges' })

    await user.click(toggle)
    expect(mockOnToggle).toHaveBeenCalledWith(false)
  })

  it('toggle has aria-label and role="switch"', () => {
    render(<DashboardPreferences initialValue={true} onToggle={mockOnToggle} />)
    const toggle = screen.getByRole('switch')
    expect(toggle).toHaveAttribute('aria-label', 'Show growth badges')
  })

  it('clicking toggle when OFF calls onToggle with true', async () => {
    const user = userEvent.setup()
    render(<DashboardPreferences initialValue={false} onToggle={mockOnToggle} />)
    const toggle = screen.getByRole('switch', { name: 'Show growth badges' })

    await user.click(toggle)
    expect(mockOnToggle).toHaveBeenCalledWith(true)
  })

  it('reverts toggle on onToggle rejection', async () => {
    const user = userEvent.setup()
    mockOnToggle.mockRejectedValue(new Error('Server error'))
    render(<DashboardPreferences initialValue={true} onToggle={mockOnToggle} />)
    const toggle = screen.getByRole('switch', { name: 'Show growth badges' })

    await user.click(toggle)
    // After rejection, should revert back to true
    expect(toggle).toHaveAttribute('aria-checked', 'true')
  })
})
