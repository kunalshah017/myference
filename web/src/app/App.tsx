import { DashboardShell } from './DashboardShell'
import DocsPage from './DocsPage'
import LandingPage from './LandingPage'

function OperationalApp({ initialView = 'overview' }: { initialView?: 'overview' | 'api' | 'hosting' }) {
  return <DashboardShell initialView={initialView} />
}

function App() {
  if (window.location.pathname === '/docs') return <DocsPage />
  if (window.location.pathname === '/devices') return <OperationalApp initialView="api" />
	if (window.location.pathname === '/host') return <OperationalApp initialView="hosting" />
  return window.location.pathname === '/app' ? <OperationalApp /> : <LandingPage />
}

export { OperationalApp }
export default App
