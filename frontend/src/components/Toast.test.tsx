import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { Toast } from './Toast'

describe('Toast', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders the message', () => {
    render(<Toast message="Account restored" onDismiss={() => {}} />)
    expect(screen.getByText('Account restored')).toBeInTheDocument()
  })

  it('calls onDismiss after 4 seconds', () => {
    const onDismiss = vi.fn()
    render(<Toast message="Test" onDismiss={onDismiss} />)

    expect(onDismiss).not.toHaveBeenCalled()

    vi.advanceTimersByTime(4000)

    expect(onDismiss).toHaveBeenCalledTimes(1)
  })

  it('does not auto-dismiss before 4 seconds', () => {
    const onDismiss = vi.fn()
    render(<Toast message="Test" onDismiss={onDismiss} />)

    vi.advanceTimersByTime(3999)

    expect(onDismiss).not.toHaveBeenCalled()
  })

  it('calls onDismiss when dismiss button clicked', async () => {
    vi.useRealTimers()
    const onDismiss = vi.fn()
    render(<Toast message="Test" onDismiss={onDismiss} />)

    await userEvent.click(screen.getByRole('button', { name: /dismiss/i }))

    expect(onDismiss).toHaveBeenCalledTimes(1)
  })

  it('has role="status" for accessibility', () => {
    render(<Toast message="Test" onDismiss={() => {}} />)
    expect(screen.getByRole('status')).toBeInTheDocument()
  })
})
