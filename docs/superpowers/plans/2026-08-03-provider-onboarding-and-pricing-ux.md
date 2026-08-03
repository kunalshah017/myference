# Provider Onboarding and Pricing UX Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace error-prone wei/model entry with CLI discovery, USD-first pricing, honest model evidence, readable MON displays, a live-model playground selector, and a password-manager-safe API-key field.

**Architecture:** Extend existing protocol capacity records with runtime evidence produced by backend discovery and persisted in routing state. Add one cached public quote handler that is informational only. Reuse integer `bigint`/`viem` conversions in a shared web money module, then simplify existing components without introducing a new component framework.

**Tech Stack:** Go, PostgreSQL, React, TypeScript, TanStack Query, viem, Vitest, Monad native MON.

---

### Task 1: Public MON/USD reference quote

**Files:**
- Create: `server/internal/api/price.go`
- Create: `server/internal/api/price_test.go`
- Modify: `server/cmd/myference-server/main.go`
- Modify: `server/cmd/myference-server/main_test.go`

- [ ] Write a failing handler test expecting `GET /api/reference-price` to return `{symbol:"MON", usd_per_mon:"0.0209", source:"CoinGecko", updated_at:...}` and to reuse the cached quote when the upstream fails.
- [ ] Run `go test ./server/internal/api -run ReferencePrice -count=1` and confirm the handler is missing.
- [ ] Implement a bounded HTTP fetcher for CoinGecko's `simple/price` endpoint, parse the decimal with `math/big`, cache successful quotes for 60 seconds, reject quotes older than 15 minutes, and return `503` when no usable quote exists.
- [ ] Register the handler at `/api/reference-price` without changing settlement code.
- [ ] Run the focused test and commit `feat: expose cached MON reference price`.

### Task 2: Shared MON and USD presentation

**Files:**
- Modify: `web/src/lib/amount.ts`
- Create: `web/src/lib/amount.test.ts`
- Modify: `web/src/lib/api.ts`
- Create: `web/src/components/Money.tsx`
- Modify: `web/src/features/billing/Billing.tsx`
- Modify: `web/src/features/billing/SpendingSession.tsx`
- Modify: `web/src/features/provider/Earnings.tsx`
- Modify: `web/src/features/provider/ProviderConsole.tsx`
- Modify: `web/src/features/analytics/UsageAnalytics.tsx`
- Modify: `web/src/features/analytics/ProviderAnalytics.tsx`
- Modify: `web/src/features/marketplace/ModelList.tsx`
- Modify: `web/src/features/marketplace/ModelDetail.tsx`

- [ ] Write failing tests for compact MON formatting, exact decimal conversion, USD-to-wei conversion with integer rounding, and component output that excludes visible `wei`.
- [ ] Run the amount and existing feature tests and confirm the new expectations fail.
- [ ] Add `formatMON`, `usdToWei`, and `weiToUSD` using decimal strings and `bigint`; never use floating-point values for transaction input.
- [ ] Add a small `ReferencePriceAPI` and query hook/component that renders MON plus an optional timestamped USD estimate.
- [ ] Replace visible wei values across account, marketplace, analytics, earnings, and sessions; keep raw wei in `<details>` where useful.
- [ ] Run focused tests and commit `feat: present marketplace money in MON and USD`.

### Task 3: Simplified offers and activation-ready machine selection

**Files:**
- Modify: `web/src/features/provider/Offers.tsx`
- Modify: `web/src/features/provider/ProviderConsole.tsx`
- Modify: `web/src/features/provider/provider.test.tsx`
- Modify: `web/src/styles/global.css`

- [ ] Write a failing component test that selects a discovered backend, derives offer/model identifiers, accepts USD rates, previews MON, and submits integer wei rates.
- [ ] Run `npm test -- --run src/features/provider/provider.test.tsx` and confirm the old manual form fails it.
- [ ] Replace offer-name/model text inputs with a backend select populated from `operations.machines[].backends`.
- [ ] Accept USD per 1M input/output and USD per compute minute; convert to on-chain rates using the current quote and clearly show the locked MON result.
- [ ] For `codex`, `claude`, and `kimi`, disable token-rate inputs and publish compute-only rates.
- [ ] Keep an advanced MON fallback when the quote is unavailable.
- [ ] Run the provider tests and commit `feat: simplify provider offer activation`.

