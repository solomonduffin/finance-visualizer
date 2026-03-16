import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import Alerts from './Alerts'
import * as client from '../api/client'

vi.mock('../api/client')

// Mock AlertRuleForm to avoid needing getAccounts in every test
vi.mock('../components/AlertRuleForm', () => ({
  default: (props: any) => (
    <div data-testid="alert-rule-form">
      <button onClick={() => props.onCancel()}>Discard Changes</button>
      <button onClick={() => props.onSave({ name: 'Test', operands: [], comparison: '<', threshold: '100', notify_on_recovery: true })}>
        Save Rule
      </button>
    </div>
  ),
}))

const mockGetAlerts = vi.mocked(client.getAlerts)
const mockCreateAlert = vi.mocked(client.createAlert)
const mockToggleAlert = vi.mocked(client.toggleAlert)
const mockDeleteAlert = vi.mocked(client.deleteAlert)

const sampleRules: client.AlertRule[] = [
  {
    id: 1,
    name: 'Low cash warning',
    operands: [
      { id: 'op1', type: 'bucket', ref: 'liquid', label: 'Liquid Balance', operator: '+' },
    ],
    expression: 'liquid',
    comparison: '<',
    threshold: '5000',
    notify_on_recovery: true,
    enabled: true,
    last_state: 'normal',
    last_eval_at: '2026-03-16T20:00:00Z',
    last_value: '6000.00',
    created_at: '2026-03-15T00:00:00Z',
    updated_at: '2026-03-16T20:00:00Z',
    history: [],
  },
  {
    id: 2,
    name: 'Savings goal',
    operands: [
      { id: 'op2', type: 'bucket', ref: 'savings', label: 'Savings Balance', operator: '+' },
    ],
    expression: 'savings',
    comparison: '>=',
    threshold: '50000',
    notify_on_recovery: false,
    enabled: true,
    last_state: 'triggered',
    last_eval_at: '2026-03-16T19:30:00Z',
    last_value: '45000.00',
    created_at: '2026-03-14T00:00:00Z',
    updated_at: '2026-03-16T19:30:00Z',
    history: [],
  },
]

describe('Alerts page', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders loading skeletons initially', () => {
    // Never resolve the promise to keep loading state
    mockGetAlerts.mockReturnValue(new Promise(() => {}))
    render(<Alerts />)

    const skeletons = document.querySelectorAll('.animate-pulse')
    expect(skeletons.length).toBe(3)
  })

  it('renders empty state when no rules', async () => {
    mockGetAlerts.mockResolvedValue([])
    render(<Alerts />)

    await waitFor(() => {
      expect(screen.getByText('No alert rules yet')).toBeInTheDocument()
    })
    expect(screen.getByText('Create your first alert to get notified when your balances cross a threshold.')).toBeInTheDocument()
    expect(screen.getByText('Create Alert')).toBeInTheDocument()
  })

  it('renders error state with retry button', async () => {
    mockGetAlerts.mockRejectedValue(new Error('Network error'))
    render(<Alerts />)

    await waitFor(() => {
      expect(screen.getByText('Failed to load alerts')).toBeInTheDocument()
    })
    expect(screen.getByText('Retry Loading')).toBeInTheDocument()
  })

  it('renders rule cards when rules exist', async () => {
    mockGetAlerts.mockResolvedValue(sampleRules)
    render(<Alerts />)

    await waitFor(() => {
      expect(screen.getByText('Low cash warning')).toBeInTheDocument()
    })
    expect(screen.getByText('Savings goal')).toBeInTheDocument()
  })

  it('"+ New Alert" button shows the builder form', async () => {
    const user = userEvent.setup()
    mockGetAlerts.mockResolvedValue(sampleRules)
    render(<Alerts />)

    await waitFor(() => {
      expect(screen.getByText('Low cash warning')).toBeInTheDocument()
    })

    await user.click(screen.getByText('+ New Alert'))
    expect(screen.getByTestId('alert-rule-form')).toBeInTheDocument()
  })
})
