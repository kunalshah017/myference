import { useEffect, useState, type FormEvent } from 'react'
import { useQuery } from '@tanstack/react-query'
import { AuthAPI, MarketplaceAPI, type APIKey } from '../../lib/api'
import { parseMON } from '../../lib/amount'
import { Money } from '../../components/Money'

export function ApiKeys({ api = new AuthAPI(), marketplaceApi = new MarketplaceAPI() }: { api?: AuthAPI; marketplaceApi?: MarketplaceAPI }) {
  const [keys, setKeys] = useState<APIKey[]>([])
  const [revealed, setRevealed] = useState('')
  const [restricted, setRestricted] = useState(false)
  const [selectedModels, setSelectedModels] = useState<string[]>([])
  const [maximum, setMaximum] = useState('')
  const [error, setError] = useState('')
  const models = useQuery({ queryKey: ['models', 'api-key-scope'], queryFn: () => marketplaceApi.models(), enabled: restricted })
  useEffect(() => { void api.listAPIKeys().then(setKeys).catch((reason) => setError(reason instanceof Error ? reason.message : 'API keys could not be loaded.')) }, [api])
  const create = async (event: FormEvent) => {
    event.preventDefault(); setError('')
    if (restricted && selectedModels.length === 0) { setError('Select at least one model or turn off model restriction.'); return }
    try {
      const key = await api.createAPIKey({ models: restricted ? selectedModels : [], endpoints: ['/v1/chat/completions', '/v1/messages'], max_spend_wei: parseMON(maximum).toString() })
      setRevealed(key.token ?? '')
      setKeys((current) => [{ ...key, token: undefined }, ...current])
    } catch (reason) { setError(reason instanceof Error ? reason.message : 'API key creation failed.') }
  }
  const revoke = async (id: string) => {
    try { await api.revokeAPIKey(id); setKeys((current) => current.filter((key) => key.id !== id)) } catch (reason) { setError(reason instanceof Error ? reason.message : 'API key revocation failed.') }
  }
  const toggleModel = (model: string) => setSelectedModels((current) => current.includes(model) ? current.filter((item) => item !== model) : [...current, model])
  const liveModels = models.data?.filter((item) => !item.stale && item.available_providers > 0 && item.total_capacity > 0) ?? []
  return <section className="auth-section" aria-labelledby="api-keys-title"><p className="eyebrow">Project access</p><h2 id="api-keys-title">API keys</h2><p className="dashboard-intro">Keys work with every model by default. Add a model restriction only when a project needs a smaller allowlist.</p>{revealed && <div className="secret-proof" role="status"><span>Copy now — this key is shown once</span><code>{revealed}</code><button type="button" onClick={() => setRevealed('')}>I saved this key</button></div>}<form className="key-form" onSubmit={create}><div className="key-field"><label htmlFor="key-maximum">Maximum spend (MON)</label><input id="key-maximum" inputMode="decimal" value={maximum} onChange={(event) => setMaximum(event.target.value)} required /><small>Limits the most this key may authorize for one request.</small></div><label className="key-scope-toggle"><input type="checkbox" checked={restricted} onChange={(event) => { setRestricted(event.target.checked); if (!event.target.checked) setSelectedModels([]) }} /> <span><strong>Restrict to selected models</strong><small>Optional. Leave off to access all current and future models.</small></span></label>{restricted && <fieldset className="key-model-options"><legend>Allowed models</legend>{models.isPending ? <p role="status">Loading live models…</p> : models.isError ? <p role="alert">Models could not be loaded. Try again or use all models.</p> : liveModels.length === 0 ? <p role="status">No live models are currently available.</p> : liveModels.map((item) => <label key={item.model}><input type="checkbox" checked={selectedModels.includes(item.model)} onChange={() => toggleModel(item.model)} /> <span>{item.model}</span></label>)}</fieldset>}<button type="submit" disabled={restricted && (models.isPending || models.isError || selectedModels.length === 0)}>Create API key</button></form>{error && <p role="alert" className="inline-error">{error}</p>}<ul className="key-list">{keys.map((key) => <li key={key.id}><div><code>{key.id}</code><span>{key.scope.models.length === 0 ? 'All models' : key.scope.models.join(', ')}</span><span>{key.scope.endpoints.join(', ')}</span><span><Money wei={key.scope.max_spend_wei}/> max</span></div><button type="button" onClick={() => void revoke(key.id)}>Revoke {key.id}</button></li>)}</ul></section>
}
