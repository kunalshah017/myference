import { formatUnits, parseUnits } from 'viem'

export function parseMON(value: string): bigint {
  if (!/^(?:0|[1-9]\d*)(?:\.\d+)?$/.test(value)) throw new Error('Enter a decimal MON amount.')
  const decimals = value.split('.')[1]?.length ?? 0
  if (decimals > 18) throw new Error('MON supports at most 18 decimal places.')
  const amount = parseUnits(value, 18)
  if (amount <= 0n) throw new Error('MON amount must be greater than zero.')
  return amount
}

export function formatMON(value: bigint | string): string {
  const decimal = formatUnits(BigInt(value), 18)
  return `${decimal.includes('.') ? decimal.replace(/0+$/, '').replace(/\.$/, '') : decimal} MON`
}

function decimalRatio(value: string, label: string): { integer: bigint; scale: bigint } {
  const normalized = value.trim()
  if (!/^(?:0|[1-9]\d*)(?:\.\d+)?$/.test(normalized)) throw new Error(`Enter a decimal ${label}.`)
  const [whole, fraction = ''] = normalized.split('.')
  if (fraction.length > 18) throw new Error(`${label} supports at most 18 decimal places.`)
  const integer = BigInt(`${whole}${fraction}`)
  if (integer <= 0n) throw new Error(`${label} must be greater than zero.`)
  return { integer, scale: 10n ** BigInt(fraction.length) }
}

function ceilDivide(numerator: bigint, denominator: bigint): bigint {
  return (numerator + denominator - 1n) / denominator
}

export function usdToWei(usd: string, usdPerMON: string): bigint {
  const amount = decimalRatio(usd, 'USD amount')
  const quote = decimalRatio(usdPerMON, 'MON reference price')
  return ceilDivide(amount.integer * quote.scale * 10n ** 18n, amount.scale * quote.integer)
}

export function computeMinuteUSDToWeiPerSecond(usd: string, usdPerMON: string): bigint {
  return ceilDivide(usdToWei(usd, usdPerMON), 60n)
}

export function weiToUSD(value: bigint | string, usdPerMON: string): string {
  const quote = decimalRatio(usdPerMON, 'MON reference price')
  const micros = BigInt(value) * quote.integer * 1_000_000n / (10n ** 18n * quote.scale)
  return `$${micros / 1_000_000n}.${(micros % 1_000_000n).toString().padStart(6, '0')}`
}

export function isFreshReferencePrice(quote: { updated_at: string } | undefined, now = Date.now()): boolean {
  if (!quote) return false
  const updated = Date.parse(quote.updated_at)
  return Number.isFinite(updated) && updated <= now + 60_000 && now - updated <= 15 * 60_000
}
