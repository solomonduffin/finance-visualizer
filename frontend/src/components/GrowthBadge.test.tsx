import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { GrowthBadge } from './GrowthBadge'

describe('GrowthBadge', () => {
  it('renders green triangle up and "+2.3%" when pctChange is "2.30"', () => {
    render(
      <GrowthBadge pctChange="2.30" dollarChange="280.00" visible={true} />
    )
    const badge = screen.getByText(/2\.3%/)
    expect(badge.textContent).toContain('\u25B2')
    expect(badge.textContent).toContain('+2.3%')
    expect(badge.className).toContain('text-green-600')
  })

  it('renders red triangle down and "-1.5%" when pctChange is "-1.50"', () => {
    render(
      <GrowthBadge pctChange="-1.50" dollarChange="-150.00" visible={true} />
    )
    const badge = screen.getByText(/1\.5%/)
    expect(badge.textContent).toContain('\u25BC')
    expect(badge.textContent).toContain('-1.5%')
    expect(badge.className).toContain('text-red-600')
  })

  it('renders invisible placeholder when pctChange is null', () => {
    const { container } = render(
      <GrowthBadge pctChange={null} dollarChange={null} visible={true} />
    )
    const span = container.querySelector('span')
    expect(span?.className).toContain('invisible')
  })

  it('renders invisible placeholder when visible prop is false', () => {
    const { container } = render(
      <GrowthBadge pctChange="2.30" dollarChange="280.00" visible={false} />
    )
    const span = container.querySelector('span')
    expect(span?.className).toContain('invisible')
  })

  it('renders invisible placeholder when pctChange is "0.00"', () => {
    const { container } = render(
      <GrowthBadge pctChange="0.00" dollarChange="0.00" visible={true} />
    )
    const span = container.querySelector('span')
    expect(span?.className).toContain('invisible')
  })

  it('shows tooltip "+$280.00 over 30 days" when dollarChange is "280.00"', () => {
    render(
      <GrowthBadge pctChange="2.30" dollarChange="280.00" visible={true} />
    )
    const badge = screen.getByText(/2\.3%/)
    expect(badge.getAttribute('title')).toBe('+$280.00 over 30 days')
  })

  it('shows tooltip "-$150.00 over 30 days" when dollarChange is "-150.00"', () => {
    render(
      <GrowthBadge pctChange="-1.50" dollarChange="-150.00" visible={true} />
    )
    const badge = screen.getByText(/1\.5%/)
    expect(badge.getAttribute('title')).toBe('-$150.00 over 30 days')
  })

  it('has text-sm font-semibold classes', () => {
    render(
      <GrowthBadge pctChange="3.00" dollarChange="100.00" visible={true} />
    )
    const badge = screen.getByText(/3\.0%/)
    expect(badge.className).toContain('text-sm')
    expect(badge.className).toContain('font-semibold')
  })

  it('has ml-2 class for spacing', () => {
    render(
      <GrowthBadge pctChange="3.00" dollarChange="100.00" visible={true} />
    )
    const badge = screen.getByText(/3\.0%/)
    expect(badge.className).toContain('ml-2')
  })

  it('renders dark mode green variant', () => {
    render(
      <GrowthBadge pctChange="5.00" dollarChange="500.00" visible={true} />
    )
    const badge = screen.getByText(/5\.0%/)
    expect(badge.className).toContain('dark:text-green-400')
  })

  it('renders dark mode red variant', () => {
    render(
      <GrowthBadge pctChange="-3.00" dollarChange="-300.00" visible={true} />
    )
    const badge = screen.getByText(/3\.0%/)
    expect(badge.className).toContain('dark:text-red-400')
  })
})
