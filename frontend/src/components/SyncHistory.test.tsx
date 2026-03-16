import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import SyncHistory from './SyncHistory'
import type { SyncLogEntry } from '../api/client'

vi.mock('../api/client', () => ({
  getSyncLog: vi.fn(),
}))

import { getSyncLog } from '../api/client'
const mockGetSyncLog = vi.mocked(getSyncLog)

function makeEntry(overrides: Partial<SyncLogEntry> & { id: number; status: SyncLogEntry['status'] }): SyncLogEntry {
  return {
    started_at: '2026-03-15T14:30:00Z',
    finished_at: '2026-03-15T14:30:10Z',
    accounts_fetched: 5,
    accounts_failed: 0,
    error_text: null,
    ...overrides,
  }
}

describe('SyncHistory', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders "No sync history yet." when entries array is empty', async () => {
    mockGetSyncLog.mockResolvedValue({ entries: [] })
    render(<SyncHistory />)
    expect(await screen.findByText('No sync history yet.')).toBeInTheDocument()
  })

  it('renders 3 sync entries when given 3 entries', async () => {
    mockGetSyncLog.mockResolvedValue({
      entries: [
        makeEntry({ id: 1, status: 'success', accounts_fetched: 5 }),
        makeEntry({ id: 2, status: 'partial', accounts_fetched: 4, accounts_failed: 1 }),
        makeEntry({ id: 3, status: 'failed', error_text: 'Connection timeout' }),
      ],
    })
    render(<SyncHistory />)
    expect(await screen.findByText(/5 accounts synced/)).toBeInTheDocument()
    expect(screen.getByText(/4 synced, 1 failed/)).toBeInTheDocument()
    expect(screen.getByText('Sync failed')).toBeInTheDocument()
  })

  it('success entry shows green checkmark icon and "{N} accounts synced" text', async () => {
    mockGetSyncLog.mockResolvedValue({
      entries: [makeEntry({ id: 1, status: 'success', accounts_fetched: 12 })],
    })
    render(<SyncHistory />)
    expect(await screen.findByText('12 accounts synced')).toBeInTheDocument()
    // Check for the success icon (green checkmark via data-testid)
    expect(screen.getByTestId('sync-icon-success')).toBeInTheDocument()
  })

  it('partial entry shows amber warning icon and "{N} synced, {M} failed" text', async () => {
    mockGetSyncLog.mockResolvedValue({
      entries: [makeEntry({ id: 1, status: 'partial', accounts_fetched: 10, accounts_failed: 2 })],
    })
    render(<SyncHistory />)
    expect(await screen.findByText('10 synced, 2 failed')).toBeInTheDocument()
    expect(screen.getByTestId('sync-icon-partial')).toBeInTheDocument()
  })

  it('failed entry shows red X icon and "Sync failed" text', async () => {
    mockGetSyncLog.mockResolvedValue({
      entries: [makeEntry({ id: 1, status: 'failed', error_text: 'Network error' })],
    })
    render(<SyncHistory />)
    expect(await screen.findByText('Sync failed')).toBeInTheDocument()
    expect(screen.getByTestId('sync-icon-failed')).toBeInTheDocument()
  })

  it('clicking a failed entry expands its error detail', async () => {
    const user = userEvent.setup()
    mockGetSyncLog.mockResolvedValue({
      entries: [makeEntry({ id: 1, status: 'failed', error_text: 'Connection refused' })],
    })
    render(<SyncHistory />)
    const row = await screen.findByRole('button', { name: /Sync failed/i })
    expect(row).toHaveAttribute('aria-expanded', 'false')

    await user.click(row)
    expect(row).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('Connection refused')).toBeVisible()
  })

  it('clicking an expanded entry collapses it', async () => {
    const user = userEvent.setup()
    mockGetSyncLog.mockResolvedValue({
      entries: [makeEntry({ id: 1, status: 'failed', error_text: 'Timeout' })],
    })
    render(<SyncHistory />)
    const row = await screen.findByRole('button', { name: /Sync failed/i })

    await user.click(row) // expand
    expect(row).toHaveAttribute('aria-expanded', 'true')

    await user.click(row) // collapse
    expect(row).toHaveAttribute('aria-expanded', 'false')
  })

  it('expanding a second entry collapses the first (accordion)', async () => {
    const user = userEvent.setup()
    mockGetSyncLog.mockResolvedValue({
      entries: [
        makeEntry({ id: 1, status: 'failed', error_text: 'Error one' }),
        makeEntry({ id: 2, status: 'partial', accounts_fetched: 3, accounts_failed: 2, error_text: 'Error two' }),
      ],
    })
    render(<SyncHistory />)

    const rows = await screen.findAllByRole('button')
    const row1 = rows[0]
    const row2 = rows[1]

    await user.click(row1) // expand first
    expect(row1).toHaveAttribute('aria-expanded', 'true')
    expect(row2).toHaveAttribute('aria-expanded', 'false')

    await user.click(row2) // expand second, should collapse first
    expect(row1).toHaveAttribute('aria-expanded', 'false')
    expect(row2).toHaveAttribute('aria-expanded', 'true')
  })

  it('success entries are not clickable/expandable', async () => {
    mockGetSyncLog.mockResolvedValue({
      entries: [makeEntry({ id: 1, status: 'success', accounts_fetched: 8 })],
    })
    render(<SyncHistory />)
    await screen.findByText('8 accounts synced')

    // Success entry should not have role="button" or be expandable
    const buttons = screen.queryAllByRole('button')
    expect(buttons).toHaveLength(0)
  })
})
