import { DashboardShell } from './DashboardShell'
import LandingPage from './LandingPage'

function OperationalApp({ initialView = 'overview' }: { initialView?: 'overview' | 'api' }) {
  return <DashboardShell initialView={initialView} />
}

function App() {
  if (window.location.pathname === '/devices') return <OperationalApp initialView="api" />
  return window.location.pathname === '/app' ? <OperationalApp /> : <LandingPage />
}

export { OperationalApp }
export default App
