import { useEffect, useMemo, useState } from 'react'
import { Activity as ActivityIcon, Banknote, Bot, CircleDollarSign, Code2, Cpu, Home, KeyRound, Laptop, PanelLeft, Server, ShieldCheck } from 'lucide-react'
import { Activity } from '../features/activity/Activity'
import { ProviderAnalytics } from '../features/analytics/ProviderAnalytics'
import { UsageAnalytics } from '../features/analytics/UsageAnalytics'
import { ApiKeys } from '../features/auth/ApiKeys'
import { ApiAccessGuide } from '../features/auth/ApiAccessGuide'
import { ConnectWallet } from '../features/auth/ConnectWallet'
import { DeviceApproval } from '../features/auth/DeviceApproval'
import { Billing } from '../features/billing/Billing'
import { ModelList } from '../features/marketplace/ModelList'
import { ProviderConsole } from '../features/provider/ProviderConsole'
import { ChatPlayground } from '../features/playground/ChatPlayground'
import { AuthAPI, MarketplaceAPI, type Session } from '../lib/api'
import { captureEvent } from '../lib/analytics'
import { OnboardingFlow, OnboardingReminder } from '../features/onboarding/OnboardingFlow'
import type { OnboardingRole } from '../features/onboarding/onboarding'
import { DashboardOverview } from './DashboardOverview'

type DashboardView = 'overview' | 'models' | 'playground' | 'funds' | 'api' | 'usage' | 'devices' | 'hosting' | 'earnings'

const navigation: { view: DashboardView; label: string; group: 'use' | 'host' }[] = [
  { view: 'overview', label: 'Overview', group: 'use' },
  { view: 'models', label: 'Models', group: 'use' },
  { view: 'playground', label: 'Playground', group: 'use' },
  { view: 'funds', label: 'Funds', group: 'use' },
  { view: 'api', label: 'API keys', group: 'use' },
  { view: 'usage', label: 'Usage', group: 'use' },
	{ view: 'devices', label: 'Devices', group: 'host' },
	{ view: 'hosting', label: 'Provider account', group: 'host' },
  { view: 'earnings', label: 'Earnings & stake', group: 'host' },
]

const icons = { overview: Home, models: Cpu, playground: Bot, funds: CircleDollarSign, api: KeyRound, usage: ActivityIcon, devices: Laptop, hosting: Server, earnings: Banknote }

function DeviceAuthorizationPage({ api, connected }: { api: AuthAPI; connected: boolean }) {
  return <section className="device-workspace">
    <p className="eyebrow">Provider access</p>
    <h1>Authorize a provider device</h1>
    <p className="dashboard-intro">Enter the short code shown by the Myference CLI, verify the machine details, then approve its signer with your wallet.</p>
    <div className="device-authorization-grid">
      <div className="device-authorization-task">
        {connected ? <DeviceApproval api={api} /> : <div className="dashboard-empty"><strong>Wallet connection required</strong><p>Connect a wallet to review and approve this provider device.</p></div>}
      </div>
      <aside className="device-authorization-guide" aria-labelledby="device-guide-title">
        <ShieldCheck size={24} aria-hidden="true" />
        <p className="eyebrow">Secure pairing</p>
        <h2 id="device-guide-title">How it works</h2>
        <ol>
          <li><span>01</span><p><strong>Start the CLI</strong>Run <code>myference</code> on the device you want to host from.</p></li>
          <li><span>02</span><p><strong>Review the identity</strong>Match the machine name and signer shown here with the terminal.</p></li>
          <li><span>03</span><p><strong>Approve with your wallet</strong>The device receives its own revocable credential. Your wallet key stays in the browser.</p></li>
        </ol>
      </aside>
    </div>
  </section>
}

function storedRole(): OnboardingRole | undefined {
  const value = localStorage.getItem('myference:onboarding-role')
  return value === 'consumer' || value === 'provider' ? value : undefined
}

