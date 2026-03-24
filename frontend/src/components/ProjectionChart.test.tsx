import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { ProjectionChart, buildCombinedData } from './ProjectionChart'

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

describe('buildCombinedData', () => {
  it('bridge point projected value is projectionData[0].value, not historical net worth', () => {
    // Regression test for double-counting bug:
    // historicalData ends at $8394 (full net worth — includes all accounts).
    // projectionData[0] is $500 (only the checking account is included in the projection).
    // The bridge must start the projected series at $500, not $8394.
    // Using $8394 would double-count every account already in net worth but not
    // explicitly included in the projection settings.
    const historicalData = [
      { date: '2026-02-01', value: 7000 },
      { date: '2026-03-01', value: 8394 },
    ]
    const projectionData = [
      { date: '2026-03-01', value: 500 }, // sum of included accounts only
      { date: '2026-04-01', value: 502 },
      { date: '2026-05-01', value: 504 },
    ]

    const combined = buildCombinedData(historicalData, projectionData)

    // The bridge point is the last historical date (2026-03-01)
    const bridge = combined.find(
      (p) => p.historical !== null && p.projected !== null,
    )
    expect(bridge).toBeDefined()
    expect(bridge!.date).toBe('2026-03-01')
    // Must use projection baseline (included accounts only), not full net worth
    expect(bridge!.projected).toBe(500)
    expect(bridge!.historical).toBe(8394)
  })

  it('projection points after bridge have historical=null', () => {
    const historicalData = [{ date: '2026-03-01', value: 8394 }]
    const projectionData = [
      { date: '2026-03-01', value: 500 },
      { date: '2026-04-01', value: 502 },
    ]

    const combined = buildCombinedData(historicalData, projectionData)

    // Month 1 projection point should have historical=null
    const month1 = combined.find((p) => p.date === '2026-04-01')
    expect(month1).toBeDefined()
    expect(month1!.historical).toBeNull()
    expect(month1!.projected).toBe(502)
  })

  it('returns only historical points when projectionData is empty', () => {
    const historicalData = [
      { date: '2026-02-01', value: 7000 },
      { date: '2026-03-01', value: 8394 },
    ]

    const combined = buildCombinedData(historicalData, [])

    expect(combined).toHaveLength(2)
    combined.forEach((p) => {
      expect(p.projected).toBeNull()
    })
  })

  it('returns only projection points when historicalData is empty', () => {
    const projectionData = [
      { date: '2026-03-01', value: 500 },
      { date: '2026-04-01', value: 502 },
    ]

    const combined = buildCombinedData([], projectionData)

    // No historical, so no bridge. todayStr = projectionData[0].date, so
    // projectionData[0] is skipped by the "skip bridge date" loop. Only
    // subsequent projection points appear.
    expect(combined).toHaveLength(1)
    expect(combined[0].date).toBe('2026-04-01')
    expect(combined[0].projected).toBe(502)
    expect(combined[0].historical).toBeNull()
  })
})
