import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { MarketplaceAPI } from '../../lib/api'
import { ModelDetail } from './ModelDetail'
import { Money } from '../../components/Money'

export function ModelList({ api = new MarketplaceAPI() }: { api?: MarketplaceAPI }) {
  const [selected, setSelected] = useState('')
  const models = useQuery({ queryKey: ['models'], queryFn: () => api.models(), refetchInterval: 15_000 })
  if (models.isPending) return <div className="empty-state" role="status"><span className="empty-glyph" aria-hidden="true">///</span><p>Loading live provider capacity…</p></div>
  if (models.isError) return <div className="empty-state" role="alert"><span className="empty-glyph" aria-hidden="true">///</span><p>Live marketplace data is not connected. The broker did not return inventory.</p></div>
  if (models.data.length === 0) return <div className="empty-state" role="status"><span className="empty-glyph" aria-hidden="true">///</span><p>No providers are serving a model right now.</p></div>
  return <div className="model-index">{models.data.map((model) => <article key={model.model} className="model-line" data-stale={model.stale}><button type="button" className="model-select" onClick={() => setSelected(selected === model.model ? '' : model.model)} aria-expanded={selected === model.model}><strong>{model.model}</strong><span>{model.stale ? 'Capacity is stale' : `${model.available_providers} provider · ${model.total_capacity} slots`}</span><span>from <Money wei={model.minimum_input_per_million_wei}/> / 1M input</span></button>{selected === model.model && <ModelDetail api={api} model={model.model} />}</article>)}</div>
}
