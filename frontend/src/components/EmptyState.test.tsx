import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { EmptyState } from './EmptyState'

describe('EmptyState', () => {
  it('renders "Connect your accounts" message', () => {
    render(
      <MemoryRouter>
        <EmptyState />
      </MemoryRouter>
    )
    expect(screen.getByText(/connect your accounts/i)).toBeInTheDocument()
  })

  it('renders a link to /settings', () => {
    render(
      <MemoryRouter>
        <EmptyState />
      </MemoryRouter>
    )
    const link = screen.getByRole('link')
    expect(link).toHaveAttribute('href', '/settings')
  })
})
