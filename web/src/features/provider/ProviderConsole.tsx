import { useState, type FormEvent } from 'react'
import { useQuery } from '@tanstack/react-query'
import { OperationsAPI } from '../../lib/api'
import { parseMON } from '../../lib/amount'
import { injectedProvider } from '../../lib/chain'
import { ViemMarketWriter, type MarketWriter, type SubmittedTransaction } from '../../lib/marketContract'
import { Earnings } from './Earnings'
import { Machines } from './Machines'
import { Offers } from './Offers'

export function ProviderConsole({ api=new OperationsAPI(),writer:supplied }:{api?:OperationsAPI;writer?:MarketWriter}){
  const operations=useQuery({queryKey:['operations'],queryFn:()=>api.operations(),retry:false,refetchInterval:15_000});const[bond,setBond]=useState('');const[status,setStatus]=useState('');const[error,setError]=useState('')
  if(operations.isPending)return <p role="status">Loading provider operations…</p>;if(operations.isError)return <p role="alert">Provider operations are unavailable.</p>
  const writer=supplied??new ViemMarketWriter(operations.data,injectedProvider());const submit=async(action:()=>Promise<SubmittedTransaction>)=>{setError('');try{const tx=await action();setStatus(`${tx.hash} pending`);await tx.confirm();setStatus('Transaction finalized. Waiting for indexer.');await operations.refetch()}catch(reason){setError(reason instanceof Error?reason.message:'Transaction failed.')}}
  const deposit=async(event:FormEvent)=>{event.preventDefault();await submit(()=>writer.depositBond(parseMON(bond)))}
  return <section className="operations-section" aria-labelledby="provider-title"><p className="eyebrow">Provider account</p><h2 id="provider-title">Machines, offers, and earnings</h2><Earnings earned={operations.data.provider_earnings_wei} claimable={operations.data.claimable_wei}/><div className="bond-control"><strong>{operations.data.provider_bond_wei} wei bonded</strong><form onSubmit={deposit}><label htmlFor="bond-mon">Bond MON</label><input id="bond-mon" value={bond} onChange={(event)=>setBond(event.target.value)} required/><button type="submit">Deposit collateral</button></form><button type="button" onClick={()=>void submit(()=>operations.data.bond_exit_available_at?writer.finalizeBondExit():writer.requestBondExit())}>{operations.data.bond_exit_available_at?'Finalize bond exit':'Request bond exit'}</button></div>{status&&<p role="status" className="transaction-proof">{status}</p>}{error&&<p role="alert" className="inline-error">{error}</p>}<Machines machines={operations.data.machines}/><Offers offers={operations.data.offers} writer={writer} submit={submit}/></section>
}
