import { useRef, useState, type FormEvent } from 'react'
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
  const [pending, setPending] = useState<{ id: string; label: string; phase: 'wallet' | 'chain' | 'refresh' }>()
  const transactionOpen = useRef(false)
  if (operations.isPending) return <p role="status">Loading indexed MON balances…</p>
  if (operations.isError) return <p role="alert">Indexed billing data is unavailable.</p>
  const writer = supplied ?? new ViemMarketWriter(operations.data,injectedProvider())
  const submit = async (id: string, label: string, action:()=>Promise<SubmittedTransaction>) => {
    if (transactionOpen.current) return
    transactionOpen.current = true; setError(''); setPending({ id, label, phase: 'wallet' }); setStatus(`Confirm ${label.toLowerCase()} in your wallet.`)
    try {
      const transaction=await action()
      setPending({ id, label, phase: 'chain' }); setStatus(`${transaction.hash} pending on Monad.`)
      await transaction.confirm()
      setPending({ id, label, phase: 'refresh' }); setStatus('Transaction finalized. Refreshing indexed account balances…')
      await operations.refetch()
      setStatus('Transaction finalized and account balances refreshed.')
    } catch(reason){ setStatus(''); setError(reason instanceof Error?reason.message:'Transaction failed.') }
    finally { transactionOpen.current = false; setPending(undefined) }
  }
  const buttonLabel = (id: string, fallback: string) => pending?.id === id ? pending.phase === 'wallet' ? `Confirm ${pending.label.toLowerCase()} in wallet…` : pending.phase === 'chain' ? 'Confirming on Monad…' : 'Refreshing account…' : fallback
  const depositMON=async(event:FormEvent)=>{event.preventDefault();await submit('deposit', 'Deposit', ()=>writer.deposit(parseMON(deposit)))}
  const withdrawMON=async(event:FormEvent)=>{event.preventDefault();await submit('withdraw', 'Withdrawal', ()=>writer.requestWithdrawal(parseMON(withdraw)))}
  return <section className="operations-section" aria-labelledby="billing-title"><p className="eyebrow">Native MON escrow</p><h2 id="billing-title">Billing and spending</h2><div className="balance-strip"><strong><Money wei={operations.data.customer_balance_wei}/> available</strong><span><Money wei={operations.data.claimable_wei}/> claimable</span><span>Finality: {operations.data.confirmations} blocks</span></div>{status&&<p role="status" className="transaction-proof">{status}</p>}{error&&<p role="alert" className="inline-error">{error}</p>}<div className="operation-forms"><form onSubmit={depositMON}><label htmlFor="deposit-mon">Deposit MON</label><input id="deposit-mon" value={deposit} onChange={(event)=>setDeposit(event.target.value)} disabled={Boolean(pending)} required /><button type="submit" disabled={Boolean(pending)}>{buttonLabel('deposit', 'Deposit to escrow')}</button></form><form onSubmit={withdrawMON}><label htmlFor="withdraw-mon">Withdraw MON</label><input id="withdraw-mon" value={withdraw} onChange={(event)=>setWithdraw(event.target.value)} disabled={Boolean(pending)} required /><button type="submit" disabled={Boolean(pending)}>{buttonLabel('withdraw', 'Request withdrawal')}</button></form><button type="button" disabled={Boolean(pending)} onClick={()=>void submit('claim', 'Claim', ()=>writer.claim())}>{buttonLabel('claim', 'Claim available MON')}</button></div><SpendingSession writer={writer} sessions={operations.data.sessions} submit={submit} pending={pending} /></section>
}
