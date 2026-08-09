import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'
import type { AuthAPI } from '../lib/api'
import { captureEvent } from '../lib/analytics'
import { DashboardShell } from './DashboardShell'

vi.mock('../lib/analytics', () => ({ captureEvent: vi.fn() }))

afterEach(() => { cleanup(); localStorage.clear(); vi.clearAllMocks() })

const authAPI = { session: async () => undefined } as AuthAPI

it('lets one account switch from consuming to hosting inference', async () => {
  const user = userEvent.setup()
  render(<QueryClientProvider client={new QueryClient()}><DashboardShell authAPI={authAPI} /></QueryClientProvider>)

  await user.click(await screen.findByRole('button', { name: /skip for now/i }))
  expect(screen.getByRole('heading', { name: /workspace overview/i })).toBeVisible()
  expect(screen.getByRole('link', { name: /documentation/i })).toHaveAttribute('href', '/docs')
	await user.click(screen.getByRole('button', { name: /provider account/i }))
	expect(screen.getByRole('heading', { name: /collateral and offer pricing/i })).toBeVisible()
	expect(screen.getByText(/connect a wallet to manage provider collateral/i)).toBeVisible()
  expect(captureEvent).toHaveBeenCalledWith('onboarding_skipped', { role: 'consumer' })
  expect(captureEvent).toHaveBeenCalledWith('dashboard_viewed', { view: 'hosting' })
})

it('starts first-time visitors in onboarding and resumes from the dashboard reminder', async () => {
  const user = userEvent.setup()
  render(<QueryClientProvider client={new QueryClient()}><DashboardShell authAPI={authAPI} /></QueryClientProvider>)

  expect(await screen.findByRole('heading', { name: /what do you want to do first/i })).toBeVisible()
  await user.click(screen.getByRole('button', { name: /skip for now/i }))
  expect(screen.getByText(/finish your first live route/i)).toBeVisible()
  expect(localStorage.getItem('myference:onboarding-skipped')).toBe('true')
  await user.click(screen.getByRole('button', { name: /continue setup/i }))
  expect(screen.getByRole('heading', { name: /connect your account/i })).toBeVisible()
  expect(captureEvent).toHaveBeenCalledWith('onboarding_resumed', { role: 'consumer' })
})

it('remembers the provider path when onboarding is skipped', async () => {
  const user = userEvent.setup()
  render(<QueryClientProvider client={new QueryClient()}><DashboardShell authAPI={authAPI} /></QueryClientProvider>)
  await user.click(await screen.findByRole('button', { name: /host your ai inference/i }))
  expect(captureEvent).toHaveBeenCalledWith('onboarding_role_selected', { role: 'provider' })
  await user.click(screen.getByRole('button', { name: /skip for now/i }))
  expect(localStorage.getItem('myference:onboarding-role')).toBe('provider')
  expect(screen.getByText(/connect a machine, bond collateral, and publish a live offer/i)).toBeVisible()
})

it('keeps API keys and provider-device authorization in separate destinations', async () => {
  localStorage.setItem('myference:onboarding-skipped', 'true')
  const user = userEvent.setup()
  render(<QueryClientProvider client={new QueryClient()}><DashboardShell authAPI={authAPI} initialView="api" /></QueryClientProvider>)

  expect(await screen.findByRole('button', { name: /api keys/i })).toHaveAttribute('aria-current', 'page')
  expect(screen.getByRole('heading', { name: /connect your application/i })).toBeVisible()
  expect(screen.queryByRole('heading', { name: /approve this exact machine/i })).not.toBeInTheDocument()

  await user.click(screen.getByRole('button', { name: /^devices$/i }))
  expect(screen.getByRole('heading', { name: /authorize a provider device/i })).toBeVisible()
  expect(screen.queryByRole('heading', { name: /connect your application/i })).not.toBeInTheDocument()
})
