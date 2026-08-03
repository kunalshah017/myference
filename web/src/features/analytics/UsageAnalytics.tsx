import { useQuery } from '@tanstack/react-query'
import { Activity, Clock3, Coins, Hash } from 'lucide-react'
import { AnalyticsAPI, type AnalyticsDay } from '../../lib/api'
import { Money } from '../../components/Money'

function DailyChart({ days, field }: { days: AnalyticsDay[]; field: 'customer_requests' | 'provider_requests' }) {
  const max = Math.max(1, ...days.map((day) => day[field]))
  return <div className="analytics-chart" aria-label="30 day request activity">{days.map((day) => <div key={day.date} title={`${day.date}: ${day[field]} requests`}><i style={{ height: `${Math.max(3, (day[field] / max) * 100)}%` }} /><span>{day.date.slice(5)}</span></div>)}</div>
}

export function UsageAnalytics({ api = new AnalyticsAPI() }: { api?: AnalyticsAPI }) {
  const analytics = useQuery({ queryKey: ['account-analytics'], queryFn: () => api.analytics(), retry: false, refetchInterval: 15_000 })
  if (analytics.isPending) return <div className="dashboard-empty" role="status"><strong>Loading confirmed usage…</strong></div>
  if (analytics.isError) return <div className="dashboard-empty" role="alert"><strong>Usage analytics are unavailable</strong><p>Reconnect the wallet or wait for the indexer to catch up.</p></div>
  const { customer, daily, usage } = analytics.data
  const metrics = [{ label: 'Settled requests', value: customer.settled_requests, icon: Activity }, { label: 'Input tokens', value: customer.input_tokens, icon: Hash }, { label: 'Output tokens', value: customer.output_tokens, icon: Hash }, { label: 'Compute time', value: `${customer.compute_milliseconds} ms`, icon: Clock3 }, { label: 'Provider charges', wei: customer.provider_charges_wei, icon: Coins }, { label: 'Protocol fees', wei: customer.protocol_fees_wei, icon: Coins }, { label: 'Total spent', wei: customer.total_spent_wei, icon: Coins }]
  return <div className="analytics-panel"><div className="analytics-metrics">{metrics.map(({ label, value, wei, icon: Icon }) => <article key={label}><Icon size={18} aria-hidden="true" /><span>{label}</span><strong>{wei === undefined ? value : <Money wei={wei}/>}</strong></article>)}</div><section className="analytics-section"><div className="section-heading"><div><p className="eyebrow">Last 30 days</p><h2>Confirmed requests</h2></div></div><DailyChart days={daily} field="customer_requests" /></section><section className="analytics-section"><p className="eyebrow">Settlement ledger</p><h2>Recent usage</h2>{usage.length === 0 ? <p className="activity-empty">No confirmed inference usage yet.</p> : <div className="analytics-table"><div className="analytics-row head"><span>Request / model</span><span>Tokens</span><span>Compute</span><span>Total</span><span>Confirmed</span></div>{usage.map((item) => <div className="analytics-row" key={item.request_id}><div><code>{item.request_id}</code><small>{item.model}</small></div><span>{item.input_tokens} in / {item.output_tokens} out</span><span>{item.compute_milliseconds} ms</span><strong><Money wei={item.total_charge_wei}/></strong><time dateTime={item.completed_at}>{new Date(item.completed_at).toLocaleString()}</time></div>)}</div>}</section></div>
}

export { DailyChart }
