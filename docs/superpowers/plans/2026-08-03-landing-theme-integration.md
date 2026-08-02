# Landing Theme Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add the supplied landing-page visual direction to the Myference web client while keeping the existing live console available at `/app`.

**Architecture:** Keep the existing `App` as the operational console and extract it into `OperationalApp`. Add a small route switch in `App` and a focused `LandingPage` component with scoped CSS. Use CSS-only network artwork to avoid remote image/data dependencies and preserve truthful live-data messaging.

**Tech Stack:** React 19, TypeScript, Vite, Vitest, existing CSS token system.

---

### Task 1: Add route and landing-page tests

**Files:**
- Modify: `web/src/app/App.test.tsx`
- Create: `web/src/app/LandingPage.test.tsx`

- [ ] Write tests proving `/` renders the landing heading and `/app` renders the existing marketplace heading.
- [ ] Run `npm test -- --run` from `web/` and confirm the new route test fails before implementation.

### Task 2: Build the landing page shell

**Files:**
- Create: `web/src/app/LandingPage.tsx`
- Modify: `web/src/app/App.tsx`

- [ ] Add semantic header, hero, network visual, how-it-works steps, pricing, FAQ, and footer.
- [ ] Use links to `/app`, `#how-it-works`, and `#pricing`; do not copy the reference waitlist URL or remote logo assets.
- [ ] Rename the existing component to `OperationalApp` and route by `window.location.pathname`, defaulting unknown paths to the landing page.
- [ ] Run the focused tests and confirm they pass.

### Task 3: Apply the visual theme without breaking console styles

**Files:**
- Modify: `web/src/styles/tokens.css`
- Modify: `web/src/styles/global.css`
- Create: `web/src/styles/landing.css`
- Modify: `web/src/main.tsx`

- [ ] Add the reference navy/lime tokens while retaining console token names.
- [ ] Import scoped landing styles and implement responsive layout, glow, grid, nodes, and reduced-motion behavior.
- [ ] Keep form/button selectors compatible with all existing feature components.
- [ ] Run lint and the full test suite.

### Task 4: Remove temporary source and verify production

**Files:**
- Delete: `web-only-landing-page-and-theme/`

- [ ] Run `npm run build` from `web/`.
- [ ] Run `npm test -- --run` and `npm run lint` from `web/`.
- [ ] Remove the exact temporary reference directory only after the checks pass.
- [ ] Run `git status --short` and verify no reference files remain.

