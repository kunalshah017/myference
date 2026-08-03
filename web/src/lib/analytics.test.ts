import { describe, expect, it, vi } from 'vitest'
import { captureEvent, initializeAnalytics, type AnalyticsClient } from './analytics'

function client() {
  return { init: vi.fn(), capture: vi.fn() } as unknown as AnalyticsClient
}

describe('client analytics', () => {
  it('initializes privacy-first PostHog through the Myference proxy', () => {
    const analytics = client()

    expect(initializeAnalytics(analytics, 'myference.xyz')).toBe(true)
    expect(analytics.init).toHaveBeenCalledWith('phc_mbvusAFLkfcuDF2t6CZovQ9WrrtBmKxMkznLStSssWAf', {
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
  })

  it.each(['localhost', '127.0.0.1'])('does not send development traffic from %s', (hostname) => {
    const analytics = client()
    expect(initializeAnalytics(analytics, hostname)).toBe(false)
    expect(analytics.init).not.toHaveBeenCalled()
  })

  it('forwards only explicitly supplied product events', () => {
    const analytics = client()
    captureEvent('onboarding_role_selected', { role: 'consumer' }, analytics, 'myference.xyz')
    expect(analytics.capture).toHaveBeenCalledWith('onboarding_role_selected', { role: 'consumer' })
  })

  it('suppresses product events during local development', () => {
    const analytics = client()
    expect(captureEvent('dashboard_viewed', { view: 'models' }, analytics, 'localhost')).toBe(false)
    expect(analytics.capture).not.toHaveBeenCalled()
  })

  it('keeps an early interaction until the lazy SDK is ready', async () => {
    vi.resetModules()
    const fresh = await import('./analytics')
    const analytics = client()
    fresh.captureEvent('onboarding_role_selected', { role: 'provider' }, undefined, 'myference.xyz')
    fresh.initializeAnalytics(analytics, 'myference.xyz')
    expect(analytics.capture).toHaveBeenCalledWith('onboarding_role_selected', { role: 'provider' })
  })

  it('never lets SDK failures interrupt the product', async () => {
    vi.resetModules()
    const fresh = await import('./analytics')
    const brokenInit = { init: vi.fn(() => { throw new Error('storage blocked') }), capture: vi.fn() } as unknown as AnalyticsClient
    expect(fresh.initializeAnalytics(brokenInit, 'myference.xyz')).toBe(false)

    const brokenCapture = { init: vi.fn(), capture: vi.fn(() => { throw new Error('transport failed') }) } as unknown as AnalyticsClient
    expect(fresh.captureEvent('dashboard_viewed', { view: 'models' }, brokenCapture, 'myference.xyz')).toBe(false)
  })
})
