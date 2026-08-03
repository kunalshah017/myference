import { useMemo, useState, type FormEvent } from 'react'
import { useQuery } from '@tanstack/react-query'
import { formatMON, parseMON } from '../../lib/amount'
import { InferenceAPI, MarketplaceAPI, type ChatMessage, type MarketOffer } from '../../lib/api'

const COMPUTE_LIMIT_MS = 120_000n

function ceilDivide(value: bigint, divisor: bigint) { return (value + divisor - 1n) / divisor }

function estimateReservation(model: MarketOffer, messages: ChatMessage[], maximumOutputTokens: number) {
  const prompt = messages.map((item) => `${item.role}: ${item.content}\n`).join('')
  const maximumInputTokens = BigInt(new TextEncoder().encode(prompt).length) * 4n + 256n
  return ceilDivide(maximumInputTokens * BigInt(model.input_per_million_wei), 1_000_000n)
    + ceilDivide(BigInt(maximumOutputTokens) * BigInt(model.output_per_million_wei), 1_000_000n)
    + ceilDivide(COMPUTE_LIMIT_MS * BigInt(model.compute_per_second_wei), 1_000n)
}

function minimumReservation(offers: MarketOffer[], messages: ChatMessage[], maximumOutputTokens: number) {
  return offers.reduce<bigint | undefined>((minimum, offer) => {
    const cost = estimateReservation(offer, messages, maximumOutputTokens)
    return minimum === undefined || cost < minimum ? cost : minimum
  }, undefined)
}

export function ChatPlayground({ api: supplied, marketplace: suppliedMarketplace }: { api?: InferenceAPI; marketplace?: MarketplaceAPI }) {
  const api = useMemo(() => supplied ?? new InferenceAPI(), [supplied])
  const marketplace = useMemo(() => suppliedMarketplace ?? new MarketplaceAPI(), [suppliedMarketplace])
  const inventory = useQuery({ queryKey: ['models'], queryFn: () => marketplace.models(), refetchInterval: 15_000, retry: false })
  const [model, setModel] = useState('')
  const [apiKey, setAPIKey] = useState('')
  const [showKey, setShowKey] = useState(false)
  const [maximumSpend, setMaximumSpend] = useState('1')
  const [maximumOutputTokens, setMaximumOutputTokens] = useState('256')
  const [message, setMessage] = useState('')
  const [conversation, setConversation] = useState<ChatMessage[]>([])
  const [error, setError] = useState('')
  const [sending, setSending] = useState(false)
  const liveModels = inventory.data?.filter((item) => !item.stale && item.available_providers > 0 && item.total_capacity > 0) ?? []
  const selectedModelName = model || liveModels[0]?.model || ''
  const detail = useQuery({ queryKey: ['model', selectedModelName, 'playground-pricing'], queryFn: () => marketplace.model(selectedModelName), enabled: selectedModelName !== '', retry: false })
  const liveOffers = detail.data?.offers.filter((offer) =>
    offer.available &&
    !offer.stale &&
    offer.capacity > 0 &&
    ['text', 'stream'].every((capability) => offer.capabilities.includes(capability)) &&
    (offer.input_per_million_wei !== '0' || offer.output_per_million_wei !== '0' || offer.compute_per_second_wei !== '0')
  ) ?? []
  const send = async (event: FormEvent) => {
    event.preventDefault(); setError(''); setSending(true)
    const next: ChatMessage[] = [...conversation, { role: 'user', content: message.trim() }]
    try {
      const selectedModel = (model || liveModels[0]?.model || '').trim()
      if (!selectedModel) throw new Error('Choose an available model before sending.')
      const outputLimit = Number(maximumOutputTokens)
      if (!Number.isSafeInteger(outputLimit) || outputLimit < 1 || outputLimit > 1_000_000) throw new Error('Maximum output tokens must be an integer from 1 to 1,000,000.')
      if (liveOffers.length === 0) throw new Error('No live offer pricing is available for the selected model.')
      const maximum = parseMON(maximumSpend)
      const required = minimumReservation(liveOffers, next, outputLimit)
      if (required === undefined) throw new Error('No live offer pricing is available for the selected model.')
      if (maximum < required) throw new Error(`Maximum spend must be at least ${formatMON(required)} for this request.`)
      const content = await api.chat(selectedModel, apiKey.trim(), maximum.toString(), next, outputLimit)
      setConversation([...next, { role: 'assistant', content }]); setMessage('')
    } catch (reason) { setError(reason instanceof Error ? reason.message : 'Inference request failed.') }
    finally { setSending(false) }
  }
  const outputLimit = Number(maximumOutputTokens)
  const estimate = Number.isSafeInteger(outputLimit) && outputLimit > 0 && outputLimit <= 1_000_000 ? minimumReservation(liveOffers, [...conversation, { role: 'user', content: message.trim() }], outputLimit) : undefined
  return <div className="playground-grid">
    <form className="playground-config" onSubmit={send}>
      <p className="eyebrow">Request settings</p>
	  {inventory.isError && <p role="alert" className="inline-error">Live model inventory could not be loaded.</p>}
      <label htmlFor="playground-model">Model</label><select id="playground-model" value={model || liveModels[0]?.model || ''} onChange={(event) => setModel(event.target.value)} disabled={inventory.isPending || liveModels.length===0} required><option value="">{inventory.isPending?'Loading live models…':liveModels.length===0?'No live models available':'Select a model'}</option>{liveModels.map((item)=><option key={item.model} value={item.model}>{item.model}</option>)}</select>
      <label htmlFor="playground-key">API key</label><div className="secret-input"><input id="playground-key" type="text" className={showKey?'':'secret-masked'} autoComplete="off" data-1p-ignore="true" data-lpignore="true" data-form-type="other" spellCheck={false} value={apiKey} onChange={(event) => setAPIKey(event.target.value)} placeholder="mf_…" required /><button type="button" onClick={()=>setShowKey((current)=>!current)}>{showKey?'Hide':'Show'}</button></div>
      <p className="form-note">Held in memory for this page only.</p>
      <label htmlFor="playground-maximum">Maximum spend (MON)</label><input id="playground-maximum" inputMode="decimal" value={maximumSpend} onChange={(event) => setMaximumSpend(event.target.value)} required />
      <label htmlFor="playground-output-limit">Maximum output tokens</label><input id="playground-output-limit" type="number" min="1" max="1000000" step="1" inputMode="numeric" value={maximumOutputTokens} onChange={(event) => setMaximumOutputTokens(event.target.value)} required />
      {estimate !== undefined && <p className="form-note">Estimated reservation: {formatMON(estimate)}. Actual measured usage may cost less.</p>}
      <label htmlFor="playground-message">Message</label><textarea id="playground-message" value={message} onChange={(event) => setMessage(event.target.value)} rows={6} required />
      <button type="submit" disabled={sending || detail.isPending}>{sending ? 'Routing request…' : detail.isPending ? 'Loading offer prices…' : 'Send request'}</button>
      {error && <p role="alert" className="inline-error">{error}</p>}
    </form>
    <section className="chat-transcript" aria-label="Chat transcript">
      {conversation.length === 0 ? <div className="dashboard-empty"><strong>No messages yet</strong><p>Choose a live model, enter an API key, and send a test prompt.</p></div> : conversation.map((item, index) => <article key={`${item.role}-${index}`} data-role={item.role}><span>{item.role}</span><p>{item.content}</p></article>)}
    </section>
  </div>
}
