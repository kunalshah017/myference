import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { MarketplaceAPI } from '../../lib/api'
import { Money } from '../../components/Money'

export function ModelDetail({ api, model }: { api: MarketplaceAPI; model: string }) {
  const [pinned, setPinned] = useState('')
  const detail = useQuery({ queryKey: ['model', model], queryFn: () => api.model(model) })
  if (detail.isPending) return <p role="status">Loading live offers…</p>
  if (detail.isError) return <p role="alert">Live offers could not be loaded.</p>
  return <div className="offer-ledger">{pinned && <p className="pin-proof">Pinned to {pinned}. Requests using this pin will not fall back to another machine.</p>}<div className="offer-ledger-head"><span>Provider</span><span>Usage price</span><span>Capacity</span><span>Route</span></div>{detail.data.offers.map((offer) => <div className="offer-row" key={`${offer.machine_id}:${offer.offer_id}:${offer.price_version}`} data-stale={offer.stale}><div><strong>{offer.machine_id}</strong><span>Price version {offer.price_version}</span><span title={offer.evidence_digest}>{offer.evidence_kind === 'ollama_digest' ? 'Ollama digest pinned' : offer.evidence_kind === 'runtime_image' ? 'Runtime image pinned' : 'Upstream model reported'}</span><span>{offer.metering_mode === 'compute_only' ? 'Compute-only metering' : 'Tokens + compute metered'}</span></div><div className="price-stack"><span><Money wei={offer.input_per_million_wei}/> / 1M input</span><span><Money wei={offer.output_per_million_wei}/> / 1M output</span><span><Money wei={offer.compute_per_second_wei}/> / compute second</span></div><div><strong>{offer.capacity}</strong><span>{offer.available ? 'available' : offer.stale ? 'stale' : 'offline'}</span></div><button type="button" disabled={!offer.available} onClick={() => setPinned(offer.machine_id)}>Use provider {offer.machine_id}</button></div>)}</div>
}
