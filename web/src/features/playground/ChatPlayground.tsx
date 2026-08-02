import { useMemo, useState, type FormEvent } from 'react'
import { InferenceAPI, type ChatMessage } from '../../lib/api'

export function ChatPlayground({ api: supplied }: { api?: InferenceAPI }) {
  const api = useMemo(() => supplied ?? new InferenceAPI(), [supplied])
  const [model, setModel] = useState('')
  const [apiKey, setAPIKey] = useState('')
  const [maximumSpend, setMaximumSpend] = useState('')
  const [message, setMessage] = useState('')
  const [conversation, setConversation] = useState<ChatMessage[]>([])
  const [error, setError] = useState('')
  const [sending, setSending] = useState(false)
  const send = async (event: FormEvent) => {
    event.preventDefault(); setError(''); setSending(true)
    const next: ChatMessage[] = [...conversation, { role: 'user', content: message.trim() }]
    try {
      const content = await api.chat(model.trim(), apiKey.trim(), maximumSpend, next)
      setConversation([...next, { role: 'assistant', content }]); setMessage('')
    } catch (reason) { setError(reason instanceof Error ? reason.message : 'Inference request failed.') }
    finally { setSending(false) }
  }
  return <div className="playground-grid">
    <form className="playground-config" onSubmit={send}>
      <p className="eyebrow">Request settings</p>
      <label htmlFor="playground-model">Model</label><input id="playground-model" value={model} onChange={(event) => setModel(event.target.value)} placeholder="e.g. qwen" required />
      <label htmlFor="playground-key">API key</label><input id="playground-key" type="password" autoComplete="off" value={apiKey} onChange={(event) => setAPIKey(event.target.value)} placeholder="mf_…" required />
      <p className="form-note">Held in memory for this page only.</p>
      <label htmlFor="playground-maximum">Maximum spend (wei)</label><input id="playground-maximum" inputMode="numeric" pattern="[0-9]+" value={maximumSpend} onChange={(event) => setMaximumSpend(event.target.value)} required />
      <label htmlFor="playground-message">Message</label><textarea id="playground-message" value={message} onChange={(event) => setMessage(event.target.value)} rows={6} required />
      <button type="submit" disabled={sending}>{sending ? 'Routing request…' : 'Send request'}</button>
      {error && <p role="alert" className="inline-error">{error}</p>}
    </form>
    <section className="chat-transcript" aria-label="Chat transcript">
      {conversation.length === 0 ? <div className="dashboard-empty"><strong>No messages yet</strong><p>Choose a live model, enter a scoped API key, and send a test prompt.</p></div> : conversation.map((item, index) => <article key={`${item.role}-${index}`} data-role={item.role}><span>{item.role}</span><p>{item.content}</p></article>)}
    </section>
  </div>
}
