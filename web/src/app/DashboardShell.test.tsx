import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it } from 'vitest'
import type { AuthAPI } from '../lib/api'
import { DashboardShell } from './DashboardShell'

afterEach(() => { cleanup(); localStorage.clear() })

const authAPI = { session: async () => undefined } as AuthAPI

it('lets one account switch from consuming to hosting inference', async () => {
  const user = userEvent.setup()
  render(<QueryClientProvider client={new QueryClient()}><DashboardShell authAPI={authAPI} /></QueryClientProvider>)

  await user.click(await screen.findByRole('button', { name: /skip for now/i }))
  expect(screen.getByRole('heading', { name: /workspace overview/i })).toBeVisible()
  expect(screen.getByRole('link', { name: /documentation/i })).toHaveAttribute('href', '/docs')
  await user.click(screen.getByRole('button', { name: /host inference/i }))
  expect(screen.getByRole('heading', { name: /host inference/i })).toBeVisible()
  expect(screen.getByText(/connect a wallet to manage provider machines/i)).toBeVisible()
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
})

it('remembers the provider path when onboarding is skipped', async () => {
  const user = userEvent.setup()
  render(<QueryClientProvider client={new QueryClient()}><DashboardShell authAPI={authAPI} /></QueryClientProvider>)
  await user.click(await screen.findByRole('button', { name: /host your ai inference/i }))
  await user.click(screen.getByRole('button', { name: /skip for now/i }))
  expect(localStorage.getItem('myference:onboarding-role')).toBe('provider')
  expect(screen.getByText(/connect a machine, bond collateral, and publish a live offer/i)).toBeVisible()
})
