// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'
import { APIError, type AccountAnalytics, type AccountOperations, type AnalyticsAPI, type AuthAPI, type InferenceAPI, type MarketplaceAPI, type OperationsAPI, type ReferencePriceAPI, type Session } from '../../lib/api'
import { OnboardingFlow } from './OnboardingFlow'

afterEach(() => { cleanup(); vi.restoreAllMocks() })

const session: Session = { account_id: 'acct-1', wallet_address: '0x1111111111111111111111111111111111111111', expires_at: '2026-08-04T00:00:00Z' }
const operations: AccountOperations = {
  chain_id: 10143, contract_address: '0x4444444444444444444444444444444444444444', explorer_url: 'https://testnet.monadexplorer.com', confirmations: 2,
  wallet_address: session.wallet_address, customer_balance_wei: '0', provider_bond_wei: '0', claimable_wei: '0', provider_earnings_wei: '0', bond_exit_available_at: 0,
  sessions: [], machines: [], offers: [],
}
const analytics = { customer: { settled_requests: 0 }, provider: { settled_requests: 0 } } as AccountAnalytics
const liveModels = [
  { model: 'costly', available_providers: 1, total_capacity: 1, minimum_input_per_million_wei: '9000', minimum_output_per_million_wei: '9000', minimum_compute_per_second_wei: '9', stale: false },
  { model: 'recommended', available_providers: 1, total_capacity: 2, minimum_input_per_million_wei: '1000', minimum_output_per_million_wei: '1000', minimum_compute_per_second_wei: '1', stale: false },
]

function dependencies(models = liveModels) {
  return {
    authAPI: { listAPIKeys: async () => [], createAPIKey: vi.fn(), inspectDevice: vi.fn() } as unknown as AuthAPI,
    operationsAPI: { operations: async () => operations } as OperationsAPI,
    marketplaceAPI: { models: async () => models } as MarketplaceAPI,
    analyticsAPI: { analytics: async () => analytics } as AnalyticsAPI,
    referencePriceAPI: { price: async () => ({ symbol: 'MON', usd_per_mon: '1', source: 'test', updated_at: new Date().toISOString() }) } as ReferencePriceAPI,
    inferenceAPI: { chat: vi.fn() } as unknown as InferenceAPI,
  }
}

function renderFlow(props: Partial<React.ComponentProps<typeof OnboardingFlow>> = {}, models = liveModels) {
  return render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><OnboardingFlow {...dependencies(models)} {...props} /></QueryClientProvider>)
}

it('prioritizes using inference while offering hosting and skip paths', async () => {
  const skipped = vi.fn()
  renderFlow({ onSkip: skipped })
  expect(screen.getByRole('heading', { name: /what do you want to do first/i })).toBeVisible()
  expect(screen.getByRole('button', { name: /use ai inference/i })).toBeVisible()
  expect(screen.getByRole('button', { name: /host your ai inference/i })).toBeVisible()
  await userEvent.click(screen.getByRole('button', { name: /skip for now/i }))
  expect(skipped).toHaveBeenCalledOnce()
})

it('selects the cheapest healthy model and lets the customer change it', async () => {
  renderFlow({ session, initialRole: 'consumer' })
  const select = await screen.findByRole('combobox', { name: /model/i })
  expect(select).toHaveDisplayValue('recommended')
  await userEvent.selectOptions(select, 'costly')
  expect(select).toHaveDisplayValue('costly')
  expect(screen.getByText(/recommended starter amount/i)).toBeVisible()
  expect(screen.getByLabelText(/deposit amount/i)).toHaveValue('0.000000000000001329')
})

it('returns an expired account session to wallet connection', async () => {
  const deps = dependencies()
  deps.operationsAPI.operations = async () => { throw new APIError(401, 'unauthorized') }
  render(<QueryClientProvider client={new QueryClient()}><OnboardingFlow {...deps} session={session} initialRole="consumer" /></QueryClientProvider>)
  expect(await screen.findByRole('heading', { name: /connect your account/i })).toBeVisible()
  expect(screen.getByRole('button', { name: /connect wallet/i })).toBeVisible()
})

