import { useEffect, useRef } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { AuthAPI, MarketplaceAPI } from '../../lib/api'
import { reconcileRequestEvent, subscribeToActivity, type RequestEvent } from '../../lib/realtime'

type Subscribe = (onEvent: (event: RequestEvent) => void, onReconnect: () => void) => () => void

export function Activity({ api = new MarketplaceAPI(), authApi = new AuthAPI(), connected, subscribe, onState }: { api?: MarketplaceAPI; authApi?: AuthAPI; connected: boolean; subscribe?: Subscribe; onState?: (state: string) => void }) {
  const queryClient = useQueryClient()
  const live = useRef({ cursor: 0, state: '', needsRefetch: false })
  const activity = useQuery({ queryKey: ['activity'], queryFn: () => api.activity(), enabled: connected, retry: false })
  useEffect(() => {
    if (!connected) return
    const connect = subscribe ?? ((onEvent, onReconnect) => subscribeToActivity(authApi, onEvent, onReconnect))
    return connect((event) => {
      live.current = reconcileRequestEvent(live.current, event)
      onState?.(live.current.state)
      void queryClient.invalidateQueries({ queryKey: ['activity'] })
      void queryClient.invalidateQueries({ queryKey: ['account-analytics'] })
    }, () => { void queryClient.invalidateQueries({ queryKey: ['activity'] }); void queryClient.invalidateQueries({ queryKey: ['account-analytics'] }) })
  }, [authApi, connected, onState, queryClient, subscribe])
  useEffect(() => { const latest = activity.data?.[0]?.state; if (latest) onState?.(latest) }, [activity.data, onState])

  if (!connected) return <p className="activity-empty">Connect a wallet to view account-scoped requests and settlements.</p>
  if (activity.isPending) return <p className="activity-empty" role="status">Loading confirmed account activity…</p>
  if (activity.isError) return <p className="activity-empty" role="alert">Account activity is disconnected. Reconnect to the broker to continue.</p>
  if (activity.data.length === 0) return <p className="activity-empty">No inference requests have been recorded for this account.</p>
  return <ol className="activity-ledger">{activity.data.map((item) => <li key={item.request_id}><code>{item.request_id}</code><strong>{item.state}{item.machine_id ? ` on ${item.machine_id}` : ''}</strong><span>{item.model || item.offer_id || 'route pending'}</span><time dateTime={item.updated_at}>{new Date(item.updated_at).toLocaleString()}</time>{item.transaction_hash && <code>{item.transaction_hash}</code>}</li>)}</ol>
}
