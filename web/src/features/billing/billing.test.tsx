// @vitest-environment jsdom
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'
import type { AccountOperations, OperationsAPI } from '../../lib/api'
import type { MarketWriter, SubmittedTransaction } from '../../lib/marketContract'
import { parseMON } from '../../lib/amount'
import { Billing } from './Billing'

afterEach(cleanup)

const operations: AccountOperations = {
  chain_id: 10143, contract_address: '0x4444444444444444444444444444444444444444', explorer_url: 'https://testnet.monadexplorer.com', confirmations: 2,
  wallet_address: '0x1111111111111111111111111111111111111111', customer_balance_wei: '1000', provider_bond_wei: '2000', claimable_wei: '3000', provider_earnings_wei: '95', bond_exit_available_at: 0,
  sessions: [], machines: [], offers: [],
}

function api(): OperationsAPI { return { operations: async () => operations } as OperationsAPI }
function wrapper(children: React.ReactNode) { return <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>{children}</QueryClientProvider> }

it('parses MON to wei without floating point or precision loss', () => {
  expect(parseMON('1.000000000000000001')).toBe(1_000_000_000_000_000_001n)
  expect(() => parseMON('0.0000000000000000001')).toThrow(/18 decimal places/i)
  expect(() => parseMON('1e3')).toThrow(/decimal MON amount/i)
})

it('shows deposit as pending until finality then refreshes indexed balances', async () => {
  let confirm!: () => void
  const pending = new Promise<void>((resolve) => { confirm = resolve })
  let deposited = 0n
  const writer = { deposit: async (value: bigint): Promise<SubmittedTransaction> => { deposited = value; return { hash: '0xabc', confirm: () => pending } } } as MarketWriter
  render(wrapper(<Billing api={api()} writer={writer} />))
  expect(await screen.findByText(/0\.000000000000001 MON/i)).toBeVisible()
  await userEvent.type(screen.getByLabelText(/deposit MON/i), '0.5')
  await userEvent.click(screen.getByRole('button', { name: /deposit to escrow/i }))
  expect(await screen.findByText(/0xabc.*pending/i)).toBeVisible()
  expect(deposited).toBe(500_000_000_000_000_000n)
  confirm()
  expect(await screen.findByText(/transaction finalized/i)).toBeVisible()
})

it('surfaces rejected wallet and contract reverts without changing balances', async () => {
  const writer = { deposit: async () => { throw new Error('Wallet request rejected') } } as unknown as MarketWriter
  render(wrapper(<Billing api={api()} writer={writer} />))
  await userEvent.type(await screen.findByLabelText(/deposit MON/i), '1')
  await userEvent.click(screen.getByRole('button', { name: /deposit to escrow/i }))
  expect(await screen.findByRole('alert')).toHaveTextContent(/wallet request rejected/i)
})

it('keeps billing transactions single-flight through wallet, chain, and refresh phases', async () => {
  let releaseWallet!: (transaction: SubmittedTransaction) => void
  let releaseChain!: () => void
  let releaseRefresh!: () => void
  const wallet = new Promise<SubmittedTransaction>((resolve) => { releaseWallet = resolve })
  const chain = new Promise<void>((resolve) => { releaseChain = resolve })
  const refresh = new Promise<AccountOperations>((resolve) => { releaseRefresh = () => resolve(operations) })
  let operationCalls = 0
  const operationsAPI = { operations: async () => operationCalls++ === 0 ? operations : refresh } as OperationsAPI
  const deposit = vi.fn(() => wallet)
  render(wrapper(<Billing api={operationsAPI} writer={{ deposit } as unknown as MarketWriter} />))

  await userEvent.type(await screen.findByLabelText(/deposit MON/i), '1')
  const button = screen.getByRole('button', { name: /deposit to escrow/i })
  await userEvent.click(button)
  await userEvent.click(button)
  expect(deposit).toHaveBeenCalledOnce()
  expect(screen.getByRole('button', { name: /confirm deposit in wallet/i })).toBeDisabled()

  releaseWallet({ hash: '0xabc', confirm: () => chain })
  expect(await screen.findByRole('button', { name: /confirming on Monad/i })).toBeDisabled()
  releaseChain()
  expect(await screen.findByRole('button', { name: /refreshing account/i })).toBeDisabled()
  releaseRefresh()
  await waitFor(() => expect(screen.getByRole('button', { name: /deposit to escrow/i })).toBeEnabled())
})
