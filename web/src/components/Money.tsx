import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'
import { formatMON, weiToUSD } from '../lib/amount'
import { ReferencePriceAPI } from '../lib/api'

export function Money({ wei, technical = false }: { wei: bigint | string; technical?: boolean }) {
  const api = useMemo(() => new ReferencePriceAPI(), [])
  const quote = useQuery({ queryKey: ['reference-price'], queryFn: () => api.price(), staleTime: 60_000, retry: false })
  return <span className="money-value">
    <span>{formatMON(wei)}</span>
    {quote.data && <small>≈ {weiToUSD(wei, quote.data.usd_per_mon)} · reference</small>}
    {technical && <details><summary>Raw amount</summary><code>{String(wei)} wei</code></details>}
  </span>
}