it('lets a provider retry a temporary account-state failure', async () => {
  const deps = dependencies()
  deps.operationsAPI.operations = async () => { throw new APIError(503, 'indexer unavailable') }
  render(<QueryClientProvider client={new QueryClient()}><OnboardingFlow {...deps} session={session} initialRole="provider" /></QueryClientProvider>)
  expect(await screen.findByRole('alert')).toHaveTextContent(/account state could not be loaded/i)
  expect(screen.getByRole('button', { name: /retry account state/i })).toBeVisible()
})

it('exposes current and completed progress to assistive technology', async () => {
  renderFlow({ session, initialRole: 'consumer' })
  const current = await screen.findByRole('listitem', { name: 'Fund requests, current step' })
  expect(current).toHaveAttribute('aria-current', 'step')
  expect(screen.getByRole('listitem', { name: 'Connect account, complete' })).toBeVisible()
  expect(screen.getByRole('main')).toHaveAttribute('aria-live', 'polite')
})

it('shows an honest recovery state when no provider is live', async () => {
  renderFlow({ session, initialRole: 'consumer' }, [])
  expect(await screen.findByRole('alert')).toHaveTextContent(/no live inference models/i)
  expect(screen.getByRole('button', { name: /refresh inventory/i })).toBeVisible()
})

it('explains that an existing API key secret cannot be recovered', async () => {
  const deps = dependencies()
  deps.authAPI.listAPIKeys = async () => [{ id: 'key-1', scope: { models: ['recommended'], endpoints: ['/v1/chat/completions'], max_spend_wei: '1000' } }]
  render(<QueryClientProvider client={new QueryClient()}><OnboardingFlow {...deps} session={session} initialRole="consumer" /></QueryClientProvider>)
  expect(await screen.findByText(/existing key's secret cannot be recovered/i)).toBeVisible()
  expect(screen.getByRole('button', { name: /create replacement key/i })).toBeVisible()
})

it('takes providers from CLI setup into an unattended native service', async () => {
  renderFlow({ session, initialRole: 'provider' })
  expect(await screen.findByRole('heading', { name: /turn this computer into a provider/i })).toBeVisible()
  expect(screen.getByText('myference host')).toBeVisible()
  expect(screen.getByText('myference service install')).toBeVisible()
  expect(screen.getByLabelText(/device code/i)).toBeVisible()
})

it('uses the same output-token ceiling for the starter estimate and live request', async () => {
  const deps = dependencies()
  const issued: Array<{ id: string; scope: { models: string[]; endpoints: string[]; max_spend_wei: string } }> = []
  deps.authAPI.listAPIKeys = async () => issued
  deps.authAPI.createAPIKey = async (scope) => {
    const key = { id: 'key-new', token: 'mf_secret', scope }
    issued.push({ id: key.id, scope })
    return key
  }
  deps.operationsAPI.operations = async () => ({ ...operations, customer_balance_wei: '1000000', sessions: [{ session_id: '0x01', allowance_wei: '1000000', spent_wei: '0', expires_at: Math.floor(Date.now() / 1000) + 3600, close_available_at: 0, finalized: false }] })
  deps.inferenceAPI.chat = vi.fn().mockResolvedValue('A live provider response.')
  const completed = vi.fn()
  render(<QueryClientProvider client={new QueryClient()}><OnboardingFlow {...deps} session={session} initialRole="consumer" onComplete={completed} /></QueryClientProvider>)

  await userEvent.click(await screen.findByRole('button', { name: /create api key/i }))
  await userEvent.click(await screen.findByRole('button', { name: /send live request/i }))
  expect(deps.inferenceAPI.chat).toHaveBeenCalledWith('recommended', 'mf_secret', expect.any(String), expect.any(Array), 1_000)
  expect(await screen.findByText('A live provider response.')).toBeVisible()
  expect(completed).toHaveBeenCalledWith('consumer')
  await userEvent.selectOptions(screen.getByRole('combobox', { name: /model/i }), 'costly')
  expect(await screen.findByRole('button', { name: /create replacement key/i })).toBeVisible()
})
