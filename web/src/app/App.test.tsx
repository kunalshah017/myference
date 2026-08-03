import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { afterEach, expect, it } from 'vitest'
import App from './App'

afterEach(cleanup)

it('shows an honest disconnected marketplace without fake inventory', async () => {
  window.history.pushState({}, '', '/app')
  const user = userEvent.setup()
  render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><App /></QueryClientProvider>)

  expect(
    screen.getByRole('heading', { name: /workspace overview/i }),
  ).toBeVisible()
  await user.click(screen.getByRole('button', { name: /models/i }))
  expect(
    await screen.findByText(/live marketplace data is not connected/i),
  ).toBeVisible()
  expect(screen.getByRole('button', { name: /connect wallet/i })).toBeEnabled()
  expect(
    screen.queryByText(/mock|demo provider|sample balance/i),
  ).not.toBeInTheDocument()
})

it('opens the machine approval workspace from the device verification link', () => {
  window.history.pushState({}, '', '/devices')
  render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><App /></QueryClientProvider>)

  expect(screen.getByRole('heading', { name: /connect your application/i })).toBeVisible()
  expect(screen.getByRole('button', { name: /api access/i })).toHaveAttribute('aria-current', 'page')
  expect(screen.getByText(/connect a wallet to create scoped api keys and approve provider devices/i)).toBeVisible()
})
