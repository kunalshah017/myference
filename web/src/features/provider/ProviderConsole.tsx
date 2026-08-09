import { useRef, useState, type FormEvent } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Money } from '../../components/Money'
import { ProviderAPI, type ProviderActionInput } from '../../lib/api'
import { parseMON } from '../../lib/amount'
import { injectedProvider } from '../../lib/chain'
import { ViemMarketWriter, type MarketWriter } from '../../lib/marketContract'
import { Earnings } from './Earnings'
import { Offers } from './Offers'
import { assertProviderWallet, executeProviderAction, type ProviderActionPhase } from './providerAction'

export function ProviderConsole({ api = new ProviderAPI(), writer: supplied }: { api?: ProviderAPI; writer?: MarketWriter }) {
  const provider = useQuery({ queryKey: ['provider-account'], queryFn: () => api.account(), retry: false, refetchInterval: 15_000 })
  const [bond, setBond] = useState('')
  const [status, setStatus] = useState('')
  const [error, setError] = useState('')
  const [pending, setPending] = useState<{ kind: ProviderActionInput['kind']; phase: ProviderActionPhase }>()
  const actionOpen = useRef(false)
  if (provider.isPending) return <p role="status">Loading provider account…</p>
  if (provider.isError) return <p role="alert">Provider account is unavailable.</p>
  const writer = supplied ?? new ViemMarketWriter(provider.data, injectedProvider())
  const perform = async (input: ProviderActionInput) => {
    if (actionOpen.current) return
    actionOpen.current = true; setPending({ kind: input.kind, phase: 'wallet' }); setError(''); setStatus('Preparing exact wallet action…')
    try {
      const action = await api.create(input)
		if (!supplied) await assertProviderWallet(action)
      await executeProviderAction(action, writer, api, undefined, (phase) => {
        setPending({ kind: input.kind, phase })
        setStatus(phase === 'wallet' ? 'Confirm the exact provider action in your wallet.' : phase === 'chain' ? 'Transaction submitted. Confirming on Monad…' : 'Transaction finalized. Refreshing indexed provider account…')
      })
      await provider.refetch()
      setStatus('Provider account refreshed from finalized indexed chain state.')
    } catch (reason) { setStatus(''); setError(reason instanceof Error ? reason.message : 'Provider action failed.') }
    finally { actionOpen.current = false; setPending(undefined) }
  }
  const pendingLabel = (kind: ProviderActionInput['kind'], fallback: string, walletLabel: string) => pending?.kind === kind ? pending.phase === 'wallet' ? walletLabel : pending.phase === 'chain' ? 'Confirming on Monad…' : 'Refreshing account…' : fallback
  const deposit = async (event: FormEvent) => { event.preventDefault(); await perform({ kind: 'deposit_collateral', amount_wei: parseMON(bond).toString() }) }
  const exitKind = provider.data.bond_exit_available_at ? 'finalize_collateral_exit' : 'request_collateral_exit'
  return <section className="operations-section" aria-labelledby="provider-title"><p className="eyebrow">Provider account</p><h2 id="provider-title">Collateral and existing offer pricing</h2><p className="provider-card-copy">Provider discovery, model setup, first publication, and hosting stay in the Myference CLI. This account view only manages wallet-backed collateral and reprices offers the CLI already created.</p><Earnings earned={provider.data.provider_earnings_wei} claimable={provider.data.claimable_wei}/><div className="provider-setup-grid"><section className="provider-card collateral-card" aria-labelledby="collateral-title"><div className="provider-card-heading"><div><p className="eyebrow">Collateral</p><h3 id="collateral-title">Provider collateral</h3></div><div className="provider-balance"><span>Bonded</span><strong><Money wei={provider.data.provider_bond_wei} technical/></strong></div></div><p className="provider-card-copy">Collateral backs requests served by your CLI providers. Wallet transactions are confirmed only after the indexer observes their finalized effect.</p><form className="collateral-form" onSubmit={(event) => void deposit(event)}><div className="provider-field"><label htmlFor="bond-mon">Bond amount</label><div className="input-with-unit"><input id="bond-mon" inputMode="decimal" value={bond} onChange={(event) => setBond(event.target.value)} disabled={Boolean(pending)} required/><span>MON</span></div><small>Contract minimum: <Money wei={provider.data.minimum_bond_wei}/>.</small></div><div className="provider-actions"><button type="submit" disabled={Boolean(pending)}>{pendingLabel('deposit_collateral', 'Deposit collateral', 'Confirm deposit in wallet…')}</button><button type="button" className="secondary-action" disabled={Boolean(pending)} onClick={() => void perform({ kind: exitKind })}>{pendingLabel(exitKind, provider.data.bond_exit_available_at ? 'Finalize collateral exit' : 'Request collateral exit', 'Confirm collateral action in wallet…')}</button></div></form></section><Offers offers={provider.data.offers} publish={perform} disabled={Boolean(pending)} pendingLabel={pending?.kind === 'publish_offer' ? pendingLabel('publish_offer', 'Publish price version', 'Confirm price update in wallet…') : undefined}/></div>{status && <p role="status" className="transaction-proof">{status}</p>}{error && <p role="alert" className="inline-error">{error}</p>}</section>
}
