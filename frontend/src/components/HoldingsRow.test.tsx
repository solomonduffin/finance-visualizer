import { render, screen, fireEvent } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { HoldingsRow } from './HoldingsRow'
import type { ProjectionHoldingSetting } from '../api/client'

const mockHoldings: ProjectionHoldingSetting[] = [
  {
    holding_id: 'h1',
    symbol: 'AAPL',
    description: 'Apple Inc.',
    market_value: '5000.00',
    apy: '7.0',
    compound: true,
    included: true,
    allocation: '50',
  },
  {
    holding_id: 'h2',
    symbol: 'GOOG',
    description: 'Alphabet Inc.',
    market_value: '3000.00',
    apy: '5.0',
    compound: false,
    included: true,
    allocation: '30',
  },
]

const defaultProps = {
  holdings: mockHoldings,
  accountId: 'acc-1',
  expanded: true,
  onApyChange: vi.fn(),
  onCompoundChange: vi.fn(),
  onIncludeChange: vi.fn(),
  isDark: false,
}

describe('HoldingsRow', () => {
  it('renders holding description and market value', () => {
    render(<HoldingsRow {...defaultProps} />)
    expect(screen.getByText('Apple Inc.')).toBeInTheDocument()
    expect(screen.getByText('$5,000.00')).toBeInTheDocument()
    expect(screen.getByText('Alphabet Inc.')).toBeInTheDocument()
    expect(screen.getByText('$3,000.00')).toBeInTheDocument()
  })

  it('renders APY input for each holding', () => {
    render(<HoldingsRow {...defaultProps} />)
    const apyInputs = screen.getAllByRole('textbox')
    expect(apyInputs).toHaveLength(2)
    expect(apyInputs[0]).toHaveValue('7.0')
    expect(apyInputs[1]).toHaveValue('5.0')
  })

  it('expanded=true shows holdings, expanded=false hides them', () => {
    const { container, rerender } = render(<HoldingsRow {...defaultProps} expanded={true} />)
    const holdingsDiv = container.querySelector('#holdings-acc-1')
    expect(holdingsDiv).not.toBeNull()
    expect(holdingsDiv!.getAttribute('style')).not.toContain('max-height: 0px')

    rerender(<HoldingsRow {...defaultProps} expanded={false} />)
    expect(holdingsDiv!.getAttribute('style')).toContain('max-height: 0px')
  })

  it('fires onApyChange when APY input changes', () => {
    const onApyChange = vi.fn()
    render(<HoldingsRow {...defaultProps} onApyChange={onApyChange} />)
    const inputs = screen.getAllByRole('textbox')
    fireEvent.change(inputs[0], { target: { value: '8.5' } })
    expect(onApyChange).toHaveBeenCalledWith('h1', '8.5')
  })

  it('fires onCompoundChange when toggle is clicked', () => {
    const onCompoundChange = vi.fn()
    render(<HoldingsRow {...defaultProps} onCompoundChange={onCompoundChange} />)
    const toggles = screen.getAllByRole('switch')
    fireEvent.click(toggles[0])
    expect(onCompoundChange).toHaveBeenCalledWith('h1', false)
  })

  it('fires onIncludeChange when checkbox changes', () => {
    const onIncludeChange = vi.fn()
    render(<HoldingsRow {...defaultProps} onIncludeChange={onIncludeChange} />)
    const checkboxes = screen.getAllByRole('checkbox')
    fireEvent.click(checkboxes[0])
    expect(onIncludeChange).toHaveBeenCalledWith('h1', false)
  })

  it('uses symbol when description is empty', () => {
    const holdingsNoDesc: ProjectionHoldingSetting[] = [
      {
        holding_id: 'h3',
        symbol: 'TSLA',
        description: '',
        market_value: '2000.00',
        apy: '6.0',
        compound: true,
        included: true,
        allocation: '20',
      },
    ]
    render(<HoldingsRow {...defaultProps} holdings={holdingsNoDesc} />)
    expect(screen.getByText('TSLA')).toBeInTheDocument()
  })

  it('has pl-8 indent class on holding rows', () => {
    const { container } = render(<HoldingsRow {...defaultProps} />)
    const holdingRows = container.querySelectorAll('.pl-8')
    expect(holdingRows.length).toBe(2)
  })

  it('has motion-reduce:transition-none for accessibility', () => {
    const { container } = render(<HoldingsRow {...defaultProps} />)
    const holdingsDiv = container.querySelector('#holdings-acc-1')
    expect(holdingsDiv!.className).toContain('motion-reduce:transition-none')
  })
})
