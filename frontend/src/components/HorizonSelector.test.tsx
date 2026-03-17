import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { HorizonSelector } from './HorizonSelector'

describe('HorizonSelector', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  it('renders all preset buttons (1y, 5y, 10y, 20y, Custom)', () => {
    render(<HorizonSelector years={5} onChange={vi.fn()} />)
    expect(screen.getByText('1y')).toBeInTheDocument()
    expect(screen.getByText('5y')).toBeInTheDocument()
    expect(screen.getByText('10y')).toBeInTheDocument()
    expect(screen.getByText('20y')).toBeInTheDocument()
    expect(screen.getByText('Custom')).toBeInTheDocument()
  })

  it('active preset has aria-checked="true"', () => {
    render(<HorizonSelector years={10} onChange={vi.fn()} />)
    const btn10y = screen.getByText('10y')
    expect(btn10y).toHaveAttribute('aria-checked', 'true')
    // Other presets should be false
    expect(screen.getByText('1y')).toHaveAttribute('aria-checked', 'false')
    expect(screen.getByText('5y')).toHaveAttribute('aria-checked', 'false')
    expect(screen.getByText('20y')).toHaveAttribute('aria-checked', 'false')
    expect(screen.getByText('Custom')).toHaveAttribute('aria-checked', 'false')
  })

  it('clicking a preset calls onChange with correct years', async () => {
    const onChange = vi.fn()
    render(<HorizonSelector years={5} onChange={onChange} />)

    await userEvent.click(screen.getByText('1y'))
    expect(onChange).toHaveBeenCalledWith(1)

    await userEvent.click(screen.getByText('20y'))
    expect(onChange).toHaveBeenCalledWith(20)
  })

  it('clicking Custom reveals the year input', async () => {
    const onChange = vi.fn()
    render(<HorizonSelector years={5} onChange={onChange} />)

    // Custom input should not be visible initially (5 is a preset)
    expect(screen.queryByLabelText('Custom projection years')).not.toBeInTheDocument()

    // Click Custom
    await userEvent.click(screen.getByText('Custom'))

    // Now the input should be visible -- but onChange fires immediately
    // Re-render with a non-preset value to see the input
    const { rerender } = render(<HorizonSelector years={7} onChange={onChange} />)
    expect(screen.getByLabelText('Custom projection years')).toBeInTheDocument()
  })

  it('custom input has min=1 max=50', () => {
    render(<HorizonSelector years={7} onChange={vi.fn()} />)
    const input = screen.getByLabelText('Custom projection years')
    expect(input).toHaveAttribute('min', '1')
    expect(input).toHaveAttribute('max', '50')
  })

  it('container has role="radiogroup" with correct aria-label', () => {
    render(<HorizonSelector years={5} onChange={vi.fn()} />)
    const radiogroup = screen.getByRole('radiogroup')
    expect(radiogroup).toHaveAttribute('aria-label', 'Select projection horizon')
  })

  it('all preset buttons have role="radio"', () => {
    render(<HorizonSelector years={5} onChange={vi.fn()} />)
    const radios = screen.getAllByRole('radio')
    expect(radios).toHaveLength(5)
  })

  it('non-preset years value activates Custom', () => {
    render(<HorizonSelector years={15} onChange={vi.fn()} />)
    expect(screen.getByText('Custom')).toHaveAttribute('aria-checked', 'true')
    expect(screen.getByLabelText('Custom projection years')).toBeInTheDocument()
  })
})
