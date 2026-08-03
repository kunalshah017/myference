// @vitest-environment jsdom
import { createServer, type Server } from 'node:http'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterAll, afterEach, beforeAll, expect, it } from 'vitest'
import { MarketplaceAPI, type MarketModel } from '../../lib/api'
import { reconcileRequestEvent } from '../../lib/realtime'
import { ModelList } from './ModelList'
import { Activity } from '../activity/Activity'

let server: Server
let baseURL = ''
let inventory: MarketModel[] = []
let activity: unknown[] = []

beforeAll(async () => {
  server = createServer((request, response) => {
    response.setHeader('content-type', 'application/json')
    if (request.url === '/api/models') response.end(JSON.stringify(inventory))
    else if (request.url === '/api/activity') response.end(JSON.stringify(activity))
    else if (request.url?.startsWith('/api/models/')) response.end(JSON.stringify({ model: inventory[0]?.model, offers: [{ machine_id: 'machine-live', provider_address: '0x1111111111111111111111111111111111111111', offer_id: 'offer-live', model: inventory[0]?.model, capabilities: ['chat', 'stream'], price_version: 3, input_per_million_wei: '10', output_per_million_wei: '20', compute_per_second_wei: '30', capacity: 2, latency_milliseconds: 40, success_basis_points: 9900, reputation: 4, evidence_kind: 'ollama_digest', evidence_digest: 'sha256:runtime-digest', metering_mode: 'tokens_and_compute', available: true, stale: false, updated_at: '2026-08-02T17:00:00Z' }] }))
    else { response.statusCode = 404; response.end('{}') }
  })
  await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve))
  const info = server.address()
  if (!info || typeof info === 'string') throw new Error('test server did not bind')
  baseURL = `http://127.0.0.1:${info.port}`
})

afterEach(() => { cleanup(); inventory = []; activity = [] })
afterAll(async () => { await new Promise<void>((resolve, reject) => server.close((error) => error ? reject(error) : resolve())) })

function renderModels() {
  const query = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={query}><ModelList api={new MarketplaceAPI(baseURL)} /></QueryClientProvider>)
}

it('renders only live API inventory with all-inclusive integer prices and provider pinning', async () => {
  inventory = [{ model: 'qwen2.5:0.5b', available_providers: 1, total_capacity: 2, minimum_input_per_million_wei: '10', minimum_output_per_million_wei: '20', minimum_compute_per_second_wei: '30', stale: false }]
  renderModels()
  await userEvent.click(await screen.findByRole('button', { name: /qwen2.5:0.5b/i }))
  expect((await screen.findAllByText('0.00000000000000001 MON')).length).toBeGreaterThan(0)
  expect(await screen.findByText(/price version 3/i)).toBeVisible()
  expect(screen.getByText(/ollama digest pinned/i)).toBeVisible()
  expect(screen.getByText(/tokens \+ compute metered/i)).toBeVisible()
  await userEvent.click(screen.getByRole('button', { name: /use provider machine-live/i }))
  expect(screen.getByText(/pinned to machine-live/i)).toBeVisible()
  expect(screen.queryByText(/mock|sample provider/i)).not.toBeInTheDocument()
})

it('renders honest empty and stale states', async () => {
  renderModels()
  expect(await screen.findByText(/no providers are serving a model right now/i)).toBeVisible()
  cleanup()
  inventory = [{ model: 'stale-model', available_providers: 0, total_capacity: 0, minimum_input_per_million_wei: '1', minimum_output_per_million_wei: '2', minimum_compute_per_second_wei: '3', stale: true }]
  renderModels()
  expect(await screen.findByText(/capacity is stale/i)).toBeVisible()
})

it('detects cursor gaps and never reopens a terminal request', () => {
  const settled = { cursor: 20, state: 'settled', needsRefetch: false }
  expect(reconcileRequestEvent(settled, { cursor: 19, state: 'streaming' })).toEqual(settled)
  expect(reconcileRequestEvent(settled, { cursor: 22, state: 'streaming' })).toEqual({ cursor: 22, state: 'settled', needsRefetch: true })
})

it('renders authoritative request activity without inventing transitions', async () => {
  activity = [{ request_id: 'request-live', session_id: 'session-live', account_id: 'acct-live', state: 'streaming', machine_id: 'machine-live', offer_id: 'offer-live', model: 'qwen', price_version: 3, updated_at: '2026-08-02T17:00:00Z' }]
  const query = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(<QueryClientProvider client={query}><Activity api={new MarketplaceAPI(baseURL)} connected subscribe={() => () => undefined} /></QueryClientProvider>)
  expect(await screen.findByText('request-live')).toBeVisible()
  expect(screen.getByText(/streaming on machine-live/i)).toBeVisible()
})
