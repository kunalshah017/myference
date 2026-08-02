import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { MarketplaceAPI } from '../../lib/api'

export function ModelDetail({ api, model }: { api: MarketplaceAPI; model: string }) {
  const [pinned, setPinned] = useState('')
  const detail = useQuery({ queryKey: ['model', model], queryFn: () => api.model(model) })
  if (detail.isPending) return <p role="status">Loading live offers…</p>
  if (detail.isError) return <p role="alert">Live offers could not be loaded.</p>
  return <div className="offer-ledger">{pinned && <p className="pin-proof">Pinned to {pinned}. Requests using this pin will not fall back to another machine.</p>}<div className="offer-ledger-head"><span>Provider</span><span>All-inclusive price</span><span>Capacity</span><span>Route</span></div>{detail.data.offers.map((offer) => <div className="offer-row" key={`${offer.machine_id}:${offer.offer_id}:${offer.price_version}`} data-stale={offer.stale}><div><strong>{offer.machine_id}</strong><span>Price version {offer.price_version}</span></div><div className="price-stack"><span>{offer.input_per_million_wei} wei / 1M input</span><span>{offer.output_per_million_wei} wei / 1M output</span><span>{offer.compute_per_second_wei} wei / compute second</span></div><div><strong>{offer.capacity}</strong><span>{offer.available ? 'available' : offer.stale ? 'stale' : 'offline'}</span></div><button type="button" disabled={!offer.available} onClick={() => setPinned(offer.machine_id)}>Use provider {offer.machine_id}</button></div>)}</div>
}
