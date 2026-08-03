# First-time onboarding design

## Goal

Help a new account complete one real Myference outcome without learning the dashboard first: either route a live inference as a customer or activate a provider offer from a machine. The same wallet may complete both paths.

## Entry and persistence

- `/app` opens a focused role choice for a browser that has not selected or skipped onboarding.
- “Use AI inference” is primary. “Host your AI inference” starts the provider path.
- “Skip for now” opens the dashboard. A compact overview reminder remains until the selected path reaches its real completion condition.
- Browser storage remembers only the selected path and whether the full-screen introduction was skipped. It never claims a blockchain or server operation succeeded.
- Progress is rebuilt from authenticated API, indexed contract, marketplace, and analytics state after refresh or on another device.
- A user may change paths at any time. Completing one path does not prevent completing the other.

## Customer path

1. **Connect account** — use the existing Monad Testnet challenge/signature login.
2. **Choose inference** — load only live models, rank by the sum of their advertised minimum input, output, and compute rates, and preselect the cheapest. The user may select another model.
3. **Fund requests** — show a recommended MON amount calculated from the selected model's published rates for a bounded starter request (2,000 input tokens, 1,000 output tokens, 120 compute seconds) plus a 20% buffer. Deposit through the deployed market contract. A positive indexed customer balance completes the step.
4. **Set a limit** — open a 24-hour spending session through the market contract. A non-finalized, unexpired session with remaining allowance completes the step.
5. **Create access** — create a key scoped to the selected model and both compatible chat endpoints. The secret is shown once and retained only in component memory for the final test. If an existing key is detected after refresh, explain that its secret cannot be recovered and offer a replacement key.
6. **Run inference** — send a real streaming request to `/v1/chat/completions`. A non-empty provider response completes onboarding immediately; settled analytics preserves completion across devices once indexed.

There is no global minimum customer balance. The recommended deposit is guidance; the actual request remains bounded by escrow balance, active session allowance, key maximum, and the per-request maximum-spend header.

## Provider path

1. **Connect account** — same wallet login as the customer path.
2. **Connect a machine** — show the platform-specific one-command installer and `myference host`. Completion requires a real machine and backend returned by account operations.
3. **Approve the signer** — embed the existing device-code review, on-chain signer authorization, and server approval flow.
4. **Bond collateral** — use the existing provider collateral contract action. Completion requires a positive indexed provider bond.
5. **Publish and activate** — use backends discovered by the CLI, not manually entered model claims. Publish an immutable on-chain price version and show the exact CLI version-sync command.
6. **Go live** — completion requires a non-revoked machine with an enabled, healthy backend linked to a published offer.

The CLI remains the source of model evidence: Ollama digests, upstream model identifiers, or pinned runtime images. The website never invents a model or machine.

## State model

`deriveConsumerProgress` and `deriveProviderProgress` are pure functions. They turn real API records into ordered steps and the next actionable step.

- Consumer completion: `analytics.customer.settled_requests > 0`, with an in-memory success override only for the response that just completed while indexing catches up.
- Provider completion: a healthy, enabled backend with an offer identity plus a matching published offer.
- Empty, loading, unauthenticated, and error states are explicit. HTTP 401 returns the user to the wallet step.
- Transaction rejection leaves the current step active and displays the wallet/contract error.
- Finalized-but-unindexed transactions show a waiting state and retry operations data.

## Interface direction

The onboarding inherits the dashboard's paper, violet relay, grid, sharp borders, display face, and monospace data labels. Its signature is a left-hand “route map”: a real ordered packet path whose completed nodes fill mint and whose active node fills violet. The content column stays quiet and focused on one action. On mobile, the route map becomes a horizontally scrollable progress strip.

Copy is short, operational, and action-led. Every amount is shown as MON, with the existing live USD reference where available; raw wei remains available only in technical disclosures.

## Recovery and accessibility

- No live model: explain that inventory is empty and keep refresh available; do not fabricate a selection.
- Lost key: create a replacement; never retrieve a stored secret.
- Expired session: return to “Set a limit.”
- Failed inference: preserve model, key, maximum spend, and prompt for retry.
- Provider daemon offline: show the CLI status/start commands and keep “Go live” incomplete.
- All controls have labels, semantic headings, live status/error roles, keyboard focus, reduced-motion-safe styling, and responsive layouts.

## Verification

- Unit-test ranking, recommended spend, and state derivation including expired/finalized sessions and unhealthy providers.
- Component-test role selection, skipping, persistent reminder, wallet handoff, key-to-playground handoff, and no-inventory recovery.
- Run the complete web test suite, lint, TypeScript/Vite build, then inspect `/app` at desktop and mobile widths.
