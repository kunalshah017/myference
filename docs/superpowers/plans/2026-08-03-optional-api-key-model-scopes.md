# Optional API Key Model Scopes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make API keys usable with every model by default while retaining catalog-backed model restrictions as an optional security control.

**Architecture:** Represent all-model access with the existing empty `models` array and enforce that meaning once in the server authorization boundary. Reuse `MarketplaceAPI.models()` in the dashboard, and align onboarding and documentation with the same scope contract.

**Tech Stack:** Go, PostgreSQL JSON scopes, React, TypeScript, TanStack Query, Vitest.

---

### Task 1: Server scope semantics

**Files:**
- Modify: `server/internal/auth/device.go`
- Test: `server/internal/auth/device_integration_test.go`

- [ ] Add a failing integration test that creates a key with `Models: []` and authorizes two different model IDs.
- [ ] Run `go test ./internal/auth -run 'TestAPIKeys'` and confirm creation fails because a model is required.
- [ ] Allow an empty model list during creation and treat it as a wildcard during authorization.
- [ ] Re-run the focused server test and confirm both unrestricted and restricted cases pass.

### Task 2: Dashboard key creation

**Files:**
- Modify: `web/src/features/auth/ApiKeys.tsx`
- Modify: `web/src/styles/global.css`
- Test: `web/src/features/auth/auth.test.tsx`

- [ ] Add failing component coverage asserting default creation sends `models: []` and optional restriction selects IDs fetched from `/api/models`.
- [ ] Run `npm test -- --run src/features/auth/auth.test.tsx` and confirm the new expectations fail.
- [ ] Add an all-model default, an optional restriction toggle, and a live-catalog multi-select using the existing marketplace client.
- [ ] Label unrestricted existing keys as `All models`, keep catalog failure isolated to restricted creation, and verify focused tests pass.

### Task 3: Onboarding and documentation

**Files:**
- Modify: `web/src/features/onboarding/onboarding.ts`
- Modify: `web/src/features/onboarding/OnboardingFlow.tsx`
- Modify: `web/src/features/onboarding/onboarding.test.ts`
- Modify: `web/src/features/onboarding/OnboardingFlow.test.tsx`
- Modify: `web/src/app/DocsPage.tsx`
- Modify: `web/src/features/playground/ChatPlayground.tsx`

- [ ] Add failing tests proving an unrestricted key completes access progress and onboarding creates `models: []`.
- [ ] Run the focused onboarding tests and confirm the new expectations fail.
- [ ] Create unrestricted onboarding keys, recognize them in progress derivation, and update user-facing guidance to explain optional restriction.
- [ ] Re-run focused tests and confirm they pass.

### Task 4: Full verification and delivery

**Files:**
- Review all modified files.

- [ ] Run server tests, web tests, web lint, and the production web build.
- [ ] Review the final diff for accidental scope broadening, secrets, and unrelated edits.
- [ ] Commit and push the verified change to `main`, then check deployment automation.
