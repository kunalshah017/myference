import { render, screen } from '@testing-library/react'
import { expect, it } from 'vitest'
import LandingPage from './LandingPage'

it('explains the marketplace and links to the operational app', () => {
  render(<LandingPage />)
  expect(screen.getByRole('heading', { name: /unused compute,useful inference/i })).toBeVisible()
  expect(screen.getByRole('link', { name: /launch app/i })).toHaveAttribute('href', '/app')
})
