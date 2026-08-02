import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { expect, it } from 'vitest'
import App from './App'

it('shows an honest disconnected marketplace without fake inventory', async () => {
  render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><App /></QueryClientProvider>)

  expect(
    screen.getByRole('heading', { name: /unused machines, useful inference/i }),
  ).toBeVisible()
  expect(
    await screen.findByText(/live marketplace data is not connected/i),
  ).toBeVisible()
  expect(screen.getByRole('button', { name: /connect wallet/i })).toBeEnabled()
  expect(
    screen.queryByText(/mock|demo provider|sample balance/i),
  ).not.toBeInTheDocument()
})
