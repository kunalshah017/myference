import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'
import { ChatPlayground } from './ChatPlayground'
import type { MarketplaceAPI } from '../../lib/api'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

afterEach(() => vi.restoreAllMocks())

it('sends a real OpenAI-compatible chat request with the supplied key', async () => {
  const stream = [
    'data: {"choices":[{"delta":{"content":"Provider "}}]}',
    'data: {"choices":[{"delta":{"content":"response"}}]}',
    'data: [DONE]',
    '',
  ].join('\n\n')
  const request = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(stream, { status: 200, headers: { 'content-type': 'text/event-stream' } }))
  const user = userEvent.setup()
  const marketplace = { models: async () => [{ model: 'qwen', available_providers: 1, total_capacity: 1, minimum_input_per_million_wei: '1', minimum_output_per_million_wei: '1', minimum_compute_per_second_wei: '1', stale: false }, { model: 'offline', available_providers: 0, total_capacity: 0, minimum_input_per_million_wei: '1', minimum_output_per_million_wei: '1', minimum_compute_per_second_wei: '1', stale: true }] } as MarketplaceAPI
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
  await user.type(screen.getByLabelText(/maximum spend/i), '0.000000000001')
  await user.type(screen.getByLabelText(/message/i), 'Hello provider')
  await user.click(screen.getByRole('button', { name: /send request/i }))

  expect(await screen.findByText('Provider response')).toBeVisible()
  const [, init] = request.mock.calls[0]
  expect(init?.headers).toMatchObject({ authorization: 'Bearer mf_test_key', 'X-Myference-Max-Spend': '1000000' })
  expect(JSON.parse(String(init?.body))).toEqual({ model: 'qwen', stream: true, messages: [{ role: 'user', content: 'Hello provider' }] })
})
