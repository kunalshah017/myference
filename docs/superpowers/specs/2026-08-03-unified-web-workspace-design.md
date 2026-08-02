# Unified Web Workspace Design

## Objective

Replace the rejected dark landing theme with the existing Myference application language and turn `/app` into a shared workspace for accounts that consume and/or host inference.

## Visual system

The entire web client uses the current application tokens: paper white, node mist, circuit ink, relay violet, packet blue, proof mint, fault coral, square corners, fine rules, monospaced data labels, and an editorial grid. The landing page is a more spacious expression of the same system, not a separate brand theme.

The signature visual is a live-routing diagram drawn as an ordered system rail. It mirrors the product's actual request lifecycle and reuses the same rail vocabulary in the dashboard. Icons come from Lucide React and are paired with text labels; they never replace accessible names.

## Information architecture

`/` contains the landing page. `/app` contains one authenticated-capable workspace for both customer and provider roles.

The workspace has a persistent sidebar with these views:

- Overview: account/network state and direct links to customer and provider work.
- Models: live marketplace models, providers, capacities, and published prices.
- Playground: an on-site chat client calling `/v1/chat/completions` with a user-supplied API key.
- Funds: native MON escrow, withdrawals, claims, and bounded spending sessions.
- API access: API base URL, OpenAI and Anthropic-compatible endpoint examples, scoped key creation, and revocation.
- Usage: realtime and indexed account request state from the existing activity APIs.
- Hosting: registered machines, provider backends, offers, health, and controls already backed by the operations API/contract writer.
- Earnings: provider earnings, claimable MON, bonded collateral, and bond-exit state.

Any account can enter all views. Authentication-dependent panels show actionable connect-wallet states.

## Data boundaries

- Models and offers: `MarketplaceAPI.models()` and `MarketplaceAPI.model()`.
- Chat: authenticated `POST /v1/chat/completions`, with model, messages, and optional streaming disabled for the initial browser client.
- Wallet/device/key state: `AuthAPI`.
- Escrow, sessions, machines, offers, bond, and earnings: `OperationsAPI` plus the existing `ViemMarketWriter`.
- Realtime request state: authenticated event stream plus `MarketplaceAPI.activity()` reconciliation.

The UI does not fabricate totals, charts, slashing events, token usage, or revenue history that the server does not expose. Missing data is labeled unavailable. Current collateral and bond-exit state are shown; historical slashing requires a future indexed API before it can be rendered as a ledger.

## Component boundaries

- `DashboardShell` owns navigation, responsive sidebar state, wallet session, and active view.
- Existing marketplace, billing, API key, activity, and provider components keep their data behavior and are composed into views.
- `ChatPlayground` owns model selection, API key input, message state, request submission, and response/error rendering.
- `ApiAccessGuide` presents environment-derived base URLs and compatible endpoint examples.
- `DashboardOverview` summarizes only values supplied by account operations.
- Landing-page sections remain independent of dashboard data and reuse global design tokens.

## Error and empty states

Every request has loading, failure, disconnected, and empty states. Chat failures preserve the prompt and display the server message. API keys are held only in component memory and are never persisted by the browser. Unavailable metrics explicitly say why they are unavailable.

## Testing

- Route tests cover `/` and `/app`.
- Navigation tests cover switching customer and provider views.
- Chat tests cover authenticated request shape, successful completion, and failure.
- Existing marketplace, auth, billing, provider, and realtime tests remain green.
- Lint and production build must pass.
