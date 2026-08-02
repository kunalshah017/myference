# Myference Web Client Design

**Date:** 2026-08-02
**Status:** Approved by direct implementation instruction

## 1. Purpose

The Myference web client is the public marketplace and account console for developers buying inference and providers selling machine capacity. Its primary public-page job is to answer from live data: which model can I call now, through how many providers, at what all-inclusive MON price?

The client never invents marketplace activity. Models, provider availability, prices, balances, requests, receipts, and transactions appear only after authenticated APIs or confirmed Monad events return them.

## 2. Users and outcomes

### Developer

- Connect an account and Monad wallet.
- Deposit native MON and open a bounded spending session.
- Discover available model capabilities and current provider liquidity.
- Create scoped API keys.
- Copy OpenAI- or Anthropic-compatible setup instructions.
- Monitor requests, charges, receipts, and settlement transactions.
- Close spending sessions and withdraw unlocked MON.

### Provider

- Complete device authorization initiated by `myference login`.
- See registered machines and realtime online state.
- Inspect enabled backends, models, capacity, health, and quarantine state.
- Deposit or withdraw provider collateral.
- Publish and version model prices.
- Monitor accepted jobs, usage receipts, pending earnings, settled earnings, and slashing evidence.

## 3. Information architecture

```text
Public
|- /                         Marketplace overview and live routing rail
|- /models                   Searchable live model catalogue
|- /models/:modelId          Capability, price, liquidity, provider choices
|- /docs                     API connection instructions
`- /device                   Provider device authorization

Authenticated console
|- /console                  Balance, spend, requests, and settlement summary
|- /console/api-keys         Create, scope, reveal-once, and revoke API keys
|- /console/billing          Deposits, sessions, withdrawals, and transactions
|- /console/activity         Requests and receipt lifecycle
|- /provider                 Machines, backends, offers, collateral, earnings
`- /settings                 Wallet, account, security, and session revocation
```

One responsive application serves both roles. Navigation changes according to verified account capabilities; it does not maintain separate duplicated dashboards.

## 4. Required user flows

### Buy inference

1. Connect wallet and authenticate the account.
2. Deposit native MON through the deployed Myference contract.
3. Open a spending session with explicit allowance and expiry.
4. Create a scoped API key bound to that session.
5. Select a live model and copy the endpoint/model identifier.
6. Submit requests from the customer's application.
7. Observe request, receipt, and confirmed settlement events in realtime.

### Start a provider

1. Run `myference login` in the provider CLI.
2. Enter the device code at `/device` and approve the exact machine.
3. Deposit the required native-MON bond.
4. Start a real backend from the CLI.
5. Publish an offer and price version.
6. Observe the machine become routable after confirmed chain and health state.

### Investigate a charge

1. Open an activity record.
2. Compare model, provider, price version, fee version, token/compute usage, and maximum spend.
3. Follow the receipt status to the Monad transaction.
4. Open the transaction in the official explorer.

Prompt and output content is never displayed because Myference does not persist it.

## 5. Framework and folders

```text
web/
|- src/
|  |- app/                    Router, providers, and global boundaries
|  |- components/             Reusable accessible interface components
|  |- features/
|  |  |- auth/                Wallet/account/device authorization
|  |  |- marketplace/         Models, offers, providers, routing liquidity
|  |  |- billing/             Deposits, sessions, withdrawals
|  |  |- activity/            Requests, receipts, settlement lifecycle
|  |  `- provider/            Machines, backends, offers, collateral, earnings
|  |- lib/                    API, chain, realtime, formatting, validation
|  |- styles/                 Tokens and global styles
|  `- test/                   Test setup and real-boundary fixtures
|- public/                    Static public assets only
|- index.html
|- package.json
|- tsconfig.json
`- vite.config.ts
```

Use React with TypeScript and Vite. A small typed route map built on the browser History API owns the fixed application routes; React Router is excluded because its current releases have overlapping high-severity advisories and the application does not need its server features. TanStack Query owns remote server state. Viem provides typed Monad reads, simulations, signatures, and transactions. Wagmi may wrap wallet connections if it removes connector lifecycle code without obscuring chain errors. Native `EventSource` handles authenticated realtime through a short-lived stream ticket; no global state library is introduced until local and query state are insufficient.

## 6. Data boundaries

The web client reads operational data from the Go control-plane API and economic data from both indexed API records and direct Monad contract reads. Before a write, it simulates the contract call; after wallet submission, it displays pending state until the configured finality policy confirms the transaction and the indexer observes it.

The client does not optimistically invent final balances. A pending deposit, bond, offer, session, settlement, or withdrawal remains visibly pending with its transaction hash.

Realtime event types are versioned and include:

- Machine presence and capacity
- Offer published or superseded
- Request state transition
- Receipt proposed, signed, submitted, confirmed, or rejected
- Deposit, bond, session, withdrawal, fee, or slashing event

Reconnect uses the last event ID. Gaps trigger an authoritative refetch before new events apply.

## 7. Visual direction

The interface is a network operations ledger, not a generic token dashboard.

### Palette

- `Circuit Ink` `#171522`: primary text and dark surfaces
- `Relay Violet` `#6E5AE6`: Monad/Myference action and active-route color
- `Packet Blue` `#2F6FED`: informational and streaming state
- `Proof Mint` `#28A982`: confirmed receipt and settlement state
- `Fault Coral` `#D95D67`: errors, rejected receipts, and quarantine
- `Node Mist` `#F3F2F8`: cool application background

