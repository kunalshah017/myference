# First-time Onboarding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a skippable, resumable onboarding that takes customers through a real first inference and providers through a real first active offer.

**Architecture:** Add one onboarding feature boundary with pure progress/ranking helpers and a single orchestrator component. Reuse existing APIs, contract writer, wallet control, device approval, and provider controls; browser storage remembers navigation preference only while success comes from indexed state and analytics.

**Tech Stack:** React 19, TypeScript, TanStack Query, viem, Vitest, Testing Library, existing CSS token system.

---

### Task 1: Real-state progress and recommendation helpers

**Files:**
- Create: `web/src/features/onboarding/onboarding.ts`
- Test: `web/src/features/onboarding/onboarding.test.ts`

- [ ] **Step 1: Write failing tests** for cheapest-live-model ranking, bounded starter-cost calculation, active-session detection, settled customer completion, connected provider detection, and active provider-offer detection.
- [ ] **Step 2: Run `npm test -- --run src/features/onboarding/onboarding.test.ts` from `web/`** and verify failure because the onboarding module does not exist.
- [ ] **Step 3: Implement pure helpers** using `bigint` only for monetary arithmetic. Rank only non-stale models with capacity and providers; use 2,000 input, 1,000 output, 120 seconds, and a ceiling 20% buffer. Treat only unexpired, non-finalized sessions with unused allowance as active.
- [ ] **Step 4: Re-run the focused test** and verify all helper cases pass.

### Task 2: Customer onboarding flow

**Files:**
- Create: `web/src/features/onboarding/OnboardingFlow.tsx`
- Test: `web/src/features/onboarding/OnboardingFlow.test.tsx`
- Modify: `web/src/features/auth/ConnectWallet.tsx`

- [ ] **Step 1: Write failing component tests** proving the primary customer role, provider alternative, skip callback, cheapest live model default, no-inventory recovery, existing-key warning, one-time new key handoff, and successful real inference completion callback.
- [ ] **Step 2: Run the focused component test** and verify it fails because `OnboardingFlow` does not exist.
- [ ] **Step 3: Implement the role choice and customer route map.** After wallet connection, query marketplace, operations, API keys, analytics, and reference price. Render only the next incomplete action plus concise completed-step summaries.
- [ ] **Step 4: Implement focused real actions.** Deposit and open-session actions call `ViemMarketWriter`; API-key creation calls `AuthAPI.createAPIKey`; test chat calls `InferenceAPI.chat`. Retain the new secret only in memory and pass it directly into the final request form.
- [ ] **Step 5: Implement recovery states** for 401/session loss, no inventory, rejected transaction, indexer delay, lost existing key secret, expired session, and provider execution errors.
- [ ] **Step 6: Re-run the focused component test** and verify the customer path passes.

### Task 3: Provider onboarding flow

**Files:**
- Modify: `web/src/features/onboarding/OnboardingFlow.tsx`
- Modify: `web/src/features/onboarding/OnboardingFlow.test.tsx`

- [ ] **Step 1: Add failing tests** for platform install guidance, real machine completion, embedded device approval, collateral gating, provider-console handoff, and completion only for an enabled healthy backend linked to a published offer.
- [ ] **Step 2: Run the focused test** and confirm these provider assertions fail.
- [ ] **Step 3: Implement provider steps** with OS-aware installer commands, `myference host`, existing `DeviceApproval`, focused collateral action, and existing `ProviderConsole` for discovered backend pricing/publishing.
- [ ] **Step 4: Re-run the focused test** and verify provider behavior passes without mock production inventory.

### Task 4: Dashboard entry, skip, resume, and reminder

**Files:**
- Modify: `web/src/app/DashboardShell.tsx`
- Modify: `web/src/app/DashboardOverview.tsx`
- Modify: `web/src/app/DashboardShell.test.tsx`

- [ ] **Step 1: Write failing shell tests** for first-visit onboarding, skip-to-dashboard, persistent “Continue setup” reminder, remembered role, resume, and reminder removal after the completion callback.
- [ ] **Step 2: Run `npm test -- --run src/app/DashboardShell.test.tsx`** and confirm the new expectations fail.
- [ ] **Step 3: Integrate onboarding** into the shell using `myference:onboarding-role` and `myference:onboarding-skipped` local-storage keys. Pass connected sessions through the existing topbar wallet control and let onboarding navigate to regular dashboard views.
- [ ] **Step 4: Add the overview reminder** with progress count, next action, continue button, and path switch. Keep it present after skip until real completion.
- [ ] **Step 5: Re-run shell tests** and verify all onboarding and existing navigation cases pass.

### Task 5: Dashboard-native visual system and responsive behavior

**Files:**
- Modify: `web/src/styles/global.css`

- [ ] **Step 1: Add the route-map layout** using existing design tokens: paper surfaces, violet active nodes, mint completed nodes, sharp circuit borders, compact monospace labels, and a restrained content column.
- [ ] **Step 2: Add mobile behavior** that changes the route map to a horizontal progress strip, keeps actions full width, avoids overflow, and preserves visible focus indicators.
- [ ] **Step 3: Run `npm run build`** and verify TypeScript and CSS bundling complete successfully.

### Task 6: Verification, visual review, and delivery

**Files:**
- Modify only if verification exposes a defect.

- [ ] **Step 1: Run `npm test -- --run`, `npm run lint`, and `npm run build` in `web/`** and require zero failures.
- [ ] **Step 2: Inspect `/app` in the browser** at desktop and mobile widths, exercising role choice, skip/reminder, and no-session recovery. Correct any layout or accessibility defects with a failing regression test where behavior changes.
- [ ] **Step 3: Review the diff** for accidental fake data, raw wei in primary UI, lost-key leakage, and unrelated changes.
- [ ] **Step 4: Commit and push** the plan and implementation to `main`, then monitor GitHub Actions and verify the deployed `/app` loads the onboarding assets.
