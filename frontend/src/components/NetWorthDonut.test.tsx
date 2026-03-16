import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { NetWorthDonut } from './NetWorthDonut'

const mockNavigate = vi.fn()
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  }
})

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

function renderDonut(props?: Partial<{ liquid: string; savings: string; investments: string; isDark: boolean }>) {
  return render(
    <MemoryRouter>
      <NetWorthDonut
        liquid={props?.liquid ?? '5000.00'}
        savings={props?.savings ?? '20000.00'}
        investments={props?.investments ?? '50000.00'}
        isDark={props?.isDark ?? false}
      />
    </MemoryRouter>
  )
}

describe('NetWorthDonut', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders segments for all non-zero values', () => {
    renderDonut()
    const pie = screen.getByTestId('pie')
    expect(pie.getAttribute('data-segments')).toBe('3')
  })

  it('does not render segment for zero-value panel', () => {
    renderDonut({ savings: '0.00' })
    const pie = screen.getByTestId('pie')
    expect(pie.getAttribute('data-segments')).toBe('2')
  })

  it('total in center label matches sum of all values', () => {
    renderDonut()
    const label = screen.getByTestId('label')
    expect(label.textContent).toContain('75,000')
  })

  it('segment colors match PANEL_COLORS accent in light mode', () => {
    renderDonut()
    const cells = screen.getAllByTestId('cell')
    const fills = cells.map((c) => c.getAttribute('data-fill'))
    expect(fills).toContain('#3b82f6')
    expect(fills).toContain('#22c55e')
    expect(fills).toContain('#a855f7')
  })

  it('segment colors use darkAccent in dark mode', () => {
    renderDonut({ isDark: true })
    const cells = screen.getAllByTestId('cell')
    const fills = cells.map((c) => c.getAttribute('data-fill'))
    expect(fills).toContain('#60a5fa')
    expect(fills).toContain('#4ade80')
    expect(fills).toContain('#c084fc')
  })

  it('legend shows panel labels and formatted amounts', () => {
    renderDonut()
    expect(screen.getByText('Liquid')).toBeInTheDocument()
    expect(screen.getByText('Savings')).toBeInTheDocument()
    expect(screen.getByText('Investments')).toBeInTheDocument()
    expect(screen.getByText('$5,000.00')).toBeInTheDocument()
    expect(screen.getByText('$20,000.00')).toBeInTheDocument()
    expect(screen.getByText('$50,000.00')).toBeInTheDocument()
  })

  it('shows "No data" message when all values are zero', () => {
    renderDonut({ liquid: '0.00', savings: '0.00', investments: '0.00' })
    expect(screen.getByText(/no data/i)).toBeInTheDocument()
    expect(screen.queryByTestId('pie-chart')).not.toBeInTheDocument()
  })

  it('has role="link" for accessibility', () => {
    renderDonut()
    expect(screen.getByRole('link', { name: /view net worth details/i })).toBeInTheDocument()
  })

  it('clicking the donut navigates to /net-worth', async () => {
    renderDonut()
    const linkWrapper = screen.getByRole('link', { name: /view net worth details/i })
    await userEvent.click(linkWrapper)
    expect(mockNavigate).toHaveBeenCalledWith('/net-worth')
  })
})
