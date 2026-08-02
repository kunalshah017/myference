import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import { DashboardOverview } from './DashboardOverview'
import type { AccountOperations, OperationsAPI } from '../lib/api'

it('shows real customer and provider account values from operations', async () => {
  const data: AccountOperations = { chain_id: 10143, contract_address: '0x1111111111111111111111111111111111111111', explorer_url: 'https://example.test', confirmations: 1, wallet_address: '0x2222222222222222222222222222222222222222', customer_balance_wei: '120', provider_bond_wei: '80', claimable_wei: '9', provider_earnings_wei: '33', bond_exit_available_at: 0, sessions: [], machines: [], offers: [] }
  const api = { operations: async () => data } as OperationsAPI
  render(<QueryClientProvider client={new QueryClient()}><DashboardOverview api={api} /></QueryClientProvider>)
  expect(await screen.findByText('120 wei')).toBeVisible()
  expect(screen.getByText('33 wei')).toBeVisible()
  expect(screen.getByText(/historical slashing data is not exposed/i)).toBeVisible()
})
