import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import Login from './Login'
import * as client from '../api/client'

vi.mock('../api/client')

const mockLogin = vi.mocked(client.login)

describe('Login', () => {
  const onSuccess = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders password input and submit button', () => {
    render(<Login onSuccess={onSuccess} />)
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument()
  })

  it('submitting with password calls login API', async () => {
    mockLogin.mockResolvedValueOnce({ ok: true })
    render(<Login onSuccess={onSuccess} />)

    await userEvent.type(screen.getByLabelText(/password/i), 'testpass')
    await userEvent.click(screen.getByRole('button', { name: /sign in/i }))

    await waitFor(() => expect(mockLogin).toHaveBeenCalledWith('testpass'))
  })

  it('calls onSuccess after successful login', async () => {
    mockLogin.mockResolvedValueOnce({ ok: true })
    render(<Login onSuccess={onSuccess} />)

    await userEvent.type(screen.getByLabelText(/password/i), 'testpass')
    await userEvent.click(screen.getByRole('button', { name: /sign in/i }))

    await waitFor(() => expect(onSuccess).toHaveBeenCalled())
  })

  it('displays error message on 401 response (invalid password)', async () => {
    mockLogin.mockResolvedValueOnce({ error: 'invalid password' })
    render(<Login onSuccess={onSuccess} />)

    await userEvent.type(screen.getByLabelText(/password/i), 'wrongpass')
    await userEvent.click(screen.getByRole('button', { name: /sign in/i }))

    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent('Invalid password')
    )
    expect(onSuccess).not.toHaveBeenCalled()
  })

  it('displays rate limit message on 429 response', async () => {
    mockLogin.mockResolvedValueOnce({ error: 'rate_limited' })
    render(<Login onSuccess={onSuccess} />)

    await userEvent.type(screen.getByLabelText(/password/i), 'anypass')
    await userEvent.click(screen.getByRole('button', { name: /sign in/i }))

    await waitFor(() =>
      expect(screen.getByRole('alert')).toHaveTextContent(
        'Too many attempts. Please wait.'
      )
    )
    expect(onSuccess).not.toHaveBeenCalled()
  })
})
