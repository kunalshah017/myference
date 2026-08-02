import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, expect, it } from 'vitest'
import { ProviderAnalytics } from './ProviderAnalytics'
import { UsageAnalytics } from './UsageAnalytics'
import type { AccountAnalytics, AnalyticsAPI } from '../../lib/api'

const data: AccountAnalytics = {
  customer: { settled_requests: 2, input_tokens: 120, output_tokens: 40, compute_milliseconds: 900, provider_charges_wei: '18', protocol_fees_wei: '2', total_spent_wei: '20', gross_revenue_wei: '0', total_slashed_wei: '0' },
  provider: { settled_requests: 3, input_tokens: 300, output_tokens: 80, compute_milliseconds: 1800, provider_charges_wei: '0', protocol_fees_wei: '0', total_spent_wei: '0', gross_revenue_wei: '57', total_slashed_wei: '5' },
  daily: [{ date: '2026-08-03', customer_requests: 2, customer_spent_wei: '20', provider_requests: 3, provider_revenue_wei: '57' }],
  usage: [{ request_id: 'request-1', model: 'qwen', input_tokens: 120, output_tokens: 40, compute_milliseconds: 900, provider_amount_wei: '18', fee_amount_wei: '2', total_charge_wei: '20', transaction_hash: '0xsettled', completed_at: '2026-08-03T00:00:00Z' }],
  settlements: [{ request_id: 'request-2', model: 'qwen', input_tokens: 300, output_tokens: 80, compute_milliseconds: 1800, revenue_wei: '57', transaction_hash: '0xrevenue', completed_at: '2026-08-03T00:00:00Z' }],
  slashes: [{ request_id: 'request-3', amount_wei: '5', block_number: 42, transaction_hash: '0xslash', indexed_at: '2026-08-03T00:00:00Z' }],
}
const api = { analytics: async () => data } as AnalyticsAPI
const renderPanel = (panel: React.ReactNode) => render(<QueryClientProvider client={new QueryClient()}>{panel}</QueryClientProvider>)
afterEach(cleanup)

it('renders confirmed customer token usage and cost', async () => {
  renderPanel(<UsageAnalytics api={api} />)
  expect(await screen.findByText('120')).toBeVisible()
  expect(screen.getAllByText('20 wei')[0]).toBeVisible()
  expect(screen.getByText('request-1')).toBeVisible()
})

it('renders provider revenue and slash history', async () => {
  renderPanel(<ProviderAnalytics api={api} />)
  expect((await screen.findAllByText('57 wei'))[0]).toBeVisible()
  expect(screen.getAllByText('5 wei')[0]).toBeVisible()
  expect(screen.getByText('0xslash')).toBeVisible()
})
