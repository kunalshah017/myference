import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'
import { ChatPlayground } from './ChatPlayground'

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
  render(<ChatPlayground />)
  await user.type(screen.getByLabelText(/model/i), 'qwen')
  await user.type(screen.getByLabelText(/api key/i), 'mf_test_key')
  await user.type(screen.getByLabelText(/maximum spend/i), '1000000')
  await user.type(screen.getByLabelText(/message/i), 'Hello provider')
  await user.click(screen.getByRole('button', { name: /send request/i }))

  expect(await screen.findByText('Provider response')).toBeVisible()
  const [, init] = request.mock.calls[0]
  expect(init?.headers).toMatchObject({ authorization: 'Bearer mf_test_key', 'X-Myference-Max-Spend': '1000000' })
  expect(JSON.parse(String(init?.body))).toEqual({ model: 'qwen', stream: true, messages: [{ role: 'user', content: 'Hello provider' }] })
})
