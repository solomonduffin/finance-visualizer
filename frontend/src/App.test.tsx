import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import App from './App'
import * as client from './api/client'
import * as darkModeHook from './hooks/useDarkMode'

vi.mock('./api/client')
vi.mock('./hooks/useDarkMode')
// Mock the Dashboard page to avoid needing full API setup for App tests
vi.mock('./pages/Dashboard', () => ({
  default: () => <div data-testid="dashboard-page">Dashboard Page</div>,
}))

const mockCheckAuth = vi.mocked(client.checkAuth)
const mockUseDarkMode = vi.mocked(darkModeHook.useDarkMode)

describe('App', () => {
  beforeEach(() => {
    vi.resetAllMocks()
    // Default: authenticated, dark mode off
    mockCheckAuth.mockResolvedValue(true)
    mockUseDarkMode.mockReturnValue({ isDark: false, toggle: vi.fn() })
  })

  describe('NavBar dark mode toggle', () => {
    it('renders a dark mode toggle button in NavBar', async () => {
      render(<App />)

      await waitFor(() => {
        // Toggle button should be in the nav
        const toggleButton = screen.getByRole('button', { name: /dark mode|light mode|toggle/i })
        expect(toggleButton).toBeInTheDocument()
      })
    })

    it('clicking toggle calls useDarkMode toggle function', async () => {
      const mockToggle = vi.fn()
      mockUseDarkMode.mockReturnValue({ isDark: false, toggle: mockToggle })

      render(<App />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /dark mode|light mode|toggle/i })).toBeInTheDocument()
      })

      const toggleButton = screen.getByRole('button', { name: /dark mode|light mode|toggle/i })
      await userEvent.click(toggleButton)

      expect(mockToggle).toHaveBeenCalledOnce()
    })

    it('shows moon icon when isDark is false (clicking will enable dark mode)', async () => {
      mockUseDarkMode.mockReturnValue({ isDark: false, toggle: vi.fn() })

      render(<App />)

      await waitFor(() => {
        const button = screen.getByRole('button', { name: /dark mode|light mode|toggle/i })
        expect(button).toBeInTheDocument()
      })

      // When isDark is false, button should indicate "enable dark mode" (moon icon)
      const button = screen.getByRole('button', { name: /dark mode|light mode|toggle/i })
      expect(button).toHaveAttribute('aria-label', expect.stringMatching(/dark mode/i))
    })

    it('shows sun icon when isDark is true (clicking will disable dark mode)', async () => {
      mockUseDarkMode.mockReturnValue({ isDark: true, toggle: vi.fn() })

      render(<App />)

      await waitFor(() => {
        const button = screen.getByRole('button', { name: /dark mode|light mode|toggle/i })
        expect(button).toBeInTheDocument()
      })

      // When isDark is true, button should indicate "enable light mode" (sun icon)
      const button = screen.getByRole('button', { name: /dark mode|light mode|toggle/i })
      expect(button).toHaveAttribute('aria-label', expect.stringMatching(/light mode/i))
    })
  })

  describe('Dashboard route', () => {
    it('renders the real Dashboard component at "/" route', async () => {
      render(<App />)

      await waitFor(() => {
        expect(screen.getByTestId('dashboard-page')).toBeInTheDocument()
      })
    })
  })
})
