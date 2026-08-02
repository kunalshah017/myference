import { useQuery } from '@tanstack/react-query'
import { Banknote, Coins, Cpu, KeyRound, Server, ShieldAlert } from 'lucide-react'
import { OperationsAPI } from '../lib/api'

export function DashboardOverview({ api = new OperationsAPI(), onNavigate }: { api?: OperationsAPI; onNavigate?: (view: 'models' | 'playground' | 'api' | 'hosting') => void }) {
  const operations = useQuery({ queryKey: ['operations'], queryFn: () => api.operations(), retry: false, refetchInterval: 15_000 })
  if (operations.isPending) return <div className="dashboard-empty" role="status"><strong>Loading account workspace…</strong></div>
  if (operations.isError) return <div className="dashboard-empty" role="alert"><strong>Account operations are unavailable</strong><p>Reconnect the wallet or wait for the indexer to catch up.</p></div>
  const data = operations.data
  const metrics = [
    { label: 'Available to spend', value: `${data.customer_balance_wei} wei`, icon: Coins },
    { label: 'Provider earnings', value: `${data.provider_earnings_wei} wei`, icon: Banknote },
    { label: 'Bonded collateral', value: `${data.provider_bond_wei} wei`, icon: ShieldAlert },
    { label: 'Connected machines', value: String(data.machines.length), icon: Server },
  ]
  return <section>
    <div className="workspace-heading"><div><p className="eyebrow">Account workspace</p><h1>Workspace overview</h1><p className="dashboard-intro">Use hosted models and offer inference from one wallet-bound account.</p></div><span className="data-label">Chain {data.chain_id} · {data.confirmations} confirmation</span></div>
    <div className="metric-grid">{metrics.map(({ label, value, icon: Icon }) => <article key={label}><Icon aria-hidden="true" size={20} /><span>{label}</span><strong>{value}</strong></article>)}</div>
    <div className="workspace-actions"><button type="button" onClick={() => onNavigate?.('models')}><Cpu size={19} aria-hidden="true" /><span><strong>Browse live models</strong><small>Compare providers, capacity, and prices.</small></span></button><button type="button" onClick={() => onNavigate?.('playground')}><KeyRound size={19} aria-hidden="true" /><span><strong>Test an inference</strong><small>Send a request from the browser playground.</small></span></button><button type="button" onClick={() => onNavigate?.('hosting')}><Server size={19} aria-hidden="true" /><span><strong>Host inference</strong><small>Manage machines, backends, stake, and offers.</small></span></button></div>
  </section>
}
