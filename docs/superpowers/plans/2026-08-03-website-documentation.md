# Website Documentation Implementation Plan

**Goal:** Ship a public, accurate `/docs` guide for consuming and hosting inference in the current Myference web client.

**Architecture:** Add a focused `DocsPage` route to the existing React entry point. Keep content and presentation local to that component, reuse the established design tokens, and add only the small clipboard interaction needed by code samples.

**Tech stack:** React 19, TypeScript, Lucide React, CSS, Vitest, Testing Library.

---

### Task 1: Lock the public documentation contract with tests

**Files:**
- Modify: `web/src/app/App.test.tsx`
- Modify: `web/src/app/LandingPage.test.tsx`
- Modify: `web/src/app/DashboardShell.test.tsx`

- [x] Add a failing `/docs` route test covering the consumer and provider journeys, Windows/macOS installation, OpenAI and Anthropic endpoints, live base URL, security, pricing, and troubleshooting.
- [x] Add failing navigation assertions for landing and dashboard docs links.
- [x] Run focused tests and confirm the expected failures.

### Task 2: Implement the journey-first docs page

**Files:**
- Create: `web/src/app/DocsPage.tsx`
- Modify: `web/src/app/App.tsx`
- Modify: `web/src/app/LandingPage.tsx`
- Modify: `web/src/app/DashboardShell.tsx`

- [x] Add `/docs` routing before the landing-page fallback.
- [x] Build semantic section navigation, user steps, host steps, compatibility examples, advanced backends, settlement/security explanations, troubleshooting, and references.
- [x] Add reusable copyable code blocks with accessible feedback.
- [x] Point all product documentation links to `/docs`.
- [x] Run focused tests until green.

### Task 3: Match the existing product theme responsively

**Files:**
- Create: `web/src/styles/docs.css`
- Modify: `web/src/main.tsx`

- [x] Style the docs header, rails, content, callouts, tables, and code samples with existing tokens.
- [x] Collapse secondary navigation and grids at tablet/mobile widths.
- [x] Preserve focus visibility, readable line lengths, and reduced-motion behavior.

### Task 4: Verify and publish

**Files:**
- Modify as required by verification only.

- [x] Run all web tests.
- [x] Run lint.
- [x] Run the production build.
- [x] Review the diff for fake data, unsupported claims, secrets, and accidental changes.
- [ ] Commit and push `main` so Render deploys the static site.
- [ ] Validate the deployed `/docs` route and key navigation links.
