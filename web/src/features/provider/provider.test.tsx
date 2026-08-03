// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it } from 'vitest'
import type { AccountOperations, OperationBackend, OperationsAPI, ReferencePrice } from '../../lib/api'
import type { MarketWriter, OfferInput, SubmittedTransaction } from '../../lib/marketContract'
import { ProviderConsole } from './ProviderConsole'
import { Offers } from './Offers'

afterEach(cleanup)

it('shows real machines, immutable offer versions, collateral, and earnings', async () => {
  const operations: AccountOperations = {
    chain_id: 10143, contract_address: '0x4444444444444444444444444444444444444444', explorer_url: 'https://testnet.monadexplorer.com', confirmations: 2,
    wallet_address: '0x1111111111111111111111111111111111111111', customer_balance_wei: '0', provider_bond_wei: '2000', claimable_wei: '3000', provider_earnings_wei: '95', bond_exit_available_at: 0,
    sessions: [], machines: [{ id: 'machine-1', name: 'studio-node', revoked: false, backends: [{ id: 'backend:machine-1:local-qwen', kind: 'ollama', model: 'qwen', enabled: true, healthy: true, capacity: 2 }] }],
    offers: [{ offer_id: '0xoffer', version: 4, model_hash: '0xmodel', capability_hash: '0xcap', input_per_million_wei: '10', output_per_million_wei: '20', compute_per_second_wei: '30' }],
  }
  const api = { operations: async () => operations } as OperationsAPI
  let bond = 0n
  let published: OfferInput | undefined
  const confirmed: SubmittedTransaction = { hash: '0xbond', confirm: async () => ({ offerVersion: 1 }) }
  const writer = { depositBond: async (value: bigint) => { bond = value; return confirmed }, publishOffer: async (offer:OfferInput)=>{published=offer;return confirmed} } as unknown as MarketWriter
  const client = new QueryClient()
  client.setQueryData<ReferencePrice>(['reference-price'], { symbol: 'MON', usd_per_mon: '0.02', source: 'CoinGecko', updated_at: new Date().toISOString() })
  render(<QueryClientProvider client={client}><ProviderConsole api={api} writer={writer} /></QueryClientProvider>)
  expect(await screen.findByText('studio-node')).toBeVisible()
  expect(screen.getByRole('region', { name: /provider collateral/i })).toBeVisible()
  expect(screen.getByRole('combobox', { name: /backend and model/i })).toHaveValue('backend:machine-1:local-qwen')
  expect(screen.getByRole('group', { name: /usage pricing/i })).toBeVisible()
  expect(screen.getByLabelText(/bond amount/i)).toHaveAccessibleDescription(/minimum 5 mon/i)
  expect(screen.getByLabelText(/input tokens/i)).toHaveAccessibleDescription(/usd.*one million input/i)
  expect(screen.getByLabelText(/output tokens/i)).toHaveAccessibleDescription(/usd.*one million output/i)
  expect(screen.getByLabelText(/compute time/i)).toHaveAccessibleDescription(/usd.*minute/i)
  expect(screen.getByText(/new immutable on-chain price version/i)).toBeVisible()
  expect(screen.getByText(/version 4/i)).toBeVisible()
  expect(screen.getByText(/0\.000000000000000095 MON/i)).toBeVisible()
  await userEvent.type(screen.getByLabelText(/bond amount/i), '2')
  await userEvent.click(screen.getByRole('button', { name: /deposit collateral/i }))
  expect(bond).toBe(2_000_000_000_000_000_000n)
  await userEvent.type(screen.getByLabelText(/input tokens/i),'0.10');await userEvent.type(screen.getByLabelText(/output tokens/i),'0.20');await userEvent.type(screen.getByLabelText(/compute time/i),'0.06');await userEvent.click(screen.getByRole('button',{name:/publish offer/i}))
  expect(published).toEqual({ offerID: 'local-qwen', model: 'qwen', capabilities: ['text','stream'], inputPerMillion: 5_000_000_000_000_000_000n, outputPerMillion: 10_000_000_000_000_000_000n, computePerSecond: 50_000_000_000_000_000n })
	expect(screen.getByText(/activate published version 1/i)).toBeVisible()
})

