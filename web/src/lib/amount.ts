import { parseUnits } from 'viem'

export function parseMON(value: string): bigint {
  if (!/^(?:0|[1-9]\d*)(?:\.\d+)?$/.test(value)) throw new Error('Enter a decimal MON amount.')
  const decimals = value.split('.')[1]?.length ?? 0
  if (decimals > 18) throw new Error('MON supports at most 18 decimal places.')
  const amount = parseUnits(value, 18)
  if (amount <= 0n) throw new Error('MON amount must be greater than zero.')
  return amount
}
