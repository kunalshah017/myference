import { useMemo, useState, type FormEvent } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Money } from '../../components/Money'
import { computeMinuteUSDToWeiPerSecond, formatMON, isFreshReferencePrice, parseMON, usdToWei } from '../../lib/amount'
import { ReferencePriceAPI, type OperationBackend, type OperationOffer } from '../../lib/api'
import type { MarketWriter, SubmittedTransaction, TransactionConfirmation } from '../../lib/marketContract'

function offerID(backend: OperationBackend): string {
  if (backend.offer_id) return backend.offer_id
  const marker = backend.id.indexOf(':', 'backend:'.length)
  return backend.id.startsWith('backend:') && marker >= 0 ? backend.id.slice(marker + 1) : backend.id
}

export function Offers({ offers, backends, writer, submit }: { offers: OperationOffer[]; backends: OperationBackend[]; writer: MarketWriter; submit: (action:()=>Promise<SubmittedTransaction>)=>Promise<TransactionConfirmation | void> }) {
  const priceAPI = useMemo(() => new ReferencePriceAPI(), [])
  const quote = useQuery({ queryKey: ['reference-price'], queryFn: () => priceAPI.price(), staleTime: 60_000, retry: false })
  const [backendID,setBackendID]=useState(backends[0]?.id ?? '')
  const [activated,setActivated]=useState<{name:string;version:number}>()
  const [activationError,setActivationError]=useState('')
  const [input,setInput]=useState(''); const [output,setOutput]=useState(''); const [compute,setCompute]=useState('')
  const selected = backends.find((backend) => backend.id === backendID) ?? backends[0]
  const computeOnly = selected ? ['codex','claude','kimi'].includes(selected.kind) : false
	const freshQuote = isFreshReferencePrice(quote.data) ? quote.data : undefined
  const rates = (() => { try {
    if (!selected || !compute) return undefined
    const computePerSecond = freshQuote ? computeMinuteUSDToWeiPerSecond(compute, freshQuote.usd_per_mon) : (parseMON(compute) + 59n) / 60n
    return { inputPerMillion: computeOnly ? 0n : freshQuote ? usdToWei(input, freshQuote.usd_per_mon) : parseMON(input), outputPerMillion: computeOnly ? 0n : freshQuote ? usdToWei(output, freshQuote.usd_per_mon) : parseMON(output), computePerSecond }
  } catch { return undefined } })()
  const publish=async(event:FormEvent)=>{event.preventDefault();if(!selected||!rates) return;setActivationError('');const capabilities=['text','stream'];if(computeOnly)capabilities.push('workspace');try{const confirmation=await submit(()=>writer.publishOffer({offerID:offerID(selected),model:selected.model,capabilities,inputPerMillion:rates.inputPerMillion,outputPerMillion:rates.outputPerMillion,computePerSecond:rates.computePerSecond}));if(!confirmation?.offerVersion){setActivationError('Published offer version could not be confirmed from Monad.');return}setActivated({name:offerID(selected),version:confirmation.offerVersion})}catch{return}}
  const unit = freshQuote ? 'USD' : 'MON'
  return <section className="provider-card offer-operations" aria-labelledby="offers-title">
    <div className="provider-card-heading"><div><p className="eyebrow">Pricing</p><h3 id="offers-title">Activate a discovered backend</h3></div><span className="version-count">{offers.length} version{offers.length===1?'':'s'}</span></div>
    <p className="provider-card-copy">Choose a backend reported by your CLI. Myference fills in its exact model and offer identity.</p>
    {offers.length>0&&<div className="offer-history"><strong>Published versions</strong><ul>{offers.map((offer)=><li key={`${offer.offer_id}:${offer.version}`}><code>{offer.offer_id}</code><strong>Version {offer.version}</strong><span><Money wei={offer.input_per_million_wei}/> input · <Money wei={offer.output_per_million_wei}/> output · <Money wei={offer.compute_per_second_wei}/> compute</span></li>)}</ul></div>}
    {backends.length===0?<div className="dashboard-empty"><strong>No CLI backend discovered</strong><p>Run <code>myference host</code> on the machine first.</p></div>:<form className="offer-form" onSubmit={(event)=>void publish(event)}>
      <div className="provider-field"><label htmlFor="offer-backend">Backend and model</label><select id="offer-backend" value={selected?.id ?? ''} onChange={(event)=>setBackendID(event.target.value)}>{backends.map((backend)=><option key={backend.id} value={backend.id}>{backend.kind} · {backend.model} · {backend.healthy?'online':'awaiting heartbeat'}</option>)}</select><small>Detected by the connected provider CLI. No model name to copy.</small></div>
      <fieldset><legend>Usage pricing</legend><p className="provider-note">Enter {unit} targets. {freshQuote?`Converted using 1 MON = $${freshQuote.usd_per_mon}; the MON result is locked on-chain.`:'The live USD quote is unavailable, so enter MON rates directly.'}</p><div className="provider-field-grid provider-pricing-grid">
        <div className="provider-field"><label htmlFor="offer-input">Input tokens</label><div className="input-with-unit"><input id="offer-input" inputMode="decimal" aria-describedby="offer-input-help" value={input} onChange={(event)=>setInput(event.target.value)} disabled={computeOnly} required={!computeOnly}/><span>{unit} / 1M</span></div><small id="offer-input-help">{computeOnly?'CLI agents use compute-only pricing.':`${unit} for one million input tokens.`}</small></div>
        <div className="provider-field"><label htmlFor="offer-output">Output tokens</label><div className="input-with-unit"><input id="offer-output" inputMode="decimal" aria-describedby="offer-output-help" value={output} onChange={(event)=>setOutput(event.target.value)} disabled={computeOnly} required={!computeOnly}/><span>{unit} / 1M</span></div><small id="offer-output-help">{computeOnly?'CLI agents use compute-only pricing.':`${unit} for one million output tokens.`}</small></div>
        <div className="provider-field"><label htmlFor="offer-compute">Compute time</label><div className="input-with-unit"><input id="offer-compute" inputMode="decimal" aria-describedby="offer-compute-help" value={compute} onChange={(event)=>setCompute(event.target.value)} required/><span>{unit} / min</span></div><small id="offer-compute-help">{unit} for each minute of measured compute.</small></div>
      </div></fieldset>
      {rates&&<div className="price-preview" role="status"><strong>Locked on-chain rates</strong><span>{formatMON(rates.inputPerMillion)} / 1M input</span><span>{formatMON(rates.outputPerMillion)} / 1M output</span><span>{formatMON(rates.computePerSecond)} / compute second</span></div>}
      <div className="provider-form-footer"><p>Publishing creates a new immutable on-chain price version.</p><button type="submit" disabled={!rates}>Publish offer</button></div>
	  {activated && <p className="provider-note" role="status">Activate published version {activated.version} on the provider machine: <code>myference backend version --name {activated.name} --price-version {activated.version}</code>. The running daemon reloads it automatically.</p>}
	  {activationError && <p role="alert" className="inline-error">{activationError}</p>}
    </form>}
  </section>
}
