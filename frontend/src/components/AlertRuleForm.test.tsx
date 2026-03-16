import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import AlertRuleForm from './AlertRuleForm'
import * as client from '../api/client'

vi.mock('../api/client')

const mockGetAccounts = vi.mocked(client.getAccounts)

const emptyAccountsResponse: client.AccountsResponse = {
  liquid: [],
  savings: [],
  investments: [],
  other: [],
  groups: [],
}

describe('AlertRuleForm', () => {
  let mockOnSave: ReturnType<typeof vi.fn>
  let mockOnCancel: ReturnType<typeof vi.fn>

  beforeEach(() => {
    vi.clearAllMocks()
    mockOnSave = vi.fn().mockResolvedValue(undefined)
    mockOnCancel = vi.fn()
    mockGetAccounts.mockResolvedValue(emptyAccountsResponse)
  })

  it('renders form with default operand dropdown', async () => {
    render(<AlertRuleForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    await waitFor(() => {
      expect(screen.getByText('Liquid Balance')).toBeInTheDocument()
    })
    expect(screen.getByText('Savings Balance')).toBeInTheDocument()
    expect(screen.getByText('Investments Balance')).toBeInTheDocument()
    expect(screen.getByText('Net Worth')).toBeInTheDocument()
    expect(screen.getByText('+ Add term')).toBeInTheDocument()
    // Verify optgroup exists via label attribute
    const selects = screen.getAllByRole('combobox') as HTMLSelectElement[]
    const operandSelect = selects[0]
    const optgroups = operandSelect.querySelectorAll('optgroup')
    expect(optgroups[0]).toHaveAttribute('label', 'Buckets')
  })

  it('validates empty name on submit', async () => {
    const user = userEvent.setup()
    render(<AlertRuleForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    await waitFor(() => {
      expect(screen.getByText('Liquid Balance')).toBeInTheDocument()
    })

    // Set threshold so only name fails
    const thresholdInput = screen.getByPlaceholderText('0.00')
    await user.type(thresholdInput, '5000')

    const saveBtn = screen.getByRole('button', { name: /Save Rule/i })
    await user.click(saveBtn)

    expect(screen.getByText('Rule name is required')).toBeInTheDocument()
    expect(mockOnSave).not.toHaveBeenCalled()
  })

  it('validates empty threshold on submit', async () => {
    const user = userEvent.setup()
    render(<AlertRuleForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    await waitFor(() => {
      expect(screen.getByText('Liquid Balance')).toBeInTheDocument()
    })

    // Set name so only threshold fails
    const nameInput = screen.getByPlaceholderText('e.g., Low cash warning')
    await user.type(nameInput, 'Test Rule')

    const saveBtn = screen.getByRole('button', { name: /Save Rule/i })
    await user.click(saveBtn)

    expect(screen.getByText('Threshold is required')).toBeInTheDocument()
    expect(mockOnSave).not.toHaveBeenCalled()
  })

  it('calls onSave with correct data on valid submit', async () => {
    const user = userEvent.setup()
    render(<AlertRuleForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    await waitFor(() => {
      expect(screen.getByText('Liquid Balance')).toBeInTheDocument()
    })

    const nameInput = screen.getByPlaceholderText('e.g., Low cash warning')
    await user.type(nameInput, 'Low cash warning')

    const thresholdInput = screen.getByPlaceholderText('0.00')
    await user.type(thresholdInput, '5000')

    const saveBtn = screen.getByRole('button', { name: /Save Rule/i })
    await user.click(saveBtn)

    await waitFor(() => {
      expect(mockOnSave).toHaveBeenCalledWith(
        expect.objectContaining({
          name: 'Low cash warning',
          threshold: '5000',
          notify_on_recovery: true,
          comparison: '<',
        })
      )
    })
    expect(mockOnSave.mock.calls[0][0].operands).toHaveLength(1)
    expect(mockOnSave.mock.calls[0][0].operands[0].type).toBe('bucket')
    expect(mockOnSave.mock.calls[0][0].operands[0].ref).toBe('liquid')
  })

  it('can add and remove operand rows', async () => {
    const user = userEvent.setup()
    render(<AlertRuleForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    await waitFor(() => {
      expect(screen.getByText('Liquid Balance')).toBeInTheDocument()
    })

    // Initially 1 operand, no remove button
    expect(screen.queryByLabelText('Remove term')).not.toBeInTheDocument()

    // Add a term
    await user.click(screen.getByText('+ Add term'))

    // Now 2 operands with remove buttons
    const removeButtons = screen.getAllByLabelText('Remove term')
    expect(removeButtons).toHaveLength(2)

    // Remove the second operand
    await user.click(removeButtons[1])

    // Back to 1 operand, no remove button
    expect(screen.queryByLabelText('Remove term')).not.toBeInTheDocument()
  })

  it('cancel button calls onCancel', async () => {
    const user = userEvent.setup()
    render(<AlertRuleForm onSave={mockOnSave} onCancel={mockOnCancel} />)
    await waitFor(() => {
      expect(screen.getByText('Liquid Balance')).toBeInTheDocument()
    })

    await user.click(screen.getByText('Discard Changes'))
    expect(mockOnCancel).toHaveBeenCalled()
  })
})
