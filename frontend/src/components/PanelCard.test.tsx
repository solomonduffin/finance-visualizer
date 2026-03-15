import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { PanelCard } from './PanelCard'

const sampleAccounts = [
  { id: '1', name: 'Chase Checking', balance: '1230.50' },
  { id: '2', name: 'Wells Fargo', balance: '3000.00' },
]

describe('PanelCard', () => {
  it('renders the panel label', () => {
    render(
      <PanelCard panelKey="liquid" total="4230.50" accounts={sampleAccounts} />
    )
    expect(screen.getByText('Liquid')).toBeInTheDocument()
  })

  it('renders the formatted total balance', () => {
    render(
      <PanelCard panelKey="liquid" total="4230.50" accounts={sampleAccounts} />
    )
    expect(screen.getByText('$4,230.50')).toBeInTheDocument()
  })

  it('renders all account names', () => {
    render(
      <PanelCard panelKey="liquid" total="4230.50" accounts={sampleAccounts} />
    )
    expect(screen.getByText('Chase Checking')).toBeInTheDocument()
    expect(screen.getByText('Wells Fargo')).toBeInTheDocument()
  })

  it('renders all account balances formatted', () => {
    render(
      <PanelCard panelKey="liquid" total="4230.50" accounts={sampleAccounts} />
    )
    expect(screen.getByText('$1,230.50')).toBeInTheDocument()
    expect(screen.getByText('$3,000.00')).toBeInTheDocument()
  })

  it('renders correctly with empty accounts array', () => {
    render(<PanelCard panelKey="savings" total="15000.00" accounts={[]} />)
    expect(screen.getByText('Savings')).toBeInTheDocument()
    expect(screen.getByText('$15,000.00')).toBeInTheDocument()
  })
})
