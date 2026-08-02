import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, expect, it, vi } from 'vitest'
import { ChatPlayground } from './ChatPlayground'

afterEach(() => vi.restoreAllMocks())

it('sends a real OpenAI-compatible chat request with the supplied key', async () => {
  const request = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ choices: [{ message: { role: 'assistant', content: 'Provider response' } }] }), { status: 200, headers: { 'content-type': 'application/json' } }))
  const user = userEvent.setup()
  render(<ChatPlayground />)
  await user.type(screen.getByLabelText(/model/i), 'qwen')
  await user.type(screen.getByLabelText(/api key/i), 'mf_test_key')
  await user.type(screen.getByLabelText(/message/i), 'Hello provider')
  await user.click(screen.getByRole('button', { name: /send request/i }))

  expect(await screen.findByText('Provider response')).toBeVisible()
  const [, init] = request.mock.calls[0]
  expect(init?.headers).toMatchObject({ authorization: 'Bearer mf_test_key' })
  expect(JSON.parse(String(init?.body))).toEqual({ model: 'qwen', stream: false, messages: [{ role: 'user', content: 'Hello provider' }] })
})
