// @vitest-environment jsdom
import { createServer, type Server } from 'node:http'
import type { Address, EIP1193Provider } from 'viem'
import { act, cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from 'vitest'
import { AuthAPI } from '../../lib/api'
import { ApiKeys } from './ApiKeys'
import { ConnectWallet } from './ConnectWallet'
import { DeviceApproval } from './DeviceApproval'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

const address = '0x1111111111111111111111111111111111111111' as Address
let server: Server
let baseURL = ''
let verifyBody: Record<string, string> = {}
let approvedCode = ''
let authorizedSigner = ''
let revokedKey = ''
let createdEndpoints: string[] = []
let loggedOut = false

afterEach(cleanup)

beforeAll(async () => {
  server = createServer((request, response) => {
    response.setHeader('content-type', 'application/json')
    const send = (status: number, value?: unknown) => {
      response.statusCode = status
      response.end(value === undefined ? undefined : JSON.stringify(value))
    }
    let raw = ''
    request.on('data', (chunk) => { raw += chunk })
    request.on('end', () => {
      const body = raw ? JSON.parse(raw) : {}
      if (request.url === '/auth/wallet/challenge') {
        send(201, { id: 'challenge-1', nonce: 'nonce-1', message: 'Myference account login\nChain ID: 10143\nNonce: nonce-1', expires_at: '2026-08-02T18:00:00Z' })
      } else if (request.url === '/auth/wallet/verify') {
        verifyBody = body
        send(200, { account_id: 'acct-1', wallet_address: address, expires_at: '2026-08-02T20:00:00Z' })
      } else if (request.url === '/auth/device/inspect') {
        send(200, { machine_name: 'studio-mac', signer_address: '0x0000000000000000000000000000000000001234', expires_at: '2026-08-02T18:00:00Z' })
      } else if (request.url === '/auth/device/approve') {
        approvedCode = body.user_code
        send(204)
      } else if (request.url === '/auth/api-keys' && request.method === 'GET') {
        send(200, [{ id: 'key-existing', scope: { models: ['qwen'], endpoints: ['/v1/chat/completions'], max_spend_wei: 500 }, created_at: '2026-08-02T17:00:00Z' }])
      } else if (request.url === '/auth/api-keys' && request.method === 'POST') {
        createdEndpoints = body.endpoints
        send(201, { id: 'key-new', token: 'key-new.one-time-secret', scope: body })
      } else if (request.url?.startsWith('/auth/api-keys/') && request.method === 'DELETE') {
        revokedKey = request.url.split('/').at(-1) ?? ''
        send(204)
      } else if (request.url === '/auth/session' && request.method === 'DELETE') {
        loggedOut = true
        send(204)
      } else if (request.url === '/auth/session' && request.method === 'GET') {
        send(204)
      } else {
        send(404, { error: 'not found' })
      }
    })
  })
  await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve))
  const info = server.address()
  if (!info || typeof info === 'string') throw new Error('test server did not bind')
  baseURL = `http://127.0.0.1:${info.port}`
})

afterAll(async () => {
  await new Promise<void>((resolve, reject) => server.close((error) => error ? reject(error) : resolve()))
})

function wallet(chainId = '0x279f'): EIP1193Provider {
  return {
    request: async ({ method }: { method: string }) => {
      if (method === 'eth_chainId') return chainId
      if (method === 'eth_requestAccounts') return [address]
      if (method === 'personal_sign') return `0x${'11'.repeat(65)}`
      throw new Error(`unexpected wallet method ${method}`)
    },
  } as unknown as EIP1193Provider
}

describe('account authentication', () => {
  it('treats a signed-out session probe as empty state', async () => {
    await expect(new AuthAPI(baseURL).session()).resolves.toBeUndefined()
  })

  it('refuses an unsupported chain before requesting a challenge', async () => {
    render(<ConnectWallet api={new AuthAPI(baseURL)} provider={wallet('0x1')} />)
    await userEvent.click(screen.getByRole('button', { name: /connect wallet/i }))
    expect(await screen.findByRole('alert')).toHaveTextContent(/switch to monad testnet/i)
  })

  it('signs the nonce-bound challenge and establishes the server session', async () => {
    render(<ConnectWallet api={new AuthAPI(baseURL)} provider={wallet()} />)
    await userEvent.click(screen.getByRole('button', { name: /connect wallet/i }))
    expect(await screen.findByText(/0x1111…1111/i)).toBeVisible()
    expect(verifyBody).toEqual({ challenge_id: 'challenge-1', signature: `0x${'11'.repeat(65)}` })
  })

  it('disconnects a restored wallet session from the server', async () => {
    loggedOut = false
    const onDisconnected = vi.fn()
    render(<ConnectWallet api={new AuthAPI(baseURL)} provider={wallet()} session={{ account_id: 'acct-1', wallet_address: address, expires_at: '2026-08-02T20:00:00Z' }} onDisconnected={onDisconnected} />)
    expect(screen.getByText(/0x1111…1111/i)).toBeVisible()
    await userEvent.click(screen.getByRole('button', { name: /disconnect wallet/i }))
    expect(loggedOut).toBe(true)
    expect(onDisconnected).toHaveBeenCalledOnce()
    expect(screen.getByRole('button', { name: /connect wallet/i })).toBeVisible()
  })

  it('shows the exact pending machine before approval', async () => {
    render(<DeviceApproval api={new AuthAPI(baseURL)} authorizeSigner={async (signer) => { authorizedSigner = signer }} />)
    await userEvent.type(screen.getByLabelText(/device code/i), 'ABCD1234')
    await userEvent.click(screen.getByRole('button', { name: /review machine/i }))
    expect(await screen.findByText('studio-mac')).toBeVisible()
    await userEvent.click(screen.getByRole('button', { name: /approve studio-mac/i }))
    expect(await screen.findByText(/machine approved/i)).toBeVisible()
    expect(approvedCode).toBe('ABCD1234')
    expect(authorizedSigner).toBe('0x0000000000000000000000000000000000001234')
  })

  it('reveals a new API key once, displays scopes, and revokes it', async () => {
    render(<QueryClientProvider client={new QueryClient()}><ApiKeys api={new AuthAPI(baseURL)} /></QueryClientProvider>)
    expect(await screen.findByText('qwen')).toBeVisible()
    await userEvent.type(screen.getByLabelText(/model/i), 'qwen2.5:0.5b')
    await userEvent.type(screen.getByLabelText(/maximum spend/i), '0.000000000000001')
    await userEvent.click(screen.getByRole('button', { name: /create api key/i }))
    expect(await screen.findByText('key-new.one-time-secret')).toBeVisible()
    expect(createdEndpoints).toEqual(['/v1/chat/completions', '/v1/messages'])
    await userEvent.click(screen.getByRole('button', { name: /i saved this key/i }))
    expect(screen.queryByText('key-new.one-time-secret')).not.toBeInTheDocument()
    await act(async () => { await userEvent.click(screen.getByRole('button', { name: /revoke key-existing/i })) })
    expect(revokedKey).toBe('key-existing')
  })
})
