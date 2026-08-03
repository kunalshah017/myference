import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import App from './app/App.tsx'
import { initializeAnalytics } from './lib/analytics.ts'
import './styles/global.css'
import './styles/landing.css'
import './styles/docs.css'

const queryClient = new QueryClient()
initializeAnalytics()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}><App /></QueryClientProvider>
  </StrictMode>,
)
