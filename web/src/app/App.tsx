import { DashboardShell } from './DashboardShell'
import DocsPage from './DocsPage'
import LandingPage from './LandingPage'
import { ProviderApproval } from '../features/provider/ProviderApproval'

function OperationalApp({ initialView = 'overview' }: { initialView?: 'overview' | 'api' | 'devices' | 'hosting' }) {
  return <DashboardShell initialView={initialView} />
}

function App() {
  if (window.location.pathname === '/docs') return <DocsPage />
  if (window.location.pathname === '/devices') return <OperationalApp initialView="devices" />
	if (window.location.pathname === '/host') return <OperationalApp initialView="hosting" />
	if (window.location.pathname === '/provider/approve') return <ProviderApproval actionID={new URLSearchParams(window.location.search).get('action') ?? ''} />
  return window.location.pathname === '/app' ? <OperationalApp /> : <LandingPage />
}

export { OperationalApp }
export default App
