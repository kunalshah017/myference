import type posthog from 'posthog-js/dist/module.no-external'

export type AnalyticsClient = Pick<typeof posthog, 'init' | 'capture'>
type AnalyticsEvents = {
  onboarding_role_selected: { role: 'consumer' | 'provider' }
  onboarding_skipped: { role: 'consumer' | 'provider' }
  onboarding_resumed: { role: 'consumer' | 'provider' }
  onboarding_completed: { role: 'consumer' | 'provider' }
  wallet_connected: { surface: 'onboarding' | 'dashboard' }
  dashboard_viewed: { view: string }
}
export type AnalyticsEvent = keyof AnalyticsEvents

const projectToken = 'phc_mbvusAFLkfcuDF2t6CZovQ9WrrtBmKxMkznLStSssWAf'
const browserHostname = () => typeof window === 'undefined' ? '' : window.location.hostname
const local = (hostname: string) => !hostname || hostname === 'localhost' || hostname === '127.0.0.1'
let activeClient: AnalyticsClient | undefined
const pending: Array<{ event: AnalyticsEvent; properties: Record<string, string> }> = []

function configure(client: AnalyticsClient) {
  try {
    client.init(projectToken, {
      api_host: 'https://t.myference.xyz',
      ui_host: 'https://us.posthog.com',
      defaults: '2026-05-30',
      autocapture: false,
      capture_pageview: true,
      capture_pageleave: true,
      capture_exceptions: false,
      disable_session_recording: true,
      person_profiles: 'identified_only',
    })
    activeClient = client
  } catch {
    return false
  }
  for (const item of pending.splice(0)) {
    try { client.capture(item.event, item.properties) } catch { /* analytics must not block the product */ }
  }
  return true
}

export function initializeAnalytics(client?: AnalyticsClient, hostname = browserHostname()) {
  if (local(hostname)) return false
  if (client) return configure(client)
  else void import('posthog-js/dist/module.no-external').then(({ default: loaded }) => configure(loaded)).catch(() => undefined)
  return true
}

export function captureEvent<Event extends AnalyticsEvent>(event: Event, properties: AnalyticsEvents[Event], client: AnalyticsClient | undefined = activeClient, hostname = browserHostname()) {
  if (local(hostname)) return false
  if (!client) {
    if (pending.length < 50) pending.push({ event, properties: properties as Record<string, string> })
    return true
  }
  try {
    client.capture(event, properties)
    return true
  } catch {
    return false
  }
}
