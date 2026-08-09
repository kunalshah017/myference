import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { Money } from '../../components/Money'
import { formatMON, parseMON } from '../../lib/amount'
import type { EditableProviderOffer, ProviderActionInput } from '../../lib/api'

export function Offers({ offers, publish }: { offers: EditableProviderOffer[]; publish: (input: ProviderActionInput) => Promise<void> }) {
  const [offerID, setOfferID] = useState(offers[0]?.offer_id ?? '')
  const [input, setInput] = useState('')
  const [output, setOutput] = useState('')
  const [compute, setCompute] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const selected = useMemo(() => offers.find((offer) => offer.offer_id === offerID) ?? offers[0], [offerID, offers])
  useEffect(() => {
    if (!selected) return
    setOfferID(selected.offer_id)
    setInput(formatMON(BigInt(selected.input_per_million_wei)).replace(' MON', ''))
    setOutput(formatMON(BigInt(selected.output_per_million_wei)).replace(' MON', ''))
    setCompute(formatMON(BigInt(selected.compute_per_second_wei)).replace(' MON', ''))
  }, [selected])
  const rates = (() => { try { return selected ? { input: selected.metering_mode === 'compute_only' ? 0n : parseMON(input), output: selected.metering_mode === 'compute_only' ? 0n : parseMON(output), compute: parseMON(compute) } : undefined } catch { return undefined } })()
  const submit = async (event: FormEvent) => {
    event.preventDefault()
    if (!selected || !rates) return
    setBusy(true); setError('')
    try {
      await publish({ kind: 'publish_offer', offers: [{ offer_id: selected.offer_id, model: selected.model, kind: selected.backend_kind, capabilities: selected.capabilities, metering_mode: selected.metering_mode, input_per_million_wei: rates.input.toString(), output_per_million_wei: rates.output.toString(), compute_per_second_wei: rates.compute.toString() }] })
    } catch (reason) { setError(reason instanceof Error ? reason.message : 'Offer repricing failed.') } finally { setBusy(false) }
  }
  return <section className="provider-card offer-operations" aria-labelledby="offers-title"><div className="provider-card-heading"><div><p className="eyebrow">Pricing</p><h3 id="offers-title">Reprice an existing CLI offer</h3></div><span className="version-count">{offers.length} offer{offers.length === 1 ? '' : 's'}</span></div><p className="provider-card-copy">New offers and model identities are created only in the Myference CLI. Here you can publish a new immutable price version for an existing offer.</p>{offers.length === 0 ? <div className="dashboard-empty"><strong>No published CLI offers</strong><p>Run <code>myference</code> to configure a provider and publish its first offer.</p></div> : <form className="offer-form" onSubmit={(event) => void submit(event)}><div className="provider-field"><label htmlFor="existing-offer">Existing offer</label><select id="existing-offer" value={selected?.offer_id ?? ''} onChange={(event) => setOfferID(event.target.value)}>{offers.map((offer) => <option key={offer.offer_id} value={offer.offer_id}>{offer.offer_id} · {offer.model} · v{offer.version}</option>)}</select><small>The model, backend, capabilities, and metering identity cannot be changed here.</small></div><div className="offer-history"><strong>Current indexed version</strong><ul><li><code>{selected?.offer_id}</code><strong>Version {selected?.version}</strong><span><Money wei={selected?.input_per_million_wei ?? '0'}/> input · <Money wei={selected?.output_per_million_wei ?? '0'}/> output · <Money wei={selected?.compute_per_second_wei ?? '0'}/> compute</span></li></ul></div><fieldset><legend>New MON rates</legend><div className="provider-field-grid provider-pricing-grid"><div className="provider-field"><label htmlFor="offer-input">Input tokens</label><div className="input-with-unit"><input id="offer-input" inputMode="decimal" value={input} onChange={(event) => setInput(event.target.value)} disabled={selected?.metering_mode === 'compute_only'} required={selected?.metering_mode !== 'compute_only'}/><span>MON / 1M</span></div></div><div className="provider-field"><label htmlFor="offer-output">Output tokens</label><div className="input-with-unit"><input id="offer-output" inputMode="decimal" value={output} onChange={(event) => setOutput(event.target.value)} disabled={selected?.metering_mode === 'compute_only'} required={selected?.metering_mode !== 'compute_only'}/><span>MON / 1M</span></div></div><div className="provider-field"><label htmlFor="offer-compute">Compute time</label><div className="input-with-unit"><input id="offer-compute" inputMode="decimal" value={compute} onChange={(event) => setCompute(event.target.value)} required/><span>MON / sec</span></div></div></div></fieldset><div className="provider-form-footer"><p>Only the exact rates above are sent to your wallet.</p><button type="submit" disabled={!rates || busy}>{busy ? 'Waiting for confirmation…' : 'Publish price version'}</button></div>{error && <p role="alert" className="inline-error">{error}</p>}</form>}</section>
}
