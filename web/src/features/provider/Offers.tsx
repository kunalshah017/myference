import { useState, type FormEvent } from 'react'
import type { OperationOffer } from '../../lib/api'
import type { MarketWriter, SubmittedTransaction } from '../../lib/marketContract'

export function Offers({ offers, writer, submit }: { offers: OperationOffer[]; writer: MarketWriter; submit: (action:()=>Promise<SubmittedTransaction>)=>Promise<void> }) {
  const [id,setID]=useState(''); const [model,setModel]=useState(''); const [input,setInput]=useState(''); const [output,setOutput]=useState(''); const [compute,setCompute]=useState(''); const [workspace,setWorkspace]=useState(false)
  const publish=async(event:FormEvent)=>{event.preventDefault();for(const value of [input,output,compute]) if(!/^\d+$/.test(value)) throw new Error('Prices must be integer wei.');await submit(()=>writer.publishOffer({offerID:id,model,capabilities:workspace?['text','stream','workspace']:['text','stream'],inputPerMillion:BigInt(input),outputPerMillion:BigInt(output),computePerSecond:BigInt(compute)}))}
  return <section className="provider-card offer-operations" aria-labelledby="offers-title">
    <div className="provider-card-heading">
      <div><p className="eyebrow">Pricing</p><h3 id="offers-title">Publish an offer</h3></div>
      <span className="version-count">{offers.length} version{offers.length===1?'':'s'}</span>
    </div>
    <p className="provider-card-copy">Set the exact model and usage rates your machine will advertise. Existing sessions keep the price version they selected.</p>
    {offers.length>0&&<div className="offer-history"><strong>Published versions</strong><ul>{offers.map((offer)=><li key={`${offer.offer_id}:${offer.version}`}><code>{offer.offer_id}</code><strong>Version {offer.version}</strong><span>{offer.input_per_million_wei} input · {offer.output_per_million_wei} output · {offer.compute_per_second_wei} compute wei</span></li>)}</ul></div>}
    <form className="offer-form" onSubmit={(event)=>void publish(event)}>
      <fieldset>
        <legend>Model identity</legend>
        <div className="provider-field-grid">
          <div className="provider-field"><label htmlFor="offer-id">Offer name</label><input id="offer-id" aria-describedby="offer-id-help" placeholder="local-qwen" value={id} onChange={(event)=>setID(event.target.value)} required/><small id="offer-id-help">A stable name used by your CLI backend.</small></div>
          <div className="provider-field"><label htmlFor="offer-model">Model</label><input id="offer-model" aria-describedby="offer-model-help" placeholder="qwen2.5:0.5b" value={model} onChange={(event)=>setModel(event.target.value)} required/><small id="offer-model-help">Must exactly match the model advertised by the machine.</small></div>
        </div>
      </fieldset>
      <fieldset>
        <legend>Usage pricing</legend>
        <div className="provider-field-grid provider-pricing-grid">
          <div className="provider-field"><label htmlFor="offer-input">Input tokens</label><div className="input-with-unit"><input id="offer-input" inputMode="numeric" pattern="[0-9]+" aria-describedby="offer-input-help" placeholder="1000000000000" value={input} onChange={(event)=>setInput(event.target.value)} required/><span>wei / 1M</span></div><small id="offer-input-help">Charge in wei for one million input tokens.</small></div>
          <div className="provider-field"><label htmlFor="offer-output">Output tokens</label><div className="input-with-unit"><input id="offer-output" inputMode="numeric" pattern="[0-9]+" aria-describedby="offer-output-help" placeholder="2000000000000" value={output} onChange={(event)=>setOutput(event.target.value)} required/><span>wei / 1M</span></div><small id="offer-output-help">Charge in wei for one million generated tokens.</small></div>
          <div className="provider-field"><label htmlFor="offer-compute">Compute time</label><div className="input-with-unit"><input id="offer-compute" inputMode="numeric" pattern="[0-9]+" aria-describedby="offer-compute-help" placeholder="1000000000" value={compute} onChange={(event)=>setCompute(event.target.value)} required/><span>wei / sec</span></div><small id="offer-compute-help">Charge in wei for each second of measured compute.</small></div>
        </div>
      </fieldset>
      <label className="workspace-option" htmlFor="offer-workspace"><input id="offer-workspace" type="checkbox" checked={workspace} onChange={(event)=>setWorkspace(event.target.checked)}/><span><strong>Allow temporary workspace</strong><small>Enable only for isolated CLI coding agents that accept disposable project files.</small></span></label>
      <div className="provider-form-footer"><p>Publishing creates a new immutable on-chain price version.</p><button type="submit">Publish next offer version</button></div>
    </form>
  </section>
}
