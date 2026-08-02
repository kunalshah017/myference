import { useEffect, useMemo, useState } from 'react'
import { Activity } from '../features/activity/Activity'
import { RoutingRail } from '../features/activity/RoutingRail'
import { ApiKeys } from '../features/auth/ApiKeys'
import { ConnectWallet } from '../features/auth/ConnectWallet'
import { DeviceApproval } from '../features/auth/DeviceApproval'
import { Billing } from '../features/billing/Billing'
import { ModelList } from '../features/marketplace/ModelList'
import { ProviderConsole } from '../features/provider/ProviderConsole'
import { AuthAPI, MarketplaceAPI, type Session } from '../lib/api'
import LandingPage from './LandingPage'

function OperationalApp() {
  const api = useMemo(() => new AuthAPI(), [])
  const marketplace = useMemo(() => new MarketplaceAPI(), [])
  const [session, setSession] = useState<Session>()
  const [routeState, setRouteState] = useState('')
  useEffect(() => { void api.session().then(setSession).catch(() => undefined) }, [api])
  return (
    <div className="app-shell">
      <header className="site-header">
        <a className="wordmark" href="/" aria-label="Myference home">
          <span aria-hidden="true">M/</span> Myference
        </a>
        <nav aria-label="Primary navigation">
          <a href="#marketplace">Models</a>
          <a href="#activity">Activity</a>
          <a href="https://github.com/kunalshah017/myference">Docs</a>
        </nav>
        <div className="network-actions">
          <span className="network-state">
            <span className="state-mark" aria-hidden="true" />
            {session ? 'Monad testnet account connected' : 'Network not connected'}
          </span>
          <ConnectWallet api={api} onConnected={setSession} />
        </div>
      </header>

      <main>
        <section className="hero" aria-labelledby="hero-title">
          <p className="eyebrow">Distributed inference market / Monad</p>
          <h1 id="hero-title">Unused machines, useful inference.</h1>
          <p className="hero-copy">
            Route OpenAI-compatible requests to independent providers and settle
            every signed usage receipt in native MON.
          </p>
        </section>

        <RoutingRail state={routeState} />

        <div className="operational-grid">
          <section id="marketplace" className="marketplace" aria-labelledby="marketplace-title">
            <div className="section-heading">
              <div>
                <p className="eyebrow">Marketplace</p>
                <h2 id="marketplace-title">Models available now</h2>
              </div>
              <span className="data-label">Live data only</span>
            </div>
            <ModelList api={marketplace} />
          </section>

          <aside className="account-context" aria-labelledby="account-title">
            <p className="eyebrow">Account context</p>
            <h2 id="account-title">{session ? `${session.wallet_address.slice(0, 6)}…${session.wallet_address.slice(-4)}` : 'No wallet connected'}</h2>
            <p>
              {session ? 'Manage escrow, bounded spending, and provider machines from this wallet-bound account.' : 'Connect a Monad wallet to view escrow, create a bounded spending session, or register a provider machine.'}
            </p>
            <dl>
              <div><dt>Escrow</dt><dd>Unavailable</dd></div>
              <div><dt>Spend session</dt><dd>Not opened</dd></div>
              <div><dt>Provider machines</dt><dd>Unavailable</dd></div>
            </dl>
          </aside>
        </div>

        {session && <><div className="account-tools"><DeviceApproval api={api} /><ApiKeys api={api} /></div><Billing /><ProviderConsole /></>}

        <section id="activity" className="activity" aria-labelledby="activity-title">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Confirmed network record</p>
              <h2 id="activity-title">Recent activity</h2>
            </div>
          </div>
          <Activity api={marketplace} authApi={api} connected={Boolean(session)} onState={setRouteState} />
        </section>
      </main>

      <footer>
        <span>Myference / real inference, provable settlement</span>
        <a href="https://github.com/kunalshah017/myference">Source on GitHub</a>
      </footer>
    </div>
  )
}

function App() {
  return window.location.pathname === '/app' ? <OperationalApp /> : <LandingPage />
}

export { OperationalApp }
export default App
