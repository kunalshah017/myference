# Landing and theme integration

## Goal

Apply the supplied landing-page visual direction to the existing Myference web client without replacing its live marketplace, wallet, billing, provider, activity, or realtime flows.

## Decisions

- `/` is a public landing page with the supplied dark navy/lime visual language.
- `/app` is the existing operational client, preserving its current API-backed behavior.
- Navigation uses the browser URL and a small route switch; no routing dependency is added.
- The landing page uses local deterministic visuals and CSS animation. The reference's remote logo.dev assets, public waitlist destination, and claims of live network data are not copied.
- Existing operational styles remain available under the same tokens, with the landing page scoped by `.landing-page`.
- The temporary reference folder is removed after the reusable design values and component ideas are transferred.

## Landing sections

1. Header with Myference mark, Platform/How it works/Pricing anchors, and Launch app.
2. Hero explaining unused-machine providers, OpenAI-compatible access, and MON settlement.
3. Provider-network visual built from CSS nodes and routing lines.
4. Four-step demand, broker, supply, and settlement explanation.
5. Pricing/FAQ copy that describes market pricing without fabricated metrics.
6. Footer with a link to the operational app.

## Acceptance criteria

- Existing app tests continue to pass.
- Landing page and `/app` route render with no network-only visual dependencies.
- A landing-page test proves the main heading and app link are present.
- Production build succeeds.
- Temporary `web-only-landing-page-and-theme` directory is removed only after verification.
