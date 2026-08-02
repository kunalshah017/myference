const routeStages = [
  { label: 'Escrow', detail: 'Awaiting balance' },
  { label: 'Router', detail: 'Awaiting request' },
  { label: 'Provider', detail: 'Awaiting capacity' },
  { label: 'Settlement', detail: 'Awaiting receipt' },
]

function App() {
  const api = useMemo(() => new AuthAPI(), [])
  const [session, setSession] = useState<Session>()
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
            Network not connected
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

        <section className="routing-rail" aria-labelledby="routing-title">
          <div className="rail-heading">
            <p className="eyebrow" id="routing-title">Live routing rail</p>
            <p>Awaiting a live request</p>
          </div>
          <ol>
            {routeStages.map((stage, index) => (
              <li key={stage.label} data-state="idle">
                <span className="stage-index" aria-hidden="true">
                  {String(index + 1).padStart(2, '0')}
                </span>
                <strong>{stage.label}</strong>
                <span>{stage.detail}</span>
              </li>
            ))}
          </ol>
        </section>

        <div className="operational-grid">
          <section id="marketplace" className="marketplace" aria-labelledby="marketplace-title">
            <div className="section-heading">
              <div>
                <p className="eyebrow">Marketplace</p>
                <h2 id="marketplace-title">Models available now</h2>
              </div>
              <span className="data-label">Live data only</span>
            </div>
            <div className="empty-state" role="status">
              <span className="empty-glyph" aria-hidden="true">///</span>
              <div>
                <h3>Live marketplace data is not connected.</h3>
                <p>
                  Models and prices will appear after the Myference broker returns
                  verified provider capacity.
                </p>
              </div>
            </div>
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

        {session && <div className="account-tools"><DeviceApproval api={api} /><ApiKeys api={api} /></div>}

        <section id="activity" className="activity" aria-labelledby="activity-title">
          <div className="section-heading">
            <div>
              <p className="eyebrow">Confirmed network record</p>
              <h2 id="activity-title">Recent activity</h2>
            </div>
          </div>
          <p className="activity-empty">
            Request and settlement events will appear after a live connection is established.
          </p>
        </section>
      </main>

      <footer>
        <span>Myference / real inference, provable settlement</span>
        <a href="https://github.com/kunalshah017/myference">Source on GitHub</a>
      </footer>
    </div>
  )
}

export default App
import { useEffect, useMemo, useState } from 'react'
import { ApiKeys } from '../features/auth/ApiKeys'
import { ConnectWallet } from '../features/auth/ConnectWallet'
import { DeviceApproval } from '../features/auth/DeviceApproval'
import { AuthAPI, type Session } from '../lib/api'
