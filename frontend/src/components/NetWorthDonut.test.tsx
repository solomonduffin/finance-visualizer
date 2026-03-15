import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { NetWorthDonut } from './NetWorthDonut'

vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: any) => <div>{children}</div>,
  PieChart: ({ children }: any) => <div data-testid="pie-chart">{children}</div>,
  Pie: ({ data, children }: any) => (
    <div data-testid="pie" data-segments={data?.length}>
      {children}
    </div>
  ),
  Cell: (props: any) => <div data-testid="cell" data-fill={props.fill} />,
  Label: (props: any) => <div data-testid="label">{props.value}</div>,
  Tooltip: () => <div />,
}))

describe('NetWorthDonut', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  it('renders segments for all non-zero values', () => {
    render(
      <NetWorthDonut
        liquid="5000.00"
        savings="20000.00"
        investments="50000.00"
        isDark={false}
      />
    )
    const pie = screen.getByTestId('pie')
    expect(pie.getAttribute('data-segments')).toBe('3')
  })

  it('does not render segment for zero-value panel', () => {
    render(
      <NetWorthDonut
        liquid="5000.00"
        savings="0.00"
        investments="50000.00"
        isDark={false}
      />
    )
    const pie = screen.getByTestId('pie')
    // Only 2 non-zero segments
    expect(pie.getAttribute('data-segments')).toBe('2')
  })

  it('total in center label matches sum of all values', () => {
    render(
      <NetWorthDonut
        liquid="5000.00"
        savings="20000.00"
        investments="50000.00"
        isDark={false}
      />
    )
    // Total = 75000.00
    const label = screen.getByTestId('label')
    expect(label.textContent).toContain('75,000')
  })

  it('segment colors match PANEL_COLORS accent in light mode', () => {
    render(
      <NetWorthDonut
        liquid="5000.00"
        savings="20000.00"
        investments="50000.00"
        isDark={false}
      />
    )
    const cells = screen.getAllByTestId('cell')
    const fills = cells.map((c) => c.getAttribute('data-fill'))
    expect(fills).toContain('#3b82f6')  // liquid accent
    expect(fills).toContain('#22c55e')  // savings accent
    expect(fills).toContain('#a855f7')  // investments accent
  })

  it('segment colors use darkAccent in dark mode', () => {
    render(
      <NetWorthDonut
        liquid="5000.00"
        savings="20000.00"
        investments="50000.00"
        isDark={true}
      />
    )
    const cells = screen.getAllByTestId('cell')
    const fills = cells.map((c) => c.getAttribute('data-fill'))
    expect(fills).toContain('#60a5fa')  // liquid darkAccent
    expect(fills).toContain('#4ade80')  // savings darkAccent
    expect(fills).toContain('#c084fc')  // investments darkAccent
  })

  it('legend shows panel labels and formatted amounts', () => {
    render(
      <NetWorthDonut
        liquid="5000.00"
        savings="20000.00"
        investments="50000.00"
        isDark={false}
      />
    )
    expect(screen.getByText('Liquid')).toBeInTheDocument()
    expect(screen.getByText('Savings')).toBeInTheDocument()
    expect(screen.getByText('Investments')).toBeInTheDocument()
    // Formatted amounts
    expect(screen.getByText('$5,000.00')).toBeInTheDocument()
    expect(screen.getByText('$20,000.00')).toBeInTheDocument()
    expect(screen.getByText('$50,000.00')).toBeInTheDocument()
  })

  it('shows "No data" message when all values are zero', () => {
    render(
      <NetWorthDonut
        liquid="0.00"
        savings="0.00"
        investments="0.00"
        isDark={false}
      />
    )
    expect(screen.getByText(/no data/i)).toBeInTheDocument()
    expect(screen.queryByTestId('pie-chart')).not.toBeInTheDocument()
  })
})
