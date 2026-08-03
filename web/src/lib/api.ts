export type Session = { account_id: string; wallet_address: `0x${string}`; expires_at: string }
export type ReferencePrice = { symbol: 'MON'; usd_per_mon: string; source: string; updated_at: string }
export type WalletChallenge = { id: string; nonce: string; message: string; expires_at: string }
export type PendingDevice = { machine_name: string; signer_address: `0x${string}`; expires_at: string }
export type APIKeyScope = { models: string[]; endpoints: string[]; max_spend_wei: string }
export type APIKey = { id: string; token?: string; scope: APIKeyScope; created_at?: string }
export type StreamTicket = { ticket: string; expires_at: string }
export type MarketModel = { model: string; available_providers: number; total_capacity: number; minimum_input_per_million_wei: string; minimum_output_per_million_wei: string; minimum_compute_per_second_wei: string; stale: boolean }
export type MarketOffer = { machine_id: string; provider_address: `0x${string}`; offer_id: string; model: string; capabilities: string[]; price_version: number; input_per_million_wei: string; output_per_million_wei: string; compute_per_second_wei: string; capacity: number; latency_milliseconds: number; success_basis_points: number; reputation: number; available: boolean; stale: boolean; updated_at: string }
export type MarketModelDetail = { model: string; offers: MarketOffer[] }
export type ActivityRecord = { request_id: string; session_id: string; account_id: string; state: string; machine_id: string; offer_id: string; model: string; price_version: number; updated_at: string; transaction_hash?: string }
export type OperationSession = { session_id: `0x${string}`; allowance_wei: string; spent_wei: string; expires_at: number; close_available_at: number; finalized: boolean }
export type OperationBackend = { id: string; kind: string; model: string; enabled: boolean; healthy: boolean; capacity: number }
export type OperationMachine = { id: string; name: string; revoked: boolean; backends: OperationBackend[] }
export type OperationOffer = { offer_id: `0x${string}`; version: number; model_hash: `0x${string}`; capability_hash: `0x${string}`; input_per_million_wei: string; output_per_million_wei: string; compute_per_second_wei: string }
export type AccountOperations = { chain_id: number; contract_address: `0x${string}`; explorer_url: string; confirmations: number; wallet_address: `0x${string}`; customer_balance_wei: string; provider_bond_wei: string; claimable_wei: string; provider_earnings_wei: string; bond_exit_available_at: number; sessions: OperationSession[]; machines: OperationMachine[]; offers: OperationOffer[] }
export type ChatMessage = { role: 'user' | 'assistant'; content: string }
export type AnalyticsTotals = { settled_requests: number; input_tokens: number; output_tokens: number; compute_milliseconds: number; provider_charges_wei: string; protocol_fees_wei: string; total_spent_wei: string; gross_revenue_wei: string; total_slashed_wei: string }
export type AnalyticsDay = { date: string; customer_requests: number; customer_spent_wei: string; provider_requests: number; provider_revenue_wei: string }
export type UsageRecord = { request_id: string; model: string; input_tokens: number; output_tokens: number; compute_milliseconds: number; provider_amount_wei: string; fee_amount_wei: string; total_charge_wei: string; transaction_hash: string; completed_at: string }
export type ProviderSettlement = { request_id: string; model: string; input_tokens: number; output_tokens: number; compute_milliseconds: number; revenue_wei: string; transaction_hash: string; completed_at: string }
export type SlashRecord = { request_id: string; amount_wei: string; block_number: number; transaction_hash: string; indexed_at: string }
export type AccountAnalytics = { customer: AnalyticsTotals; provider: AnalyticsTotals; daily: AnalyticsDay[]; usage: UsageRecord[]; settlements: ProviderSettlement[]; slashes: SlashRecord[] }

export class AuthAPI {
  private readonly baseURL: string

  constructor(baseURL = import.meta.env.VITE_MYFERENCE_API_URL ?? '') { this.baseURL = baseURL }

  challenge(address: `0x${string}`) {
    return this.request<WalletChallenge>('/auth/wallet/challenge', { method: 'POST', body: { address } })
  }

  session() { return this.request<Session | undefined>('/auth/session') }

  logout() { return this.request<void>('/auth/session', { method: 'DELETE' }) }

  verify(challengeId: string, signature: `0x${string}`) {
    return this.request<Session>('/auth/wallet/verify', { method: 'POST', body: { challenge_id: challengeId, signature } })
  }

  inspectDevice(userCode: string) {
    return this.request<PendingDevice>('/auth/device/inspect', { method: 'POST', body: { user_code: userCode } })
  }

