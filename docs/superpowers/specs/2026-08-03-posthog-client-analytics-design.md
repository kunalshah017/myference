# PostHog Client Analytics Design

## Goal

Measure acquisition and activation across the public site and dashboard through the first-party PostHog proxy at `https://t.myference.xyz` without collecting inference prompts, API keys, wallet addresses, signatures, or transaction payloads.

## Approaches considered

1. **Privacy-first explicit events (selected).** Use `posthog-js`, automatic pageview/pageleave events, and a small typed capture helper for meaningful product actions. Disable DOM autocapture and session recording. This produces stable funnels with the smallest sensitive-data surface.
2. **Default autocapture.** Faster to instrument, but button labels and changing DOM structure create noisy events and unnecessary exposure around wallet and API-key screens.
3. **Pageviews only.** Smallest integration, but it cannot measure onboarding role choice, skips, resumes, or completion.

## Architecture

`web/src/lib/analytics.ts` owns initialization and event capture. It initializes only in a real browser, only outside localhost/test environments, and uses the public project token with `api_host` set to the managed proxy and `ui_host` set to US Cloud. Callers send only enumerated event names and low-cardinality properties.

The application initializes analytics once before React renders. Existing callbacks in `DashboardShell` and `OnboardingFlow` record the onboarding funnel, wallet connection state, and dashboard navigation. No component receives the PostHog client directly.

## Events

- `$pageview` and `$pageleave` through PostHog's built-in page tracking.
- `onboarding_role_selected` with `role`.
- `onboarding_skipped` with `role`.
- `onboarding_resumed` with `role`.
- `onboarding_completed` with `role`.
- `wallet_connected` with `surface`, never an address or account ID.
- `dashboard_viewed` with the dashboard `view`.

## Privacy and failure behavior

DOM autocapture and session recording remain disabled. The integration does not identify users because the available identifiers are wallet/account identifiers and are unnecessary for the initial aggregate funnel. Ad blockers, offline use, or proxy failure must never interrupt product behavior; PostHog remains an observational side effect.

## Verification

Unit tests verify production initialization settings, local/test suppression, and safe event forwarding. Component tests verify funnel events through existing user interactions. The full web tests, lint, and production build must pass, followed by a deployed proxy request check.
