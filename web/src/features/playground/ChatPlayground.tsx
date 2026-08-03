import { useMemo, useState, type FormEvent } from 'react'
import { useQuery } from '@tanstack/react-query'
import { parseMON } from '../../lib/amount'
import { InferenceAPI, MarketplaceAPI, type ChatMessage } from '../../lib/api'

export function ChatPlayground({ api: supplied, marketplace: suppliedMarketplace }: { api?: InferenceAPI; marketplace?: MarketplaceAPI }) {
  const api = useMemo(() => supplied ?? new InferenceAPI(), [supplied])
  const marketplace = useMemo(() => suppliedMarketplace ?? new MarketplaceAPI(), [suppliedMarketplace])
  const inventory = useQuery({ queryKey: ['models'], queryFn: () => marketplace.models(), refetchInterval: 15_000, retry: false })
  const [model, setModel] = useState('')
  const [apiKey, setAPIKey] = useState('')
  const [showKey, setShowKey] = useState(false)
  const [maximumSpend, setMaximumSpend] = useState('')
  const [message, setMessage] = useState('')
  const [conversation, setConversation] = useState<ChatMessage[]>([])
  const [error, setError] = useState('')
  const [sending, setSending] = useState(false)
  const send = async (event: FormEvent) => {
    event.preventDefault(); setError(''); setSending(true)
    const next: ChatMessage[] = [...conversation, { role: 'user', content: message.trim() }]
    try {
      const content = await api.chat((model || liveModels[0]?.model).trim(), apiKey.trim(), parseMON(maximumSpend).toString(), next)
      setConversation([...next, { role: 'assistant', content }]); setMessage('')
    } catch (reason) { setError(reason instanceof Error ? reason.message : 'Inference request failed.') }
    finally { setSending(false) }
  }
  const liveModels = inventory.data?.filter((item) => !item.stale && item.available_providers > 0 && item.total_capacity > 0) ?? []
  return <div className="playground-grid">
    <form className="playground-config" onSubmit={send}>
      <p className="eyebrow">Request settings</p>
      <label htmlFor="playground-model">Model</label><select id="playground-model" value={model || liveModels[0]?.model || ''} onChange={(event) => setModel(event.target.value)} disabled={inventory.isPending || liveModels.length===0} required><option value="">{inventory.isPending?'Loading live models…':liveModels.length===0?'No live models available':'Select a model'}</option>{liveModels.map((item)=><option key={item.model} value={item.model}>{item.model}</option>)}</select>
      <label htmlFor="playground-key">API key</label><div className="secret-input"><input id="playground-key" type="text" className={showKey?'':'secret-masked'} autoComplete="off" data-1p-ignore="true" data-lpignore="true" data-form-type="other" spellCheck={false} value={apiKey} onChange={(event) => setAPIKey(event.target.value)} placeholder="mf_…" required /><button type="button" onClick={()=>setShowKey((current)=>!current)}>{showKey?'Hide':'Show'}</button></div>
      <p className="form-note">Held in memory for this page only.</p>
      <label htmlFor="playground-maximum">Maximum spend (MON)</label><input id="playground-maximum" inputMode="decimal" value={maximumSpend} onChange={(event) => setMaximumSpend(event.target.value)} required />
      <label htmlFor="playground-message">Message</label><textarea id="playground-message" value={message} onChange={(event) => setMessage(event.target.value)} rows={6} required />
      <button type="submit" disabled={sending}>{sending ? 'Routing request…' : 'Send request'}</button>
      {error && <p role="alert" className="inline-error">{error}</p>}
    </form>
    <section className="chat-transcript" aria-label="Chat transcript">
      {conversation.length === 0 ? <div className="dashboard-empty"><strong>No messages yet</strong><p>Choose a live model, enter a scoped API key, and send a test prompt.</p></div> : conversation.map((item, index) => <article key={`${item.role}-${index}`} data-role={item.role}><span>{item.role}</span><p>{item.content}</p></article>)}
    </section>
  </div>
}
