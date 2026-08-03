# PostHog Client Analytics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add privacy-first PostHog product analytics to the Myference browser client through `t.myference.xyz`.

**Architecture:** Keep SDK configuration and event capture in one small library module. Initialize once at the application entry point, then instrument existing onboarding and dashboard callbacks with safe, low-cardinality properties.

**Tech Stack:** React 19, TypeScript, Vite, Vitest, Testing Library, `posthog-js`.

---

### Task 1: Analytics boundary

**Files:**
- Create: `web/src/lib/analytics.ts`
- Create: `web/src/lib/analytics.test.ts`
- Modify: `web/src/main.tsx`
- Modify: `web/package.json`
- Modify: `web/package-lock.json`

- [ ] **Step 1: Write failing tests** that require production initialization with the public token, `https://t.myference.xyz`, US Cloud UI host, pageview/pageleave capture, disabled autocapture/session recording, and no initialization on localhost.
- [ ] **Step 2: Run `npm test -- --run src/lib/analytics.test.ts`** and confirm failure because the analytics module does not exist.
- [ ] **Step 3: Install `posthog-js` and implement `initializeAnalytics` and `captureEvent`** with no user identifiers or sensitive automatic capture.
- [ ] **Step 4: Initialize analytics once in `web/src/main.tsx`** before rendering React.
- [ ] **Step 5: Re-run the focused test** and require all analytics-boundary cases to pass.

### Task 2: Safe activation funnel events

**Files:**
- Modify: `web/src/app/DashboardShell.tsx`
- Modify: `web/src/app/DashboardShell.test.tsx`
- Modify: `web/src/features/onboarding/OnboardingFlow.tsx`
- Modify: `web/src/features/onboarding/OnboardingFlow.test.tsx`

- [ ] **Step 1: Add failing component tests** for role selection, skip, resume, completion, wallet connection, and dashboard view events.
- [ ] **Step 2: Run the focused component tests** and confirm the missing captures fail.
- [ ] **Step 3: Add `captureEvent` calls to existing state-transition callbacks.** Send only `role`, `surface`, or `view`; never prompt, model response, API key, wallet address, account ID, signature, or transaction data.
- [ ] **Step 4: Re-run the focused component tests** and require the funnel assertions to pass.

### Task 3: Verification and delivery

**Files:**
- Modify only if verification exposes a defect.

- [ ] **Step 1: Run `npm test -- --run`, `npm run lint`, and `npm run build` from `web/`.**
- [ ] **Step 2: Run `git diff --check` and review the diff for sensitive analytics properties.**
- [ ] **Step 3: Commit and push to `main`, monitor GitHub Actions, and confirm the Render deployment is live.**
- [ ] **Step 4: Load the deployed client and verify a PostHog request goes through `https://t.myference.xyz` without affecting application rendering.**
