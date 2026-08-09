import { useState, type FormEvent } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Money } from '../../components/Money'
import { OperationsAPI, ProviderActivationAPI } from '../../lib/api'
import { parseMON } from '../../lib/amount'
import { injectedProvider } from '../../lib/chain'
import { ViemMarketWriter, type MarketWriter, type SubmittedTransaction } from '../../lib/marketContract'
import { Earnings } from './Earnings'
import { Machines } from './Machines'
import { Offers } from './Offers'

export function ProviderConsole({ api=new OperationsAPI(),activationAPI=new ProviderActivationAPI(),writer:supplied }:{api?:OperationsAPI;activationAPI?:ProviderActivationAPI;writer?:MarketWriter}){
  const operations=useQuery({queryKey:['operations'],queryFn:()=>api.operations(),retry:false,refetchInterval:15_000});const[bond,setBond]=useState('');const[status,setStatus]=useState('');const[error,setError]=useState('')
	const activationID=typeof window==='undefined'?'':new URLSearchParams(window.location.search).get('activation')??''
	const activation=useQuery({queryKey:['provider-activation',activationID],queryFn:()=>activationAPI.get(activationID),enabled:activationID!=='',retry:false})
  if(operations.isPending)return <p role="status">Loading provider operations…</p>;if(operations.isError)return <p role="alert">Provider operations are unavailable.</p>
  const writer=supplied??new ViemMarketWriter(operations.data,injectedProvider());const submit=async(action:()=>Promise<SubmittedTransaction>)=>{setError('');try{const tx=await action();setStatus(`${tx.hash} pending`);const confirmation=await tx.confirm();setStatus('Transaction finalized. Waiting for indexer.');await operations.refetch();return confirmation}catch(reason){setError(reason instanceof Error?reason.message:'Transaction failed.');return undefined}}
  const deposit=async(event:FormEvent)=>{event.preventDefault();await submit(()=>writer.depositBond(parseMON(bond)))}
  return <section className="operations-section" aria-labelledby="provider-title">
    <p className="eyebrow">Provider account</p>
    <h2 id="provider-title">Machines, offers, and collateral</h2>
	{activationID&&activation.isPending&&<p role="status" className="transaction-proof">Loading the terminal activation…</p>}
	{activationID&&activation.isError&&<p role="alert" className="inline-error">This terminal activation is unavailable or has expired. Return to the CLI and retry.</p>}
    <Earnings earned={operations.data.provider_earnings_wei} claimable={operations.data.claimable_wei}/>
    <div className="provider-setup-grid">
      <section className="provider-card collateral-card" aria-labelledby="collateral-title">
        <div className="provider-card-heading">
          <div><p className="eyebrow">Collateral</p><h3 id="collateral-title">Provider collateral</h3></div>
          <div className="provider-balance"><span>Bonded</span><strong><Money wei={operations.data.provider_bond_wei} technical/></strong></div>
        </div>
        <p className="provider-card-copy">Collateral backs requests served by your machines. It remains yours unless a proven violation is slashed.</p>
        <form className="collateral-form" onSubmit={deposit}>
          <div className="provider-field">
            <label htmlFor="bond-mon">Bond amount</label>
            <div className="input-with-unit"><input id="bond-mon" inputMode="decimal" aria-describedby="bond-help" value={bond} onChange={(event)=>setBond(event.target.value)} required/><span>MON</span></div>
            <small id="bond-help">Minimum 5 MON. Deposits become active after chain confirmation.</small>
          </div>
          <div className="provider-actions">
            <button type="submit">Deposit collateral</button>
            <button type="button" className="secondary-action" onClick={()=>void submit(()=>operations.data.bond_exit_available_at?writer.finalizeBondExit():writer.requestBondExit())}>{operations.data.bond_exit_available_at?'Finalize bond exit':'Request bond exit'}</button>
          </div>
        </form>
        <p className="provider-note">Bond exits use the contract delay before funds can be withdrawn.</p>
      </section>
      <Offers offers={operations.data.offers} backends={operations.data.machines.flatMap((machine)=>machine.backends)} writer={writer} submit={submit} activation={activation.data} onActivationComplete={activationID?async(versions)=>{await activationAPI.complete(activationID,versions);setStatus('Offers published. Return to the terminal.')} : undefined}/>
    </div>
    {status&&<p role="status" className="transaction-proof">{status}</p>}
    {error&&<p role="alert" className="inline-error">{error}</p>}
    <Machines machines={operations.data.machines}/>
  </section>
}
