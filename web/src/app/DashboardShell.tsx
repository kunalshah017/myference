import { useEffect, useMemo, useState } from 'react'
import { Activity as ActivityIcon, Banknote, Bot, CircleDollarSign, Code2, Cpu, Home, KeyRound, PanelLeft, Server } from 'lucide-react'
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

type DashboardView = 'overview' | 'models' | 'playground' | 'funds' | 'api' | 'usage' | 'hosting' | 'earnings'

const navigation: { view: DashboardView; label: string; group: 'use' | 'host' }[] = [
  { view: 'overview', label: 'Overview', group: 'use' },
  { view: 'models', label: 'Models', group: 'use' },
  { view: 'playground', label: 'Playground', group: 'use' },
  { view: 'funds', label: 'Funds', group: 'use' },
  { view: 'api', label: 'API access', group: 'use' },
  { view: 'usage', label: 'Usage', group: 'use' },
  { view: 'hosting', label: 'Host inference', group: 'host' },
  { view: 'earnings', label: 'Earnings & stake', group: 'host' },
]

const icons = { overview: Home, models: Cpu, playground: Bot, funds: CircleDollarSign, api: KeyRound, usage: ActivityIcon, hosting: Server, earnings: Banknote }

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
        {skipped && !completed && <OnboardingReminder role={role ?? 'consumer'} session={session} onContinue={resumeOnboarding} onSwitch={() => changeRole((role ?? 'consumer') === 'consumer' ? 'provider' : 'consumer')} onComplete={finishOnboarding} onSessionExpired={() => setSession(undefined)} />}
        {view === 'overview' && (session ? <DashboardOverview onNavigate={navigate} /> : <section><p className="eyebrow">Account workspace</p><h1>Workspace overview</h1><p className="dashboard-intro">Use hosted models and offer inference from the same wallet-bound account.</p>{disconnected('Connect a wallet to load balances, sessions, machines, and earnings.')}</section>)}
        {view === 'models' && <section><p className="eyebrow">Public inference market</p><h1>Models and prices</h1><ModelList api={marketplace} /></section>}
        {view === 'playground' && <section><p className="eyebrow">Browser test client</p><h1>Model playground</h1><p className="dashboard-intro">Send a real request through the OpenAI-compatible endpoint.</p><ChatPlayground /></section>}
        {view === 'funds' && (session ? <Billing /> : disconnected('Connect a wallet to deposit MON and open bounded spending sessions.'))}
        {view === 'api' && <><ApiAccessGuide />{session ? <><ApiKeys api={api} marketplaceApi={marketplace} /><DeviceApproval api={api} /></> : disconnected('Connect a wallet to create API keys and approve provider devices.')}</>}
        {view === 'usage' && <section><p className="eyebrow">Requests and settlement</p><h1>Usage</h1>{session ? <><UsageAnalytics /><section className="embedded-activity"><p className="eyebrow">Realtime request state</p><h2>In-flight activity</h2><Activity api={marketplace} authApi={api} connected onState={setRouteState} /></section></> : disconnected('Connect a wallet to inspect confirmed tokens, cost, and request activity.')}{routeState && <p className="route-state">Latest route state: {routeState}</p>}</section>}
        {view === 'hosting' && <section><p className="eyebrow">Provider workspace</p><h1>Host inference</h1><p className="dashboard-intro">Manage local models, cloud APIs, and CLI agents; follow accepted requests through settlement in realtime.</p>{session ? <><ProviderConsole /><section className="embedded-activity"><p className="eyebrow">Realtime provider traffic</p><h2>Inference requests</h2><Activity api={marketplace} authApi={api} connected onState={setRouteState} /></section></> : disconnected('Connect a wallet to manage provider machines, backends, and offers.')}</section>}
        {view === 'earnings' && <section><p className="eyebrow">Provider settlement</p><h1>Earnings and stake</h1>{session ? <><ProviderAnalytics /><ProviderConsole /></> : disconnected('Connect a wallet to inspect earnings, collateral, slashing, and bond-exit state.')}</section>}
      </main>
    </div>
  </div>
}

export type { DashboardView }