it('selects a backend discovered after the provider screen opens', async () => {
  const client = new QueryClient()
  const writer = { publishOffer: async () => ({ hash: '0x1', confirm: async () => undefined }) } as unknown as MarketWriter
  const submit = async (action:()=>Promise<SubmittedTransaction>) => { const transaction=await action(); return transaction.confirm() }
  const view = render(<QueryClientProvider client={client}><Offers offers={[]} backends={[]} writer={writer} submit={submit} /></QueryClientProvider>)
  const discovered: OperationBackend[] = [{ id: 'backend:machine-1:local', kind: 'ollama', model: 'qwen', enabled: true, healthy: false, capacity: 0 }]
  view.rerender(<QueryClientProvider client={client}><Offers offers={[]} backends={discovered} writer={writer} submit={submit} /></QueryClientProvider>)
  expect(screen.getByRole('combobox', { name: /backend and model/i })).toHaveValue('backend:machine-1:local')
	await userEvent.type(screen.getByLabelText(/input tokens/i), '0.1')
	await userEvent.type(screen.getByLabelText(/output tokens/i), '0.1')
	await userEvent.type(screen.getByLabelText(/compute time/i), '0.1')
	expect(screen.getByRole('button', { name: /publish offer/i })).toBeEnabled()
})

it('does not activate an unconfirmed offer version', async () => {
  const client = new QueryClient()
  const backend: OperationBackend[] = [{ id: 'backend:machine-1:local', kind: 'ollama', model: 'qwen', enabled: true, healthy: false, capacity: 0 }]
  render(<QueryClientProvider client={client}><Offers offers={[]} backends={backend} writer={{ publishOffer: async () => ({ hash: '0x1', confirm: async () => undefined }) } as unknown as MarketWriter} submit={async()=>undefined} /></QueryClientProvider>)
  await userEvent.type(screen.getByLabelText(/input tokens/i), '0.1')
  await userEvent.type(screen.getByLabelText(/output tokens/i), '0.1')
  await userEvent.type(screen.getByLabelText(/compute time/i), '0.1')
  await userEvent.click(screen.getByRole('button', { name: /publish offer/i }))
  expect(screen.queryByText(/activate published version/i)).not.toBeInTheDocument()
  expect(screen.getByRole('alert')).toHaveTextContent(/could not be confirmed from monad/i)
})

it('never converts wallet values with an expired USD quote', async () => {
  const operations: AccountOperations = {
    chain_id: 10143, contract_address: '0x4444444444444444444444444444444444444444', explorer_url: '', confirmations: 2,
    wallet_address: '0x1111111111111111111111111111111111111111', customer_balance_wei: '0', provider_bond_wei: '0', claimable_wei: '0', provider_earnings_wei: '0', bond_exit_available_at: 0, sessions: [],
    machines: [{ id: 'machine-1', name: 'node', revoked: false, backends: [{ id: 'backend:machine-1:local', kind: 'ollama', model: 'qwen', enabled: true, healthy: false, capacity: 0 }] }], offers: [],
  }
  const api = { operations: async () => operations } as OperationsAPI
  const client = new QueryClient()
  client.setQueryData<ReferencePrice>(['reference-price'], { symbol: 'MON', usd_per_mon: '0.02', source: 'CoinGecko', updated_at: '2020-01-01T00:00:00Z' })
  render(<QueryClientProvider client={client}><ProviderConsole api={api} writer={{} as MarketWriter} /></QueryClientProvider>)
  expect(await screen.findByText(/live usd quote is unavailable/i)).toBeVisible()
  expect(screen.getAllByText('MON / 1M')).toHaveLength(2)
})
