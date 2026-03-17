import { render, screen, fireEvent } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { AllocationRow } from './AllocationRow'

describe('AllocationRow', () => {
  it('renders name and percentage input', () => {
    render(<AllocationRow name="Chase Checking" percentage="30" onChange={vi.fn()} />)
    expect(screen.getByText('Chase Checking')).toBeInTheDocument()
    expect(screen.getByRole('textbox')).toHaveValue('30')
  })

  it('onChange fires when input value changes', () => {
    const onChange = vi.fn()
    render(<AllocationRow name="Savings" percentage="50" onChange={onChange} />)
    const input = screen.getByRole('textbox')
    fireEvent.change(input, { target: { value: '60' } })
    expect(onChange).toHaveBeenCalledWith('60')
  })

  it('visual bar width matches percentage', () => {
    const { container } = render(
      <AllocationRow name="Account" percentage="75" onChange={vi.fn()} />
    )
    const fill = container.querySelector('.bg-blue-500')
    expect(fill).not.toBeNull()
    expect(fill!.getAttribute('style')).toContain('width: 75%')
  })

  it('bar width capped at 100% even if percentage > 100', () => {
    const { container } = render(
      <AllocationRow name="Account" percentage="150" onChange={vi.fn()} />
    )
    const fill = container.querySelector('.bg-blue-500')
    expect(fill!.getAttribute('style')).toContain('width: 100%')
  })

  it('bar width is 0% for empty string', () => {
    const { container } = render(
      <AllocationRow name="Account" percentage="" onChange={vi.fn()} />
    )
    const fill = container.querySelector('.bg-blue-500')
    expect(fill!.getAttribute('style')).toContain('width: 0%')
  })

  it('renders rounded-full class on bar elements', () => {
    const { container } = render(
      <AllocationRow name="Account" percentage="50" onChange={vi.fn()} />
    )
    const roundedElements = container.querySelectorAll('.rounded-full')
    expect(roundedElements.length).toBeGreaterThanOrEqual(2) // track + fill
  })

  it('renders transition-all class on fill bar', () => {
    const { container } = render(
      <AllocationRow name="Account" percentage="50" onChange={vi.fn()} />
    )
    const transitionElement = container.querySelector('.transition-all')
    expect(transitionElement).not.toBeNull()
  })
})
