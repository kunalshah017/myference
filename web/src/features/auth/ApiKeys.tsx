import { useEffect, useState, type FormEvent } from 'react'
import { AuthAPI, type APIKey } from '../../lib/api'
import { parseMON } from '../../lib/amount'
import { Money } from '../../components/Money'

export function ApiKeys({ api = new AuthAPI() }: { api?: AuthAPI }) {
  const [keys, setKeys] = useState<APIKey[]>([])
  const [revealed, setRevealed] = useState('')
  const [model, setModel] = useState('')
  const [maximum, setMaximum] = useState('')
  const [error, setError] = useState('')
  useEffect(() => { void api.listAPIKeys().then(setKeys).catch((reason) => setError(reason instanceof Error ? reason.message : 'API keys could not be loaded.')) }, [api])
  const create = async (event: FormEvent) => {
    event.preventDefault(); setError('')
    try {
      const key = await api.createAPIKey({ models: [model.trim()], endpoints: ['/v1/chat/completions', '/v1/messages'], max_spend_wei: parseMON(maximum).toString() })
      setRevealed(key.token ?? '')
      setKeys((current) => [{ ...key, token: undefined }, ...current])
    } catch (reason) { setError(reason instanceof Error ? reason.message : 'API key creation failed.') }
  }
  const revoke = async (id: string) => {
    try { await api.revokeAPIKey(id); setKeys((current) => current.filter((key) => key.id !== id)) } catch (reason) { setError(reason instanceof Error ? reason.message : 'API key revocation failed.') }
  }
  return <section className="auth-section" aria-labelledby="api-keys-title"><p className="eyebrow">Project access</p><h2 id="api-keys-title">Scoped API keys</h2>{revealed && <div className="secret-proof" role="status"><span>Copy now — this key is shown once</span><code>{revealed}</code><button type="button" onClick={() => setRevealed('')}>I saved this key</button></div>}<form className="key-form" onSubmit={create}><label htmlFor="key-model">Model</label><input id="key-model" value={model} onChange={(event) => setModel(event.target.value)} required /><label htmlFor="key-maximum">Maximum spend (MON)</label><input id="key-maximum" inputMode="decimal" value={maximum} onChange={(event) => setMaximum(event.target.value)} required /><button type="submit">Create API key</button></form>{error && <p role="alert" className="inline-error">{error}</p>}<ul className="key-list">{keys.map((key) => <li key={key.id}><div><code>{key.id}</code><span>{key.scope.models.join(', ')}</span><span>{key.scope.endpoints.join(', ')}</span><span><Money wei={key.scope.max_spend_wei}/> max</span></div><button type="button" onClick={() => void revoke(key.id)}>Revoke {key.id}</button></li>)}</ul></section>
}
