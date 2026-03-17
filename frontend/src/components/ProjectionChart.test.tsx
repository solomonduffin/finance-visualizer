import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ProjectionChart } from './ProjectionChart'

vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: any) => (
    <div data-testid="responsive-container">{children}</div>
  ),
  ComposedChart: ({ data, children }: any) => (
    <div data-testid="composed-chart" data-count={data?.length}>
      {children}
    </div>
  ),
  Area: (props: any) => (
    <div
      data-testid="chart-area"
      data-datakey={props.dataKey}
      data-connectnulls={String(props.connectNulls)}
    />
  ),
  Line: (props: any) => (
    <div
      data-testid="chart-line"
      data-datakey={props.dataKey}
      data-strokedasharray={props.strokeDasharray}
      data-connectnulls={String(props.connectNulls)}
    />
  ),
  XAxis: () => <div data-testid="x-axis" />,
  YAxis: () => <div data-testid="y-axis" />,
  CartesianGrid: () => <div data-testid="cartesian-grid" />,
  Tooltip: () => <div data-testid="tooltip" />,
  ReferenceLine: ({ children, x }: any) => (
    <div data-testid="reference-line" data-x={x}>
      {children}
    </div>
  ),
  Label: (props: any) => <span data-testid="label" data-value={props.value} />,
}))

describe('ProjectionChart', () => {
  beforeEach(() => {
    vi.resetAllMocks()
  })

  it('renders chart container with role="img"', () => {
    const { container } = render(
      <ProjectionChart
        historicalData={[{ date: '2026-03-01', value: 50000 }]}
        projectionData={[
          { date: '2026-03-01', value: 50000 },
          { date: '2026-04-01', value: 51000 },
        ]}
        isDark={false}
      />,
    )
    const chartContainer = container.querySelector('[role="img"]')
    expect(chartContainer).toBeTruthy()
    expect(chartContainer?.getAttribute('aria-label')).toBe(
      'Net worth projection chart showing historical and projected values',
    )
  })

  it('renders empty state message when no data provided', () => {
    render(
      <ProjectionChart
        historicalData={[]}
        projectionData={[]}
        isDark={false}
      />,
    )
    expect(
      screen.getByText(
        'No data to project. Sync your accounts to see projections.',
      ),
    ).toBeInTheDocument()
  })

  it('renders with historical and projection data', () => {
    render(
      <ProjectionChart
        historicalData={[
          { date: '2026-02-01', value: 48000 },
          { date: '2026-03-01', value: 50000 },
        ]}
        projectionData={[
          { date: '2026-03-01', value: 50000 },
          { date: '2026-04-01', value: 51000 },
          { date: '2026-05-01', value: 52000 },
        ]}
        isDark={false}
      />,
    )
    // ComposedChart should be rendered
    expect(screen.getByTestId('composed-chart')).toBeInTheDocument()
    // Should have Line elements
    const lines = screen.getAllByTestId('chart-line')
    expect(lines.length).toBeGreaterThanOrEqual(2)
    // Should have a dashed projected line
    const dashedLine = lines.find(
      (l) => l.getAttribute('data-strokedasharray') === '8 4',
    )
    expect(dashedLine).toBeTruthy()
    // All lines should have connectNulls=false
    lines.forEach((l) => {
      expect(l.getAttribute('data-connectnulls')).toBe('false')
    })
    // Should have ReferenceLine for "Now" marker
    expect(screen.getByTestId('reference-line')).toBeInTheDocument()
    // Should have "Now" label
    const label = screen.getByTestId('label')
    expect(label.getAttribute('data-value')).toBe('Now')
  })

  it('renders empty state container with role="img" when no data', () => {
    const { container } = render(
      <ProjectionChart
        historicalData={[]}
        projectionData={[]}
        isDark={false}
      />,
    )
    const chartContainer = container.querySelector('[role="img"]')
    expect(chartContainer).toBeTruthy()
  })
})
