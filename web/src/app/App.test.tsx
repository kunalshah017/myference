import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import App from './App'

it('shows an honest disconnected marketplace without fake inventory', () => {
  render(<App />)

  expect(
    screen.getByRole('heading', { name: /unused machines, useful inference/i }),
  ).toBeVisible()
  expect(
    screen.getByText(/live marketplace data is not connected/i),
  ).toBeVisible()
  expect(screen.getByRole('button', { name: /connect wallet/i })).toBeEnabled()
  expect(
    screen.queryByText(/mock|demo provider|sample balance/i),
  ).not.toBeInTheDocument()
})
