export type Session = { account_id: string; wallet_address: `0x${string}`; expires_at: string }
export type WalletChallenge = { id: string; nonce: string; message: string; expires_at: string }
export type PendingDevice = { machine_name: string; expires_at: string }
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

export class AuthAPI {
  private readonly baseURL: string

  constructor(baseURL = import.meta.env.VITE_MYFERENCE_API_URL ?? '') { this.baseURL = baseURL }

  challenge(address: `0x${string}`) {
    return this.request<WalletChallenge>('/auth/wallet/challenge', { method: 'POST', body: { address } })
  }

  session() { return this.request<Session>('/auth/session') }

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

export class OperationsAPI {
  private readonly baseURL: string
  constructor(baseURL = import.meta.env.VITE_MYFERENCE_API_URL ?? '') { this.baseURL = baseURL }
  operations() { return requestJSON<AccountOperations>(`${this.baseURL}/api/account/operations`) }
}

async function requestJSON<T>(url: string): Promise<T> {
  const response = await fetch(url, { credentials: 'include' })
  if (!response.ok) throw new Error((await response.text()).trim() || `Request failed (${response.status})`)
  return response.json() as Promise<T>
}
