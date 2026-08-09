// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'
import type { ProviderAPI, ProviderAccount, ProviderAction, ProviderActionInput } from '../../lib/api'
import type { MarketWriter, OfferInput, SubmittedTransaction } from '../../lib/marketContract'
import { ProviderConsole } from './ProviderConsole'
import { ProviderApproval } from './ProviderApproval'
import { assertProviderWallet, executeProviderAction } from './providerAction'

afterEach(cleanup)

const providerAccount: ProviderAccount = {
  chain_id: 10143, contract_address: '0x4444444444444444444444444444444444444444', explorer_url: 'https://testnet.monadexplorer.com', confirmations: 2,
  wallet_address: '0x1111111111111111111111111111111111111111', minimum_bond_wei: '5000000000000000000', provider_bond_wei: '2000', claimable_wei: '3000', provider_earnings_wei: '95', bond_exit_available_at: 0,
  offers: [{ offer_id: 'local-qwen', model: 'qwen', backend_kind: 'ollama', capabilities: ['text', 'stream'], metering_mode: 'tokens_and_compute', version: 4, input_per_million_wei: '100000000000000000', output_per_million_wei: '200000000000000000', compute_per_second_wei: '300000000000000000' }],
}

function action(input: ProviderActionInput, status: ProviderAction['status'] = 'pending_wallet'): ProviderAction {
  return { id: 'action-1', status, wallet_address: providerAccount.wallet_address, expires_at: new Date(Date.now() + 60_000).toISOString(), ...input }
}

it('shows collateral and only reprices account-owned existing offers', async () => {
  const created: ProviderActionInput[] = []
  const api = { account: async () => providerAccount, create: async (input: ProviderActionInput) => { created.push(input); return action(input) }, submitted: async (_id: string, _hashes: string[]) => action(created.at(-1)!, 'pending_chain'), get: async () => action(created.at(-1)!, 'confirmed') } as unknown as ProviderAPI
  let deposited = 0n
  let published: OfferInput | undefined
  const transaction: SubmittedTransaction = { hash: '0x1111111111111111111111111111111111111111111111111111111111111111', confirm: async () => undefined }
  const writer = { depositBond: async (value: bigint) => { deposited = value; return transaction }, publishOffer: async (offer: OfferInput) => { published = offer; return transaction } } as unknown as MarketWriter
  render(<QueryClientProvider client={new QueryClient()}><ProviderConsole api={api} writer={writer}/></QueryClientProvider>)
  expect(await screen.findByRole('heading', { name: /collateral and existing offer pricing/i })).toBeVisible()
  expect(screen.queryByText(/studio-node|discovered backend/i)).not.toBeInTheDocument()
  expect(screen.getByRole('combobox', { name: /existing offer/i })).toHaveValue('local-qwen')
  expect(screen.getByText(/new offers.*created only in the myference cli/i)).toBeVisible()
  await userEvent.type(screen.getByLabelText(/bond amount/i), '2')
  await userEvent.click(screen.getByRole('button', { name: /deposit collateral/i }))
  await waitFor(() => expect(deposited).toBe(2_000_000_000_000_000_000n))
  const input = screen.getByLabelText(/input tokens/i)
  const output = screen.getByLabelText(/output tokens/i)
  const compute = screen.getByLabelText(/compute time/i)
  await userEvent.clear(input); await userEvent.type(input, '1')
  await userEvent.clear(output); await userEvent.type(output, '2')
  await userEvent.clear(compute); await userEvent.type(compute, '3')
  await userEvent.click(screen.getByRole('button', { name: /publish price version/i }))
  await waitFor(() => expect(published).toEqual({ offerID: 'local-qwen', model: 'qwen', capabilities: ['text', 'stream'], inputPerMillion: 1_000_000_000_000_000_000n, outputPerMillion: 2_000_000_000_000_000_000n, computePerSecond: 3_000_000_000_000_000_000n }))
  expect(created.at(-1)?.offers?.[0]).toMatchObject({ offer_id: 'local-qwen', model: 'qwen', kind: 'ollama' })
})

it('executes a CLI draft exactly and waits for indexed confirmation', async () => {
  const draft = action({ kind: 'publish_offer', offers: [{ offer_id: 'cli-offer', model: 'qwen', kind: 'ollama', capabilities: ['stream', 'text'], metering_mode: 'tokens_and_compute', input_per_million_wei: '1', output_per_million_wei: '2', compute_per_second_wei: '3' }] })
  const publishOffer = vi.fn(async () => ({ hash: '0x2222222222222222222222222222222222222222222222222222222222222222', confirm: async () => undefined }))
  const submitted = vi.fn(async () => ({ ...draft, status: 'pending_chain' as const }))
  const api = { submitted, get: async () => ({ ...draft, status: 'confirmed' as const }) } as unknown as ProviderAPI
  const confirmed = await executeProviderAction(draft, { publishOffer } as unknown as MarketWriter, api, async () => undefined)
  expect(publishOffer).toHaveBeenCalledWith({ offerID: 'cli-offer', model: 'qwen', capabilities: ['stream', 'text'], inputPerMillion: 1n, outputPerMillion: 2n, computePerSecond: 3n })
  expect(submitted).toHaveBeenCalledWith(draft.id, ['0x2222222222222222222222222222222222222222222222222222222222222222'])
  expect(confirmed.status).toBe('confirmed')
})

it('renders the dedicated CLI approval page without exposing a hosting form', async () => {
  const draft = action({ kind: 'deposit_collateral', amount_wei: '5' })
  const api = { account: async () => providerAccount, get: async () => draft } as unknown as ProviderAPI
  render(<QueryClientProvider client={new QueryClient()}><ProviderApproval actionID="action-1" api={api} writer={{} as MarketWriter} checkWallet={async () => undefined}/></QueryClientProvider>)
  expect(await screen.findByRole('heading', { name: /approve provider action/i })).toBeVisible()
  expect(screen.getByText(providerAccount.wallet_address)).toBeVisible()
  expect(screen.queryByRole('combobox')).not.toBeInTheDocument()
})

it('rejects the wrong connected wallet before sending a CLI action', async () => {
  Object.assign(window, { ethereum: { request: vi.fn(async () => ['0x2222222222222222222222222222222222222222']) } })
  await expect(assertProviderWallet(action({ kind: 'deposit_collateral', amount_wei: '5' }))).rejects.toThrow(/switch your wallet/i)
  Object.assign(window, { ethereum: undefined })
})
