import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { TimeRangeSelector } from './TimeRangeSelector'

describe('TimeRangeSelector', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  it('renders 5 buttons with role="radio"', () => {
    render(<TimeRangeSelector selected={90} onChange={vi.fn()} />)
    const radios = screen.getAllByRole('radio')
    expect(radios).toHaveLength(5)
  })

  it('renders labels 30d, 90d, 6m, 1y, All', () => {
    render(<TimeRangeSelector selected={90} onChange={vi.fn()} />)
    expect(screen.getByText('30d')).toBeInTheDocument()
    expect(screen.getByText('90d')).toBeInTheDocument()
    expect(screen.getByText('6m')).toBeInTheDocument()
    expect(screen.getByText('1y')).toBeInTheDocument()
    expect(screen.getByText('All')).toBeInTheDocument()
  })

  it('container has role="radiogroup"', () => {
    render(<TimeRangeSelector selected={90} onChange={vi.fn()} />)
    expect(screen.getByRole('radiogroup')).toBeInTheDocument()
  })

  it('default selected option 90d has aria-checked="true"', () => {
    render(<TimeRangeSelector selected={90} onChange={vi.fn()} />)
    const btn90d = screen.getByText('90d')
    expect(btn90d).toHaveAttribute('aria-checked', 'true')
  })

  it('non-selected options have aria-checked="false"', () => {
    render(<TimeRangeSelector selected={90} onChange={vi.fn()} />)
    const btn30d = screen.getByText('30d')
    expect(btn30d).toHaveAttribute('aria-checked', 'false')
  })

  it('clicking a different option calls onChange with correct days value', async () => {
    const onChange = vi.fn()
    render(<TimeRangeSelector selected={90} onChange={onChange} />)

    await userEvent.click(screen.getByText('1y'))
    expect(onChange).toHaveBeenCalledWith(365)

    await userEvent.click(screen.getByText('All'))
    expect(onChange).toHaveBeenCalledWith(0)

    await userEvent.click(screen.getByText('30d'))
    expect(onChange).toHaveBeenCalledWith(30)
  })
})
