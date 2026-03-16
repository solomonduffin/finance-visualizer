import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { NetWorthStats } from './NetWorthStats'
import type { NetWorthStatsData } from '../api/client'

describe('NetWorthStats', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  const baseStats: NetWorthStatsData = {
    current_net_worth: '75000.00',
    period_change_dollars: '3200.00',
    period_change_pct: '4.45',
    all_time_high: '78000.00',
    all_time_high_date: '2026-03-10',
  }

  it('renders current net worth', () => {
    render(<NetWorthStats stats={baseStats} selectedDays={90} />)
    expect(screen.getByText('Current Net Worth')).toBeInTheDocument()
    expect(screen.getByText('$75,000.00')).toBeInTheDocument()
  })

  it('renders period change value', () => {
    render(<NetWorthStats stats={baseStats} selectedDays={90} />)
    // Should show formatted change with + prefix
    expect(screen.getByText(/\+\$3,200\.00/)).toBeInTheDocument()
  })

  it('renders all-time high', () => {
    render(<NetWorthStats stats={baseStats} selectedDays={90} />)
    expect(screen.getByText('All-Time High')).toBeInTheDocument()
    expect(screen.getByText('$78,000.00')).toBeInTheDocument()
  })

  it('positive change shows green color class', () => {
    const { container } = render(<NetWorthStats stats={baseStats} selectedDays={90} />)
    const greenEl = container.querySelector('.text-green-600')
    expect(greenEl).toBeTruthy()
  })

  it('negative change shows red color class', () => {
    const negStats: NetWorthStatsData = {
      ...baseStats,
      period_change_dollars: '-1500.00',
      period_change_pct: '-2.00',
    }
    const { container } = render(<NetWorthStats stats={negStats} selectedDays={90} />)
    const redEl = container.querySelector('.text-red-600')
    expect(redEl).toBeTruthy()
  })

  it('period change label reflects 90 days', () => {
    render(<NetWorthStats stats={baseStats} selectedDays={90} />)
    expect(screen.getByText('90-Day Change')).toBeInTheDocument()
  })

  it('period change label reflects 30 days', () => {
    render(<NetWorthStats stats={baseStats} selectedDays={30} />)
    expect(screen.getByText('30-Day Change')).toBeInTheDocument()
  })

  it('period change label reflects 6 months', () => {
    render(<NetWorthStats stats={baseStats} selectedDays={180} />)
    expect(screen.getByText('6-Month Change')).toBeInTheDocument()
  })

  it('period change label reflects 1 year', () => {
    render(<NetWorthStats stats={baseStats} selectedDays={365} />)
    expect(screen.getByText('1-Year Change')).toBeInTheDocument()
  })

  it('period change label reflects all-time', () => {
    render(<NetWorthStats stats={baseStats} selectedDays={0} />)
    expect(screen.getByText('All-Time Change')).toBeInTheDocument()
  })

  it('handles null period_change_pct gracefully', () => {
    const nullPctStats: NetWorthStatsData = {
      ...baseStats,
      period_change_pct: null,
    }
    render(<NetWorthStats stats={nullPctStats} selectedDays={90} />)
    // Should still render without crashing
    expect(screen.getByText('Current Net Worth')).toBeInTheDocument()
  })
})
