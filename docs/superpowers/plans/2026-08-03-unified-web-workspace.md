# Unified Web Workspace Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver one consistently styled Myference landing page and application workspace for consuming and hosting inference.

**Architecture:** A `DashboardShell` owns view navigation and composes existing live-data features into customer and provider pages. New chat and API guide components use the existing server APIs directly. Landing and dashboard CSS share the existing token system and rectangular editorial-grid grammar.

**Tech Stack:** React 19, TypeScript, TanStack Query, Lucide React, Vitest, Vite, existing Go HTTP APIs and Monad contract writer.

---

### Task 1: Add the dashboard shell

**Files:**
- Create: `web/src/app/DashboardShell.tsx`
- Create: `web/src/app/DashboardShell.test.tsx`
- Modify: `web/src/app/App.tsx`

- [ ] Write a failing test that renders `DashboardShell` and clicks the `Host inference` navigation item.
- [ ] Verify failure with `npm test -- --run src/app/DashboardShell.test.tsx`.
- [ ] Implement `type DashboardView = 'overview' | 'models' | 'playground' | 'funds' | 'api' | 'usage' | 'hosting' | 'earnings'` and render the selected real feature panel.
- [ ] Re-run the focused test and confirm it passes.

### Task 2: Add real browser chat and API access

**Files:**
- Modify: `web/src/lib/api.ts`
- Create: `web/src/features/playground/ChatPlayground.tsx`
- Create: `web/src/features/playground/ChatPlayground.test.tsx`
- Create: `web/src/features/auth/ApiAccessGuide.tsx`

- [ ] Write a failing chat test asserting `{model, stream:false, messages}` is posted to `/v1/chat/completions` with `Authorization: Bearer <key>`.
- [ ] Add `InferenceAPI.chat(model, apiKey, messages)` returning the first assistant message from the OpenAI-compatible response.
- [ ] Build the model/key/prompt form with loading, response, and server-error states; keep the key only in memory.
- [ ] Add the environment-derived base URL and endpoint examples beside the existing scoped-key component.
- [ ] Run chat and auth tests.

### Task 3: Restructure customer and provider views

**Files:**
- Create: `web/src/app/DashboardOverview.tsx`
- Modify: `web/src/features/provider/ProviderConsole.tsx`
- Modify: `web/src/features/activity/Activity.tsx`
- Modify: `web/src/styles/global.css`

- [ ] Add operation summary cards sourced from `AccountOperations`; disconnected state directs the account to connect.
- [ ] Compose Models, Playground, Funds, API access, and Usage as customer navigation destinations.
- [ ] Compose Hosting and Earnings as provider destinations using current machine, offer, bond, claimable, and revenue values.
- [ ] Label historical slashing and token-cost analytics unavailable because the indexer response does not expose them.
- [ ] Run the dashboard and existing feature tests.

### Task 4: Align the landing page and add icons

**Files:**
- Modify: `web/package.json`
- Modify: `web/package-lock.json`
- Modify: `web/src/app/LandingPage.tsx`
- Modify: `web/src/styles/landing.css`

- [ ] Install `lucide-react` and use icons with visible text throughout navigation and action cards.
- [ ] Replace the dark navy/lime theme with current app tokens, square rules, paper/mist panels, and the shared routing rail.
- [ ] Preserve responsive layout, keyboard focus, semantic headings, and reduced-motion behavior.
- [ ] Run landing tests and production build.

### Task 5: Complete verification and delivery

**Files:**
- Modify: tests only if an assertion needs to reflect intentional navigation text.

- [ ] Run `npm test -- --run` with local listener permission.
- [ ] Run `npm run lint`.
- [ ] Run `npm run build`.
- [ ] Inspect `git diff --check` and confirm no fake dashboard metrics or secrets are present.
- [ ] Commit and push the verified result to `main`.
