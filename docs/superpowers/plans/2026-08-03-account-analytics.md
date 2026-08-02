# Account Analytics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Index confirmed slash events and expose real customer usage and provider revenue analytics in the web dashboard.

**Architecture:** Monad events remain canonical for money movement while signed receipt proposals provide usage dimensions. A store aggregator returns one account-scoped analytics document through an authenticated API, consumed by focused Usage and Provider Analytics React panels.

**Tech Stack:** PostgreSQL migrations, Go indexer/store/HTTP, React/TypeScript, TanStack Query, Vitest.

---

### Task 1: Persist reorg-safe slashing events

**Files:** `migrations/000013_account_analytics.sql`, `server/internal/chain/indexer.go`, `server/internal/chain/indexer_integration_test.go`

- [ ] Add a failing integration assertion that a `ProviderSlashed` event creates a uniquely keyed ledger record.
- [ ] Create `chain_slashes(chain_id, contract_address, request_id, provider, amount, block_number, transaction_hash, indexed_at)` with provider/time indexes.
- [ ] Insert the decoded event before updating collateral and include `chain_slashes` in rewind deletion.
- [ ] Run the chain integration tests.

### Task 2: Aggregate account analytics

**Files:** `server/internal/store/analytics.go`, `server/internal/store/analytics_integration_test.go`

- [ ] Write a failing fixture test containing a customer, provider, receipt, confirmed settlement, and slash.
- [ ] Define `AccountAnalytics` with customer/provider totals, daily points, usage, settlements, and slashes; encode wei as strings.
- [ ] Implement account-scoped joins through sessions for customers and machines/accounts for providers, capped at 100 recent records and 30 UTC days.
- [ ] Run store integration tests.

### Task 3: Expose authenticated analytics

**Files:** `server/internal/api/analytics.go`, `server/internal/api/analytics_test.go`, `server/cmd/myference-server/main.go`

- [ ] Write tests for authentication failure and JSON response.
- [ ] Add `GET /api/account/analytics` using the same browser-account resolver and chain configuration as operations.
- [ ] Register the handler explicitly in the root mux without shadowing operations routes.
- [ ] Run API and command package tests.

### Task 4: Render usage, revenue, and slashing

**Files:** `web/src/lib/api.ts`, `web/src/features/analytics/UsageAnalytics.tsx`, `web/src/features/analytics/ProviderAnalytics.tsx`, tests, `web/src/app/DashboardShell.tsx`, `web/src/styles/global.css`

- [ ] Write failing component tests for confirmed totals, daily activity, settlement history, and slash history.
- [ ] Add `AnalyticsAPI.analytics()` and matching TypeScript types.
- [ ] Render customer analytics in Usage and provider analytics in Earnings; invalidate analytics on realtime request events.
- [ ] Remove the historical-data-unavailable notices now backed by the indexer.
- [ ] Run web tests, lint, and build.

### Task 5: Verify and deliver

- [ ] Run Go unit tests and available database/chain integration tests.
- [ ] Run the complete web verification suite.
- [ ] Run `git diff --check`, commit, and push to `main`.
