import { useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { AuthAPI, ProviderAPI, type ProviderAction } from '../../lib/api'
import { injectedProvider } from '../../lib/chain'
import { ViemMarketWriter, type MarketWriter } from '../../lib/marketContract'
import { ConnectWallet } from '../auth/ConnectWallet'
import { assertProviderWallet, executeProviderAction, type ProviderActionPhase } from './providerAction'

export function ProviderApproval({ actionID, api = new ProviderAPI(), writer: supplied, checkWallet = assertProviderWallet }: { actionID: string; api?: ProviderAPI; writer?: MarketWriter; checkWallet?: (action: ProviderAction) => Promise<void> }) {
  const account = useQuery({ queryKey: ['provider-account'], queryFn: () => api.account(), retry: false })
  const action = useQuery({ queryKey: ['provider-action', actionID], queryFn: () => api.get(actionID), retry: false, refetchInterval: 2_000 })
  const [status, setStatus] = useState('')
  const [error, setError] = useState('')
  const [phase, setPhase] = useState<ProviderActionPhase>()
  const approvalOpen = useRef(false)
  if (account.isPending || action.isPending) return <main className="approval-page"><p role="status">Loading the exact provider action…</p></main>
  if (account.isError || action.isError) return <main className="approval-page"><section className="provider-card"><p role="alert">Sign in with the provider account wallet. If you are already signed in, this action is unavailable, expired, or belongs to another account.</p><ConnectWallet api={new AuthAPI()} onConnected={() => { void account.refetch(); void action.refetch() }}/></section></main>
  const approve = async () => {
	if (action.data.status !== 'pending_wallet' || approvalOpen.current) return
    approvalOpen.current = true; setPhase('wallet'); setError(''); setStatus('Check the exact values in your wallet.')
    try {
      await checkWallet(action.data)
      const writer = supplied ?? new ViemMarketWriter(account.data, injectedProvider())
      await executeProviderAction(action.data, writer, api, undefined, (next) => {
        setPhase(next)
        setStatus(next === 'wallet' ? 'Confirm the exact action in your wallet.' : next === 'chain' ? 'Transaction submitted. Confirming on Monad…' : 'Transaction finalized. Refreshing indexed action state…')
      })
      await action.refetch()
      setStatus('Confirmed from finalized indexed chain state. You can return to the CLI.')
    } catch (reason) { setStatus(''); setError(reason instanceof Error ? reason.message : 'Provider action failed.') }
    finally { approvalOpen.current = false; setPhase(undefined) }
  }
  const buttonLabel = phase === 'wallet' ? 'Confirm action in wallet…' : phase === 'chain' ? 'Confirming on Monad…' : phase === 'refresh' ? 'Refreshing action…' : action.data.status === 'confirmed' ? 'Action confirmed' : action.data.status === 'pending_chain' ? 'Waiting for indexed confirmation' : 'Approve in wallet'
  return <main className="approval-page"><section className="provider-card" aria-labelledby="approval-title"><p className="eyebrow">CLI wallet approval</p><h1 id="approval-title">Approve provider action</h1><p>The CLI prepared these exact public values. Myference never asks for or stores your wallet key.</p><dl><div><dt>Action</dt><dd>{action.data.kind.replaceAll('_', ' ')}</dd></div><div><dt>Wallet</dt><dd><code>{action.data.wallet_address}</code></dd></div>{action.data.amount_wei && <div><dt>Amount (wei)</dt><dd><code>{action.data.amount_wei}</code></dd></div>}<div><dt>Offers</dt><dd>{action.data.offers?.map((offer) => `${offer.offer_id} · ${offer.model}`).join(', ') || 'None'}</dd></div></dl><button type="button" onClick={() => void approve()} disabled={action.data.status !== 'pending_wallet' || Boolean(phase)}>{buttonLabel}</button>{status && <p role="status" className="transaction-proof">{status}</p>}{error && <p role="alert" className="inline-error">{error}</p>}</section></main>
}