All semantic combinations must pass WCAG 2.2 AA contrast. Color is never the only state signal.

### Type

- Display: `Unbounded`, used sparingly for the Myference wordmark and major marketplace statement
- Body/UI: `Manrope`, optimized for dense readable controls
- Data: `Fragment Mono`, used for model IDs, MON amounts, hashes, latency, and token counts

Fonts are self-hosted and subset before production. System fallbacks keep the first scaffold functional without a third-party font request.

### Layout

```text
+------------------------------------------------------------------+
| MYFERENCE      Models  Docs                  Network  Connect     |
+------------------------------------------------------------------+
| Live routing rail: ESCROW -> ROUTER -> PROVIDER -> SETTLEMENT     |
+--------------------------------------+---------------------------+
| Models available now                 | Account / network context |
| filter, capability, price, liquidity | exact next required action|
+--------------------------------------+---------------------------+
| Recent confirmed network activity                                |
+------------------------------------------------------------------+
```

Desktop uses a 12-column operational grid. Mobile collapses to one task-focused column with the account action first. Cards are reserved for bounded resources such as a model offer or machine; page structure uses dividers and aligned data columns rather than nested card grids.

### Signature element

The routing rail shows the true state of a selected or active request across escrow reservation, broker routing, provider execution, receipt signing, and Monad settlement. It is static and explicit when no live request exists. Motion occurs only on actual state transitions and is disabled under `prefers-reduced-motion`.

## 8. Content rules

- Use all-inclusive MON prices and always disclose the Myference fee version.
- Show full model identifiers and capability limits before API-key creation.
- Label unconfirmed chain state as pending, never available.
- Empty marketplace: “No providers are serving this model right now.”
- Disconnected realtime: “Live updates disconnected. Reconnecting; displayed data may be stale.”
- Insufficient escrow: state the required and available MON amounts.
- API keys are revealed once and never reconstructed in the UI.
- Destructive actions name their exact scope: “Revoke key,” “Stop routing machine,” or “Close spending session.”

## 9. Security and privacy

- Wallet signatures include domain, chain ID, origin, nonce, issued time, and expiry.
- The client refuses economic writes on an unsupported chain.
- API tokens live in memory or secure same-site HTTP-only sessions, not local storage.
- Device codes are short-lived, one-time, rate-limited, and display the approving machine identity.
- Every contract write is simulated and decoded errors are shown before signing.
- User-controlled model/provider labels are rendered as text, never HTML.
- External explorer links use fixed chain configuration and safe new-tab attributes.
- No prompts, outputs, private keys, backend secrets, or bearer tokens enter analytics or error reporting.

## 10. Accessibility and resilience

- Semantic landmarks, headings, tables, forms, and status messages
- Full keyboard operation and visible focus
- `aria-live` only for meaningful request/transaction transitions
- Reduced-motion support
- Responsive layouts down to 320 CSS pixels
- Loading, empty, stale, disconnected, unauthorized, wrong-network, rejected-wallet, and contract-revert states
- Amount entry parses decimal MON into integer wei without floating point
- Times show local display plus exact UTC detail

## 11. Verification

- Unit tests for wei formatting/parsing, price presentation, state reducers, and event-gap handling
- Component tests for loading, empty, wrong-network, permission, and error states
- Accessibility checks for every route
- Browser tests against a real local Go server, PostgreSQL, Anvil contract, and wallet automation
- A Monad testnet browser run that deposits MON, opens a session, creates an API key, observes real inference, follows settlement, and verifies explorer-visible balances
- Production build with no hard-coded model, provider, balance, transaction, or availability data

The web client is release-ready only when its displayed economic state matches the deployed contract and its request lifecycle matches the broker database after a real end-to-end inference.