### Task 4: Live-model playground and safe secret entry

**Files:**
- Modify: `web/src/features/playground/ChatPlayground.tsx`
- Modify: `web/src/features/playground/ChatPlayground.test.tsx`
- Modify: `web/src/styles/global.css`

- [ ] Write failing tests for a live inventory select, exclusion of stale/unavailable models, MON maximum-spend conversion, and an API-key field whose DOM type is not `password` and whose autocomplete/password-manager attributes opt out.
- [ ] Run the playground test and confirm the old text/password fields fail.
- [ ] Query `MarketplaceAPI.models()`, render a native `<select>`, convert maximum MON to wei before `InferenceAPI.chat`, and render loading/empty/error states.
- [ ] Use a text input with CSS masking, a show/hide button, `autocomplete="off"`, `data-1p-ignore`, and `data-lpignore`; keep the value only in component state.
- [ ] Run the focused test and commit `feat: streamline playground request setup`.

### Task 5: Runtime model evidence

**Files:**
- Modify: `protocol/v1/messages.go`
- Modify: `protocol/v1/messages_test.go`
- Modify: `cli/internal/backend/backend.go`
- Modify: `cli/internal/backend/ollama/ollama.go`
- Modify: `cli/internal/backend/openai/openai.go`
- Modify: `cli/cmd/myference/main.go`
- Modify: `cli/cmd/myference/main_test.go`
- Create: `migrations/000015_model_evidence.sql`
- Modify: `server/internal/store/inference.go`
- Modify: `server/internal/store/marketplace.go`
- Modify: `server/internal/store/operations.go`
- Modify: corresponding store/API integration tests
- Modify: `web/src/lib/api.ts`
- Modify: `web/src/features/provider/Machines.tsx`
- Modify: `web/src/features/marketplace/ModelDetail.tsx`

- [ ] Write failing protocol/backend tests for `ollama_digest`, `upstream_model`, `runtime_image`, and `provider_claimed` evidence validation.
- [ ] Run focused Go tests and confirm the evidence fields are absent.
- [ ] Populate evidence from Ollama tag digests, upstream model listings, and pinned command images; include it in capacity.
- [ ] Persist evidence on `provider_routing_state`, reject an Ollama digest change for the same immutable offer version, and expose evidence in operations/marketplace JSON.
- [ ] Render explicit evidence labels without using an unqualified verified badge.
- [ ] Run protocol, CLI, store, API, and web tests; commit `feat: pin and disclose provider model evidence`.

### Task 6: One-command local hosting

**Files:**
- Modify: `cli/cmd/myference/main.go`
- Modify: `cli/cmd/myference/main_test.go`
- Modify: `README.md`
- Modify: `docs/demo.md`

- [ ] Write failing CLI tests showing `myference host --config PATH` discovers local Ollama models, adds a selected model idempotently, prints the activation URL, and never asks for a wallet key.
- [ ] Run the focused CLI tests and confirm `host` is unknown.
- [ ] Implement `host` as a thin orchestration of existing login, Ollama discovery, atomic config save, and reconnecting serve paths; preserve explicit `backend add` commands.
- [ ] Print actionable dependency errors and an activation URL for wallet-only steps.
- [ ] Document the short happy path and commit `feat: add guided local hosting command`.

### Task 7: Honest command-agent metering and real smoke tests

**Files:**
- Modify: `cli/internal/backend/command/command.go`
- Modify: `cli/internal/backend/command/command_test.go`
- Modify: `cli/internal/backend/command/proxy_sidecar.go`
- Modify: `docs/demo.md`

- [ ] Write a failing test proving command agents default to compute-only usage and cannot advertise token pricing without observed upstream usage.
- [ ] Run the focused command test and confirm the metering policy is absent.
- [ ] Expose the metering mode in backend evidence/capacity and preserve measured compute duration; aggregate upstream token usage only when a provider response includes it.
- [ ] Run all Go tests, all web tests, lint, build, `forge test`, and `git diff --check`.
- [ ] Build the CLI and run real local Ollama plus installed Codex smoke checks without printing credentials; record only model/runtime, metering mode, request outcome, and errors.
- [ ] Commit `test: verify simplified provider flows` and push `main`.
