import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { SkeletonDashboard } from './SkeletonDashboard'

describe('SkeletonDashboard', () => {
  it('renders 3 skeleton cards', () => {
    const { container } = render(<SkeletonDashboard />)
    const cards = container.querySelectorAll('[data-testid="skeleton-card"]')
    expect(cards).toHaveLength(3)
  })

  it('skeleton cards have animate-pulse class', () => {
    const { container } = render(<SkeletonDashboard />)
    const cards = container.querySelectorAll('[data-testid="skeleton-card"]')
    cards.forEach((card) => {
      expect(card.className).toContain('animate-pulse')
    })
  })
})
