import { describe, expect, it } from 'vitest'
import type { AccountAnalytics, AccountOperations, APIKey, MarketModel } from '../../lib/api'
import { activeSpendingSession, deriveConsumerProgress, deriveProviderProgress, rankLiveModels, recommendedStarterSpend } from './onboarding'

const models: MarketModel[] = [
  { model: 'expensive', available_providers: 2, total_capacity: 2, minimum_input_per_million_wei: '1000000', minimum_output_per_million_wei: '2000000', minimum_compute_per_second_wei: '10', stale: false },
  { model: 'cheap', available_providers: 1, total_capacity: 1, minimum_input_per_million_wei: '1000', minimum_output_per_million_wei: '2000', minimum_compute_per_second_wei: '1', stale: false },
  { model: 'offline', available_providers: 0, total_capacity: 0, minimum_input_per_million_wei: '1', minimum_output_per_million_wei: '1', minimum_compute_per_second_wei: '1', stale: false },
  { model: 'stale', available_providers: 4, total_capacity: 4, minimum_input_per_million_wei: '1', minimum_output_per_million_wei: '1', minimum_compute_per_second_wei: '1', stale: true },
]

const operations: AccountOperations = {
  chain_id: 10143,
  contract_address: '0x4444444444444444444444444444444444444444',
  explorer_url: 'https://testnet.monadexplorer.com',
  confirmations: 2,
  wallet_address: '0x1111111111111111111111111111111111111111',
  customer_balance_wei: '0',
  provider_bond_wei: '0',
  claimable_wei: '0',
  provider_earnings_wei: '0',
  bond_exit_available_at: 0,
  sessions: [],
  machines: [],
  offers: [],
}

const analytics = { customer: { settled_requests: 0 } } as AccountAnalytics

describe('onboarding model recommendations', () => {
  it('ranks only live models by their published minimum usage rates', () => {
    expect(rankLiveModels(models).map((model) => model.model)).toEqual(['cheap', 'expensive'])
  })

  it('calculates a buffered starter maximum using bigint arithmetic', () => {
    expect(recommendedStarterSpend(models[1])).toBe(149n)
  })
})

describe('consumer progress', () => {
  it('ignores finalized, expired, and fully spent sessions', () => {
    const now = 1_800_000_000
    expect(activeSpendingSession([
      { session_id: '0x01', allowance_wei: '100', spent_wei: '0', expires_at: now + 10, close_available_at: 0, finalized: true },
      { session_id: '0x02', allowance_wei: '100', spent_wei: '0', expires_at: now - 1, close_available_at: 0, finalized: false },
      { session_id: '0x03', allowance_wei: '100', spent_wei: '100', expires_at: now + 10, close_available_at: 0, finalized: false },
    ], now)).toBeUndefined()
  })

  it('derives each completion from real account records', () => {
    const progress = deriveConsumerProgress({
      connected: true,
      selectedModel: models[1],
      operations: { ...operations, customer_balance_wei: '100', sessions: [{ session_id: '0x04', allowance_wei: '100', spent_wei: '1', expires_at: 1_900_000_000, close_available_at: 0, finalized: false }] },
      apiKeys: [{ id: 'key-1', scope: { models: ['cheap'], endpoints: ['/v1/chat/completions', '/v1/messages'], max_spend_wei: '10000' } }],
      analytics: { ...analytics, customer: { ...analytics.customer, settled_requests: 1 } },
      now: 1_800_000_000,
    })
    expect(progress.steps.map((step) => step.complete)).toEqual([true, true, true, true, true, true])
    expect(progress.complete).toBe(true)
  })
})

describe('provider progress', () => {
  it('requires a healthy enabled backend linked to a published offer', () => {
    const linked: AccountOperations = {
      ...operations,
      provider_bond_wei: '1',
      offers: [{ offer_id: '0x2ead6e226846be57400e3914d7936aaa5ab80640af1b3c1b12b56277434c95f2', version: 1, model_hash: '0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb', capability_hash: '0xcccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc', input_per_million_wei: '1', output_per_million_wei: '1', compute_per_second_wei: '1' }],
      machines: [{ id: 'machine-1', name: 'studio', revoked: false, backends: [{ id: 'discovered-qwen', offer_hashes: ['0x2ead6e226846be57400e3914d7936aaa5ab80640af1b3c1b12b56277434c95f2'], kind: 'ollama', model: 'qwen', enabled: true, healthy: true, capacity: 1 }] }],
    }
    const progress = deriveProviderProgress({ connected: true, operations: linked })
    expect(progress.steps.map((step) => step.complete)).toEqual([true, true, true, true])
    expect(progress.complete).toBe(true)
    expect(deriveProviderProgress({ connected: true, operations: { ...linked, machines: [{ ...linked.machines[0], backends: [{ ...linked.machines[0].backends[0], healthy: false }] }] } }).complete).toBe(false)
  })
})

it('only completes API access for a key that covers the selected route', () => {
  const wrongKeys: APIKey[] = [{ id: 'wrong', scope: { models: ['other'], endpoints: ['/v1/chat/completions'], max_spend_wei: '1' } }]
  const progress = deriveConsumerProgress({ connected: true, selectedModel: models[1], operations, apiKeys: wrongKeys, analytics })
  expect(progress.steps.find((step) => step.id === 'key')?.complete).toBe(false)
})
