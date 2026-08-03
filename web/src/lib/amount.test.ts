import { describe, expect, it } from 'vitest'
import { computeMinuteUSDToWeiPerSecond, formatMON, parseMON, usdToWei, weiToUSD } from './amount'

describe('MON money helpers', () => {
  it('formats small contract values as readable MON without losing them to zero', () => {
    expect(formatMON(54_750_000n)).toBe('0.00000000005475 MON')
    expect(formatMON(5_000_000_000_000_000_000n)).toBe('5 MON')
  })

  it('converts decimal USD targets to integer wei without floating point', () => {
    expect(usdToWei('0.10', '0.02')).toBe(parseMON('5'))
    expect(usdToWei('0.01', '3')).toBe(3_333_333_333_333_334n)
    expect(computeMinuteUSDToWeiPerSecond('0.06', '0.02')).toBe(50_000_000_000_000_000n)
  })

  it('formats a secondary USD estimate', () => {
    expect(weiToUSD(parseMON('5'), '0.02090926')).toBe('$0.104546')
  })
})
