import { DashboardShell } from './DashboardShell'
import LandingPage from './LandingPage'

function OperationalApp() {
  return <DashboardShell />
}

function App() {
  return window.location.pathname === '/app' ? <OperationalApp /> : <LandingPage />
}

export { OperationalApp }
export default App
