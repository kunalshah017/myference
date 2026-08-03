# Playground Reservation Guidance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make live-model playground requests fit an explicit token budget and report insufficient request/session budgets accurately.

**Architecture:** Mirror the server's conservative reservation arithmetic in a small client helper using marketplace rates. Keep provider eligibility and economic eligibility separate in the router so the HTTP layer can return a useful status without weakening any routing check.

**Tech Stack:** Go, React, TypeScript, Vitest, TanStack Query.

---

### Task 1: Classify budget rejection

**Files:**
- Modify: `server/internal/router/router.go`
- Test: `server/internal/router/router_test.go`
- Modify: `server/internal/api/openai.go`
- Test: `server/internal/api/openai_integration_test.go`

- [ ] Change the existing insufficient-session assertion to expect `ErrInsufficientBudget` and add an API assertion for HTTP 402.
- [ ] Run the focused Go tests and confirm they fail because the error does not exist.
- [ ] Track structurally eligible candidates rejected only by request/session ceilings and return `ErrInsufficientBudget` when no affordable candidate remains.
- [ ] Map that error to HTTP 402 and re-run the focused tests.

### Task 2: Bound and explain playground requests

**Files:**
- Modify: `web/src/features/playground/ChatPlayground.tsx`
- Test: `web/src/features/playground/ChatPlayground.test.tsx`

- [ ] Update the component test to require a default 256-token output ceiling, a 1 MON request ceiling, a visible estimate, and `max_completion_tokens: 256` in the request body.
- [ ] Run the focused Vitest file and confirm the payload/default assertions fail.
- [ ] Add the output-token control, conservative estimate, local budget validation, and explicit request parameter.
- [ ] Re-run the focused test and confirm it passes.

### Task 3: Verify and deliver

**Files:**
- Review all modified files.

- [ ] Run `make verify` and confirm every Go, contract, web, lint, build, and script check passes.
- [ ] Review the diff for weakened authorization or routing checks.
- [ ] Commit and push to `main`, monitor GitHub Actions and Render deployment, then inspect the live playground in the existing browser tab.
