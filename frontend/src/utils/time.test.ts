import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { timeAgo } from './time'

describe('timeAgo', () => {
  const BASE_DATE = new Date('2026-03-15T10:00:00Z').getTime()

  beforeEach(() => {
    vi.spyOn(Date, 'now').mockReturnValue(BASE_DATE)
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('returns "just now" for less than 60 seconds', () => {
    vi.spyOn(Date, 'now').mockReturnValue(BASE_DATE + 30 * 1000)
    expect(timeAgo('2026-03-15T10:00:00Z')).toBe('just now')
  })

  it('returns "Xm ago" for minutes', () => {
    vi.spyOn(Date, 'now').mockReturnValue(BASE_DATE + 120 * 1000)
    expect(timeAgo('2026-03-15T10:00:00Z')).toBe('2m ago')
  })

  it('returns "Xh ago" for hours', () => {
    vi.spyOn(Date, 'now').mockReturnValue(BASE_DATE + 7200 * 1000)
    expect(timeAgo('2026-03-15T10:00:00Z')).toBe('2h ago')
  })

  it('returns "Xd ago" for days', () => {
    vi.spyOn(Date, 'now').mockReturnValue(BASE_DATE + 172800 * 1000)
    expect(timeAgo('2026-03-15T10:00:00Z')).toBe('2d ago')
  })
})
