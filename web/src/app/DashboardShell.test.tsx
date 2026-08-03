import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { expect, it } from 'vitest'
import { DashboardShell } from './DashboardShell'

it('lets one account switch from consuming to hosting inference', async () => {
  const user = userEvent.setup()
  render(<QueryClientProvider client={new QueryClient()}><DashboardShell /></QueryClientProvider>)

  expect(screen.getByRole('heading', { name: /workspace overview/i })).toBeVisible()
  expect(screen.getByRole('link', { name: /documentation/i })).toHaveAttribute('href', '/docs')
  await user.click(screen.getByRole('button', { name: /host inference/i }))
  expect(screen.getByRole('heading', { name: /host inference/i })).toBeVisible()
  expect(screen.getByText(/connect a wallet to manage provider machines/i)).toBeVisible()
})
