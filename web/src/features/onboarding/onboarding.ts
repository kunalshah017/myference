import type { AccountAnalytics, AccountOperations, APIKey, MarketModel, OperationSession } from '../../lib/api'

export type OnboardingRole = 'consumer' | 'provider'
export type ProgressStep = { id: string; label: string; complete: boolean }
export type OnboardingProgress = { steps: ProgressStep[]; complete: boolean; completed: number; next?: ProgressStep }

function ceilDivide(value: bigint, divisor: bigint) {
  return (value + divisor - 1n) / divisor
}

function modelRate(model: MarketModel) {
  return BigInt(model.minimum_input_per_million_wei) + BigInt(model.minimum_output_per_million_wei) + BigInt(model.minimum_compute_per_second_wei)
}

export function rankLiveModels(models: MarketModel[]) {
  return models
    .filter((model) => !model.stale && model.available_providers > 0 && model.total_capacity > 0)
    .sort((left, right) => modelRate(left) < modelRate(right) ? -1 : modelRate(left) > modelRate(right) ? 1 : left.model.localeCompare(right.model))
}

export function recommendedStarterSpend(model: MarketModel) {
  const input = ceilDivide(BigInt(model.minimum_input_per_million_wei) * 2_000n, 1_000_000n)
  const output = ceilDivide(BigInt(model.minimum_output_per_million_wei) * 1_000n, 1_000_000n)
  const compute = BigInt(model.minimum_compute_per_second_wei) * 120n
  return ceilDivide((input + output + compute) * 120n, 100n)
}

export function activeSpendingSession(sessions: OperationSession[], now = Math.floor(Date.now() / 1000)) {
  return sessions.find((session) => !session.finalized && session.expires_at > now && BigInt(session.spent_wei) < BigInt(session.allowance_wei))
}

function progress(steps: ProgressStep[]): OnboardingProgress {
  const completed = steps.filter((step) => step.complete).length
  return { steps, completed, complete: completed === steps.length, next: steps.find((step) => !step.complete) }
}

export function deriveConsumerProgress({ connected, selectedModel, operations, apiKeys, analytics, inferenceSucceeded = false, now }: {
  connected: boolean
  selectedModel?: MarketModel
  operations?: AccountOperations
  apiKeys: APIKey[]
  analytics?: AccountAnalytics
  inferenceSucceeded?: boolean
  now?: number
}) {
  const requiredSpend = selectedModel ? recommendedStarterSpend(selectedModel) : 0n
  const scopedKey = selectedModel && apiKeys.some((key) => key.scope.models.includes(selectedModel.model) && ['/v1/chat/completions', '/v1/messages'].every((endpoint) => key.scope.endpoints.includes(endpoint)) && BigInt(key.scope.max_spend_wei) >= requiredSpend)
  return progress([
    { id: 'wallet', label: 'Connect account', complete: connected },
    { id: 'model', label: 'Choose inference', complete: Boolean(selectedModel) },
    { id: 'deposit', label: 'Fund requests', complete: BigInt(operations?.customer_balance_wei ?? 0) > 0n },
    { id: 'session', label: 'Set a limit', complete: Boolean(activeSpendingSession(operations?.sessions ?? [], now)) },
    { id: 'key', label: 'Create access', complete: Boolean(scopedKey) },
    { id: 'inference', label: 'Run inference', complete: inferenceSucceeded || (analytics?.customer.settled_requests ?? 0) > 0 },
  ])
}

export function deriveProviderProgress({ connected, operations }: { connected: boolean; operations?: AccountOperations }) {
  const machines = operations?.machines.filter((machine) => !machine.revoked) ?? []
  const offerHashes = new Set((operations?.offers ?? []).map((offer) => offer.offer_id.toLowerCase()))
  const live = machines.some((machine) => machine.backends.some((backend) => backend.enabled && backend.healthy && backend.capacity > 0 && backend.offer_hashes.some((offerHash) => offerHashes.has(offerHash.toLowerCase()))))
  return progress([
    { id: 'wallet', label: 'Connect account', complete: connected },
    { id: 'machine', label: 'Connect a machine', complete: machines.some((machine) => machine.backends.length > 0) },
    { id: 'bond', label: 'Bond collateral', complete: BigInt(operations?.provider_bond_wei ?? 0) > 0n },
    { id: 'offer', label: 'Go live', complete: live },
  ])
}
