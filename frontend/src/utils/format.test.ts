import { describe, expect, it } from 'vitest'
import { formatCurrency } from './format'

describe('formatCurrency', () => {
  it('formats "4230.50" as "$4,230.50"', () => {
    expect(formatCurrency('4230.50')).toBe('$4,230.50')
  })

  it('formats "0.00" as "$0.00"', () => {
    expect(formatCurrency('0.00')).toBe('$0.00')
  })

  it('formats "15000" as "$15,000.00"', () => {
    expect(formatCurrency('15000')).toBe('$15,000.00')
  })

  it('formats "-500.25" as "-$500.25"', () => {
    expect(formatCurrency('-500.25')).toBe('-$500.25')
  })
})
