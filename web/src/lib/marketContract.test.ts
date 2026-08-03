import { expect, it } from 'vitest'
import { assertSuccessfulReceipt } from './marketContract'

it('rejects reverted Monad receipts before version handoff', () => {
  expect(() => assertSuccessfulReceipt({ status: 'reverted' })).toThrow(/reverted/i)
  expect(() => assertSuccessfulReceipt({ status: 'success' })).not.toThrow()
})
