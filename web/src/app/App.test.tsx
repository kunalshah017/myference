import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { afterEach, expect, it } from 'vitest'
import App from './App'

afterEach(() => { cleanup(); localStorage.clear() })

it('shows an honest disconnected marketplace without fake inventory', async () => {
  window.history.pushState({}, '', '/app')
  localStorage.setItem('myference:onboarding-skipped', 'true')
  const user = userEvent.setup()
  render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><App /></QueryClientProvider>)

  expect(await screen.findByRole('heading', { name: /workspace overview/i })).toBeVisible()
  await user.click(screen.getByRole('button', { name: /models/i }))
  expect(
    await screen.findByText(/live marketplace data is not connected/i),
  ).toBeVisible()
  expect(screen.getByRole('button', { name: /connect wallet/i })).toBeEnabled()
  expect(
    screen.queryByText(/mock|demo provider|sample balance/i),
  ).not.toBeInTheDocument()
})

it('opens the machine approval workspace from the device verification link', async () => {
  window.history.pushState({}, '', '/devices')
  render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><App /></QueryClientProvider>)

  expect(await screen.findByRole('heading', { name: /connect your application/i })).toBeVisible()
  expect(screen.getByRole('button', { name: /api access/i })).toHaveAttribute('aria-current', 'page')
  expect(screen.getByText(/connect a wallet to create scoped api keys and approve provider devices/i)).toBeVisible()
})

it('serves complete public documentation for using and hosting inference', () => {
  window.history.pushState({}, '', '/docs')
  render(<QueryClientProvider client={new QueryClient()}><App /></QueryClientProvider>)

  expect(screen.getByRole('heading', { name: /build with myference/i })).toBeVisible()
  expect(screen.getByRole('heading', { name: /use hosted inference/i })).toBeVisible()
  expect(screen.getByRole('heading', { name: /host from your machine/i })).toBeVisible()
  expect(screen.getByText('https://api.myference.xyz', { exact: true })).toBeVisible()
  expect(screen.getByText(/windows 64-bit/i)).toBeVisible()
  expect(screen.getByText(/macos apple silicon/i, { selector: 'strong' })).toBeVisible()
  expect(screen.getByText('irm https://myference.xyz/install.ps1 | iex', { exact: true })).toBeVisible()
  expect(screen.getByText('curl -fsSL https://myference.xyz/install.sh | sh', { exact: true })).toBeVisible()
  expect(screen.getAllByText(/\/v1\/chat\/completions/i).length).toBeGreaterThan(0)
  expect(screen.getAllByText(/\/v1\/messages/i).length).toBeGreaterThan(0)
  expect(screen.getByRole('heading', { name: /pricing and settlement/i })).toBeVisible()
  expect(screen.getByRole('heading', { name: /security and model evidence/i })).toBeVisible()
  expect(screen.getByRole('heading', { name: /troubleshooting/i })).toBeVisible()
  expect(screen.getAllByText(/public API exposes model responses only/i).length).toBeGreaterThan(0)
  expect(screen.getByText(/starts Docker Desktop and pulls missing digest-pinned images/i)).toBeInTheDocument()
  expect(screen.getByText(/Linux container proxy/i)).toBeVisible()
})
