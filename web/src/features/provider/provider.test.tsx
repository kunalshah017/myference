// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it } from 'vitest'
import type { AccountOperations, OperationsAPI } from '../../lib/api'
import type { MarketWriter, SubmittedTransaction } from '../../lib/marketContract'
import { ProviderConsole } from './ProviderConsole'

afterEach(cleanup)

it('shows real machines, immutable offer versions, collateral, and earnings', async () => {
  const operations: AccountOperations = {
    chain_id: 10143, contract_address: '0x4444444444444444444444444444444444444444', explorer_url: 'https://testnet.monadexplorer.com', confirmations: 2,
    wallet_address: '0x1111111111111111111111111111111111111111', customer_balance_wei: '0', provider_bond_wei: '2000', claimable_wei: '3000', provider_earnings_wei: '95', bond_exit_available_at: 0,
    sessions: [], machines: [{ id: 'machine-1', name: 'studio-node', revoked: false, backends: [{ id: 'backend-1', kind: 'ollama', model: 'qwen', enabled: true, healthy: true, capacity: 2 }] }],
    offers: [{ offer_id: '0xoffer', version: 4, model_hash: '0xmodel', capability_hash: '0xcap', input_per_million_wei: '10', output_per_million_wei: '20', compute_per_second_wei: '30' }],
  }
  const api = { operations: async () => operations } as OperationsAPI
  let bond = 0n
  let capabilities: string[] = []
  const confirmed: SubmittedTransaction = { hash: '0xbond', confirm: async () => undefined }
  const writer = { depositBond: async (value: bigint) => { bond = value; return confirmed }, publishOffer: async (offer:{capabilities:string[]})=>{capabilities=offer.capabilities;return confirmed} } as unknown as MarketWriter
  render(<QueryClientProvider client={new QueryClient()}><ProviderConsole api={api} writer={writer} /></QueryClientProvider>)
  expect(await screen.findByText('studio-node')).toBeVisible()
  expect(screen.getByRole('region', { name: /provider collateral/i })).toBeVisible()
  expect(screen.getByRole('group', { name: /model identity/i })).toBeVisible()
  expect(screen.getByRole('group', { name: /usage pricing/i })).toBeVisible()
  expect(screen.getByLabelText(/bond amount/i)).toHaveAccessibleDescription(/minimum 5 mon/i)
  expect(screen.getByLabelText(/input tokens/i)).toHaveAccessibleDescription(/one million input tokens/i)
  expect(screen.getByLabelText(/output tokens/i)).toHaveAccessibleDescription(/one million generated tokens/i)
  expect(screen.getByLabelText(/compute time/i)).toHaveAccessibleDescription(/each second/i)
  expect(screen.getByText(/new immutable on-chain price version/i)).toBeVisible()
  expect(screen.getByText(/version 4/i)).toBeVisible()
  expect(screen.getByText(/95 wei earned/i)).toBeVisible()
  await userEvent.type(screen.getByLabelText(/bond MON/i), '2')
  await userEvent.click(screen.getByRole('button', { name: /deposit collateral/i }))
  expect(bond).toBe(2_000_000_000_000_000_000n)
  await userEvent.type(screen.getByLabelText(/offer name/i),'local-qwen');await userEvent.type(screen.getByLabelText(/^model$/i),'qwen');await userEvent.click(screen.getByLabelText(/temporary workspace/i));await userEvent.type(screen.getByLabelText(/input \/ 1M/i),'10');await userEvent.type(screen.getByLabelText(/output \/ 1M/i),'20');await userEvent.type(screen.getByLabelText(/compute \/ second/i),'30');await userEvent.click(screen.getByRole('button',{name:/publish next offer/i}))
  expect(capabilities).toEqual(['text','stream','workspace'])
})
