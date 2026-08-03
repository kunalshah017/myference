import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'
import { ChatPlayground } from './ChatPlayground'
import type { MarketplaceAPI } from '../../lib/api'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

afterEach(() => { cleanup(); vi.restoreAllMocks() })

it('sends a real OpenAI-compatible chat request with the supplied key', async () => {
  const stream = [
    'data: {"choices":[{"delta":{"content":"Provider "}}]}',
    'data: {"choices":[{"delta":{"content":"response"}}]}',
    'data: [DONE]',
    '',
  ].join('\n\n')
  const request = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(stream, { status: 200, headers: { 'content-type': 'text/event-stream' } }))
  const user = userEvent.setup()
  const marketplace = {
    models: async () => [{ model: 'qwen', available_providers: 1, total_capacity: 1, minimum_input_per_million_wei: '1', minimum_output_per_million_wei: '1', minimum_compute_per_second_wei: '1', stale: false }, { model: 'offline', available_providers: 0, total_capacity: 0, minimum_input_per_million_wei: '1', minimum_output_per_million_wei: '1', minimum_compute_per_second_wei: '1', stale: true }],
    model: async () => ({ model: 'qwen', offers: [{ model: 'qwen', machine_id: 'machine', provider_address: '0x1111111111111111111111111111111111111111', offer_id: 'offer', capabilities: ['text', 'stream'], price_version: 1, input_per_million_wei: '1000000', output_per_million_wei: '2000000', compute_per_second_wei: '3', capacity: 1, latency_milliseconds: 1, success_basis_points: 9900, reputation: 1, evidence_kind: 'upstream_model', evidence_digest: 'qwen', metering_mode: 'tokens_and_compute', available: true, stale: false, updated_at: new Date().toISOString() }] }),
  } as unknown as MarketplaceAPI
  render(<QueryClientProvider client={new QueryClient()}><ChatPlayground marketplace={marketplace} /></QueryClientProvider>)
  const model = await screen.findByRole('combobox', { name: /model/i })
  await screen.findByRole('option', { name: 'qwen' })
  expect(model).toHaveDisplayValue('qwen')
  expect(screen.queryByRole('option', { name: 'offline' })).not.toBeInTheDocument()
  const key = screen.getByLabelText(/api key/i)
  expect(key).toHaveAttribute('type', 'text')
  expect(key).toHaveAttribute('autocomplete', 'off')
  expect(key).toHaveAttribute('data-1p-ignore', 'true')
  await user.type(key, 'mf_test_key')
  expect(screen.getByLabelText(/maximum spend/i)).toHaveValue('1')
  expect(screen.getByLabelText(/maximum output tokens/i)).toHaveValue(256)
  await user.type(screen.getByLabelText(/message/i), 'Hello provider')
  expect(screen.getByText('Estimated reservation: 0.000000000000001212 MON. Actual measured usage may cost less.')).toBeVisible()
  await user.click(screen.getByRole('button', { name: /send request/i }))

  expect(await screen.findByText('Provider response')).toBeVisible()
  const [, init] = request.mock.calls[0]
  expect(init?.headers).toMatchObject({ authorization: 'Bearer mf_test_key', 'X-Myference-Max-Spend': '1000000000000000000' })
  expect(JSON.parse(String(init?.body))).toEqual({ model: 'qwen', stream: true, messages: [{ role: 'user', content: 'Hello provider' }], max_completion_tokens: 256 })
})

it('blocks a request maximum below the cheapest complete live offer estimate', async () => {
  const request = vi.spyOn(globalThis, 'fetch')
  const user = userEvent.setup()
  const marketplace = {
    models: async () => [{ model: 'gpt-5.6-luna', available_providers: 1, total_capacity: 1, minimum_input_per_million_wei: '1', minimum_output_per_million_wei: '1', minimum_compute_per_second_wei: '1', stale: false }],
    model: async () => ({ model: 'gpt-5.6-luna', offers: [
      { model: 'gpt-5.6-luna', machine_id: 'free', provider_address: '0x1111111111111111111111111111111111111111', offer_id: 'free', capabilities: ['text', 'stream'], price_version: 1, input_per_million_wei: '0', output_per_million_wei: '0', compute_per_second_wei: '0', capacity: 1, latency_milliseconds: 1, success_basis_points: 9900, reputation: 1, evidence_kind: 'upstream_model', evidence_digest: 'gpt-5.6-luna', metering_mode: 'tokens_and_compute', available: true, stale: false, updated_at: new Date().toISOString() },
      { model: 'gpt-5.6-luna', machine_id: 'machine', provider_address: '0x1111111111111111111111111111111111111111', offer_id: 'luna', capabilities: ['text', 'stream'], price_version: 1, input_per_million_wei: '58553408759882717523', output_per_million_wei: '351320452559296305134', compute_per_second_wei: '813241788332', capacity: 1, latency_milliseconds: 1, success_basis_points: 9900, reputation: 1, evidence_kind: 'upstream_model', evidence_digest: 'gpt-5.6-luna', metering_mode: 'tokens_and_compute', available: true, stale: false, updated_at: new Date().toISOString() },
    ] }),
  } as unknown as MarketplaceAPI
  render(<QueryClientProvider client={new QueryClient()}><ChatPlayground marketplace={marketplace} /></QueryClientProvider>)
  await screen.findByRole('option', { name: 'gpt-5.6-luna' })
  await user.type(screen.getByLabelText(/api key/i), 'mf_test_key')
  await user.clear(screen.getByLabelText(/maximum spend/i))
  await user.type(screen.getByLabelText(/maximum spend/i), '0.01')
  await user.type(screen.getByLabelText(/message/i), 'Hello provider')
  expect(screen.getByText('Estimated reservation: 0.109943783848139819 MON. Actual measured usage may cost less.')).toBeVisible()
  await user.click(screen.getByRole('button', { name: /send request/i }))
  expect(await screen.findByRole('alert')).toHaveTextContent('Maximum spend must be at least 0.109943783848139819 MON')
  expect(request).not.toHaveBeenCalled()
})
