export type Session = { account_id: string; wallet_address: `0x${string}`; expires_at: string }
export type WalletChallenge = { id: string; nonce: string; message: string; expires_at: string }
export type PendingDevice = { machine_name: string; expires_at: string }
export type APIKeyScope = { models: string[]; endpoints: string[]; max_spend_wei: string }
export type APIKey = { id: string; token?: string; scope: APIKeyScope; created_at?: string }

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