export function DashboardShell({ initialView = 'overview', authAPI }: { initialView?: DashboardView; authAPI?: AuthAPI }) {
  const api = useMemo(() => authAPI ?? new AuthAPI(), [authAPI])
  const marketplace = useMemo(() => new MarketplaceAPI(), [])
  const [session, setSession] = useState<Session>()
  const [sessionLoaded, setSessionLoaded] = useState(false)
  const [view, setView] = useState<DashboardView>(initialView)
  const [routeState, setRouteState] = useState('')
  const [role, setRole] = useState<OnboardingRole | undefined>(storedRole)
  const [skipped, setSkipped] = useState(() => localStorage.getItem('myference:onboarding-skipped') === 'true' || initialView !== 'overview')
  const [onboardingOpen, setOnboardingOpen] = useState(() => localStorage.getItem('myference:onboarding-skipped') !== 'true' && initialView === 'overview')
  const [completed, setCompleted] = useState(false)
  useEffect(() => { void api.session().then(setSession).catch(() => undefined).finally(() => setSessionLoaded(true)) }, [api])

  const changeRole = (next: OnboardingRole) => { setRole(next); setCompleted(false); localStorage.setItem('myference:onboarding-role', next) }
  const navigate = (next: DashboardView) => { setView(next); captureEvent('dashboard_viewed', { view: next }) }
  const resumeOnboarding = () => { captureEvent('onboarding_resumed', { role: role ?? 'consumer' }); setOnboardingOpen(true) }
  const skipOnboarding = () => {
    const selected = role ?? 'consumer'
    changeRole(selected)
    captureEvent('onboarding_skipped', { role: selected })
    localStorage.setItem('myference:onboarding-skipped', 'true')
    setSkipped(true)
    setOnboardingOpen(false)
  }
  const markOnboardingComplete = (finishedRole: OnboardingRole) => {
    changeRole(finishedRole)
    setCompleted(true)
    setSkipped(true)
    localStorage.setItem('myference:onboarding-skipped', 'true')
  }
  const finishOnboarding = (finishedRole: OnboardingRole) => { markOnboardingComplete(finishedRole); setOnboardingOpen(false) }

  if (!sessionLoaded) return <div className="onboarding-screen"><div className="dashboard-empty" role="status"><strong>Loading Myference…</strong></div></div>
  if (onboardingOpen) return <OnboardingFlow session={session} initialRole={role} authAPI={api} onConnected={setSession} onSkip={skipOnboarding} onComplete={markOnboardingComplete} onRoleChange={changeRole} onSessionExpired={() => setSession(undefined)} />

  const disconnected = (message: string) => <div className="dashboard-empty"><strong>Wallet connection required</strong><p>{message}</p></div>
  return <div className="dashboard-shell">
    <aside className="dashboard-sidebar">
      <a className="wordmark" href="/" aria-label="Myference home"><span aria-hidden="true">M/</span> Myference</a>
      <div className="dashboard-nav-group"><p>Use inference</p>{navigation.filter((item) => item.group === 'use').map((item) => { const Icon = icons[item.view]; return <button type="button" key={item.view} aria-current={view === item.view ? 'page' : undefined} onClick={() => navigate(item.view)}><Icon size={17} aria-hidden="true" />{item.label}</button> })}</div>
      <div className="dashboard-nav-group"><p>Provide inference</p>{navigation.filter((item) => item.group === 'host').map((item) => { const Icon = icons[item.view]; return <button type="button" key={item.view} aria-current={view === item.view ? 'page' : undefined} onClick={() => navigate(item.view)}><Icon size={17} aria-hidden="true" />{item.label}</button> })}</div>
      <a className="dashboard-docs" href="/docs"><Code2 size={16} aria-hidden="true" /> Documentation</a>
    </aside>
    <div className="dashboard-main">
      <header className="dashboard-topbar"><div><PanelLeft size={17} aria-hidden="true" /><span className="state-mark" aria-hidden="true" />{session ? 'Monad testnet connected' : 'Network not connected'}</div><ConnectWallet api={api} session={session} analyticsSurface="dashboard" onConnected={setSession} onDisconnected={() => setSession(undefined)} /></header>
      <main className="dashboard-workspace">
        {view === 'overview' && skipped && !completed && <OnboardingReminder role={role ?? 'consumer'} session={session} onContinue={resumeOnboarding} onSwitch={() => changeRole((role ?? 'consumer') === 'consumer' ? 'provider' : 'consumer')} onComplete={finishOnboarding} onSessionExpired={() => setSession(undefined)} />}
        {view === 'overview' && (session ? <DashboardOverview onNavigate={navigate} /> : <section><p className="eyebrow">Account workspace</p><h1>Workspace overview</h1><p className="dashboard-intro">Use hosted models and offer inference from the same wallet-bound account.</p>{disconnected('Connect a wallet to load balances, sessions, machines, and earnings.')}</section>)}
        {view === 'models' && <section><p className="eyebrow">Public inference market</p><h1>Models and prices</h1><ModelList api={marketplace} /></section>}
        {view === 'playground' && <section><p className="eyebrow">Browser test client</p><h1>Model playground</h1><p className="dashboard-intro">Send a real request through the OpenAI-compatible endpoint.</p><ChatPlayground /></section>}
        {view === 'funds' && (session ? <Billing /> : disconnected('Connect a wallet to deposit MON and open bounded spending sessions.'))}
        {view === 'api' && <><ApiAccessGuide />{session ? <ApiKeys api={api} marketplaceApi={marketplace} /> : disconnected('Connect a wallet to create and revoke API keys.')}</>}
        {view === 'usage' && <section><p className="eyebrow">Requests and settlement</p><h1>Usage</h1>{session ? <><UsageAnalytics /><section className="embedded-activity"><p className="eyebrow">Realtime request state</p><h2>In-flight activity</h2><Activity api={marketplace} authApi={api} connected onState={setRouteState} /></section></> : disconnected('Connect a wallet to inspect confirmed tokens, cost, and request activity.')}{routeState && <p className="route-state">Latest route state: {routeState}</p>}</section>}
		{view === 'devices' && <DeviceAuthorizationPage api={api} connected={Boolean(session)} />}
		{view === 'hosting' && <section><p className="eyebrow">Provider account</p><h1>Collateral and offer pricing</h1><p className="dashboard-intro">Use the Myference terminal UI to discover models, create offers, and run hosting. This page manages collateral and reprices existing offers.</p>{session ? <><ProviderConsole /><section className="embedded-activity"><p className="eyebrow">Realtime provider traffic</p><h2>Inference requests</h2><Activity api={marketplace} authApi={api} connected onState={setRouteState} /></section></> : disconnected('Connect a wallet to manage provider collateral and existing offer prices.')}</section>}
        {view === 'earnings' && <section><p className="eyebrow">Provider settlement</p><h1>Earnings and stake</h1>{session ? <><ProviderAnalytics /><ProviderConsole /></> : disconnected('Connect a wallet to inspect earnings, collateral, slashing, and bond-exit state.')}</section>}
      </main>
    </div>
  </div>
}

export type { DashboardView }
