import { useState, type FormEvent } from 'react'
import { useQuery } from '@tanstack/react-query'
import { OperationsAPI } from '../../lib/api'
import { parseMON } from '../../lib/amount'
import { Money } from '../../components/Money'
import { ViemMarketWriter, type MarketWriter, type SubmittedTransaction } from '../../lib/marketContract'
import { injectedProvider } from '../../lib/chain'
import { SpendingSession } from './SpendingSession'

export function Billing({ api = new OperationsAPI(), writer: supplied }: { api?: OperationsAPI; writer?: MarketWriter }) {
  const operations = useQuery({ queryKey:['operations'],queryFn:()=>api.operations(),retry:false,refetchInterval:15_000 })
  const [deposit,setDeposit]=useState(''); const [withdraw,setWithdraw]=useState(''); const [status,setStatus]=useState(''); const [error,setError]=useState('')
  if (operations.isPending) return <p role="status">Loading indexed MON balances…</p>
  if (operations.isError) return <p role="alert">Indexed billing data is unavailable.</p>
  const writer = supplied ?? new ViemMarketWriter(operations.data,injectedProvider())
  const submit = async (action:()=>Promise<SubmittedTransaction>) => { setError(''); try { const transaction=await action(); setStatus(`${transaction.hash} pending`); await transaction.confirm(); setStatus('Transaction finalized. Waiting for indexed balance.'); await operations.refetch() } catch(reason){ setStatus(''); setError(reason instanceof Error?reason.message:'Transaction failed.') } }
  const depositMON=async(event:FormEvent)=>{event.preventDefault();await submit(()=>writer.deposit(parseMON(deposit)))}
  const withdrawMON=async(event:FormEvent)=>{event.preventDefault();await submit(()=>writer.requestWithdrawal(parseMON(withdraw)))}
  return <section className="operations-section" aria-labelledby="billing-title"><p className="eyebrow">Native MON escrow</p><h2 id="billing-title">Billing and spending</h2><div className="balance-strip"><strong><Money wei={operations.data.customer_balance_wei}/> available</strong><span><Money wei={operations.data.claimable_wei}/> claimable</span><span>Finality: {operations.data.confirmations} blocks</span></div>{status&&<p role="status" className="transaction-proof">{status}</p>}{error&&<p role="alert" className="inline-error">{error}</p>}<div className="operation-forms"><form onSubmit={depositMON}><label htmlFor="deposit-mon">Deposit MON</label><input id="deposit-mon" value={deposit} onChange={(event)=>setDeposit(event.target.value)} required /><button type="submit">Deposit to escrow</button></form><form onSubmit={withdrawMON}><label htmlFor="withdraw-mon">Withdraw MON</label><input id="withdraw-mon" value={withdraw} onChange={(event)=>setWithdraw(event.target.value)} required /><button type="submit">Request withdrawal</button></form><button type="button" onClick={()=>void submit(()=>writer.claim())}>Claim available MON</button></div><SpendingSession writer={writer} sessions={operations.data.sessions} submit={submit} /></section>
}
