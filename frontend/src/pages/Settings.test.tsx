import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import Settings from './Settings'
import * as client from '../api/client'

vi.mock('../api/client')

const mockGetSettings = vi.mocked(client.getSettings)
const mockSaveSettings = vi.mocked(client.saveSettings)
const mockTriggerSync = vi.mocked(client.triggerSync)

const notConfiguredResponse: client.SettingsResponse = {
  configured: false,
  last_sync_at: null,
  last_sync_status: null,
}

const configuredResponse: client.SettingsResponse = {
  configured: true,
  last_sync_at: '2026-03-15T06:00:00Z',
  last_sync_status: 'success',
}

describe('Settings', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  describe('TestSettings_RendersForm', () => {
    it('renders an input for access URL and a Save button', async () => {
      mockGetSettings.mockResolvedValue(notConfiguredResponse)
      render(<Settings onNavigateDashboard={() => {}} />)

      expect(screen.getByRole('textbox')).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /save/i })).toBeInTheDocument()
    })
  })

  describe('TestSettings_ShowsNotConfigured', () => {
    it('shows "Not configured" status when not configured', async () => {
      mockGetSettings.mockResolvedValue(notConfiguredResponse)
      render(<Settings onNavigateDashboard={() => {}} />)

      await waitFor(() => {
        expect(screen.getByText(/not configured/i)).toBeInTheDocument()
      })
    })
  })

  describe('TestSettings_ShowsConfigured', () => {
    it('shows "Configured" status and last sync time when configured', async () => {
      mockGetSettings.mockResolvedValue(configuredResponse)
      render(<Settings onNavigateDashboard={() => {}} />)

      await waitFor(() => {
        expect(screen.getByText(/configured/i)).toBeInTheDocument()
      })
      // last sync time should appear (success status)
      await waitFor(() => {
        expect(screen.getByText(/success/i)).toBeInTheDocument()
      })
    })
  })

  describe('TestSettings_SaveURL', () => {
    it('entering a URL and clicking Save calls saveSettings with the URL, then refreshes status', async () => {
      mockGetSettings
        .mockResolvedValueOnce(notConfiguredResponse)
        .mockResolvedValueOnce(configuredResponse)
      mockSaveSettings.mockResolvedValue({ ok: true })

      render(<Settings onNavigateDashboard={() => {}} />)
      await waitFor(() => screen.getByText(/not configured/i))

      const input = screen.getByRole('textbox')
      await userEvent.type(input, 'https://bridge.simplefin.org/simplefin/b64token')

      const saveButton = screen.getByRole('button', { name: /save/i })
      await userEvent.click(saveButton)

      await waitFor(() =>
        expect(mockSaveSettings).toHaveBeenCalledWith(
          'https://bridge.simplefin.org/simplefin/b64token'
        )
      )
      await waitFor(() => expect(mockGetSettings).toHaveBeenCalledTimes(2))
    })
  })

  describe('TestSettings_SaveEmptyURL', () => {
    it('Save button is disabled when input is empty', async () => {
      mockGetSettings.mockResolvedValue(notConfiguredResponse)
      render(<Settings onNavigateDashboard={() => {}} />)

      const saveButton = screen.getByRole('button', { name: /save/i })
      expect(saveButton).toBeDisabled()
    })
  })

  describe('TestSettings_SyncNow', () => {
    it('clicking Sync Now calls triggerSync', async () => {
      mockGetSettings
        .mockResolvedValueOnce(configuredResponse)
        .mockResolvedValueOnce(configuredResponse)
      mockTriggerSync.mockResolvedValue({ ok: true })

      render(<Settings onNavigateDashboard={() => {}} />)
      await waitFor(() => screen.getByText(/configured/i))

      const syncButton = screen.getByRole('button', { name: /sync now/i })
      await userEvent.click(syncButton)

      await waitFor(() => expect(mockTriggerSync).toHaveBeenCalled())
    })
  })

  describe('TestSettings_SyncNowDisabled', () => {
    it('Sync Now button is disabled when not configured', async () => {
      mockGetSettings.mockResolvedValue(notConfiguredResponse)
      render(<Settings onNavigateDashboard={() => {}} />)

      await waitFor(() => screen.getByText(/not configured/i))

      const syncButton = screen.getByRole('button', { name: /sync now/i })
      expect(syncButton).toBeDisabled()
    })
  })
})