  approveDevice(userCode: string) {
    return this.request<void>('/auth/device/approve', { method: 'POST', body: { user_code: userCode } })
  }

  listAPIKeys() { return this.request<APIKey[]>('/auth/api-keys') }

  createAPIKey(scope: APIKeyScope) {
    return this.request<APIKey>('/auth/api-keys', { method: 'POST', body: scope })
  }

  revokeAPIKey(id: string) {
    return this.request<void>(`/auth/api-keys/${encodeURIComponent(id)}`, { method: 'DELETE' })
  }

  streamTicket() { return this.request<StreamTicket>('/auth/stream-ticket', { method: 'POST', body: {} }) }

  eventsURL(ticket: string) { return `${this.baseURL}/events?ticket=${encodeURIComponent(ticket)}` }

  private async request<T>(path: string, options: { method?: string; body?: unknown } = {}): Promise<T> {
    const response = await fetch(`${this.baseURL}${path}`, {
      method: options.method ?? 'GET',
      credentials: 'include',
      headers: options.body === undefined ? undefined : { 'content-type': 'application/json' },
      body: options.body === undefined ? undefined : JSON.stringify(options.body),
    })
    if (!response.ok) {
      const message = (await response.text()).trim()
      throw new Error(message || `Request failed (${response.status})`)
    }
    if (response.status === 204) return undefined as T
    return response.json() as Promise<T>
  }
}

export class MarketplaceAPI {
  constructor(privateBaseURL = import.meta.env.VITE_MYFERENCE_API_URL ?? '') { this.baseURL = privateBaseURL }
  private readonly baseURL: string
  models() { return requestJSON<MarketModel[]>(`${this.baseURL}/api/models`) }
  model(name: string) { return requestJSON<MarketModelDetail>(`${this.baseURL}/api/models/${encodeURIComponent(name)}`) }
  activity() { return requestJSON<ActivityRecord[]>(`${this.baseURL}/api/activity`) }
}

export class ReferencePriceAPI {
  private readonly baseURL: string
  constructor(baseURL = import.meta.env.VITE_MYFERENCE_API_URL ?? '') { this.baseURL = baseURL }
  price() { return requestJSON<ReferencePrice>(`${this.baseURL}/api/reference-price`) }
}

export class OperationsAPI {
  private readonly baseURL: string
  constructor(baseURL = import.meta.env.VITE_MYFERENCE_API_URL ?? '') { this.baseURL = baseURL }
  operations() { return requestJSON<AccountOperations>(`${this.baseURL}/api/account/operations`) }
}

export class InferenceAPI {
  private readonly baseURL: string
  constructor(baseURL = import.meta.env.VITE_MYFERENCE_API_URL ?? '') { this.baseURL = baseURL }
  async chat(model: string, apiKey: string, maximumSpend: string, messages: ChatMessage[]) {
    const response = await fetch(`${this.baseURL}/v1/chat/completions`, {
      method: 'POST',
      headers: { authorization: `Bearer ${apiKey}`, 'content-type': 'application/json', 'X-Myference-Max-Spend': maximumSpend },
      body: JSON.stringify({ model, stream: true, messages }),
    })
    if (!response.ok) throw new Error((await response.text()).trim() || `Request failed (${response.status})`)
    let content = ''
    for (const line of (await response.text()).split(/\r?\n/)) {
      if (!line.startsWith('data: ')) continue
      const data = line.slice(6)
      if (data === '[DONE]') break
      const payload = JSON.parse(data) as { choices?: { delta?: { content?: string } }[]; error?: { message?: string } }
      if (payload.error) throw new Error(payload.error.message || 'Provider execution failed.')
      content += payload.choices?.[0]?.delta?.content ?? ''
    }
    if (!content) throw new Error('The provider returned no assistant message.')
    return content
  }
}

export class AnalyticsAPI {
  private readonly baseURL: string
  constructor(baseURL = import.meta.env.VITE_MYFERENCE_API_URL ?? '') { this.baseURL = baseURL }
  analytics() { return requestJSON<AccountAnalytics>(`${this.baseURL}/api/account/analytics`) }
}

export function publicAPIBaseURL() {
  return import.meta.env.VITE_MYFERENCE_API_URL || window.location.origin
}

async function requestJSON<T>(url: string): Promise<T> {
  const response = await fetch(url, { credentials: 'include' })
  if (!response.ok) throw new Error((await response.text()).trim() || `Request failed (${response.status})`)
  return response.json() as Promise<T>
}
