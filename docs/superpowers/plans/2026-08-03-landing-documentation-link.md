# Landing Documentation Link Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a direct documentation action to the landing page’s How it works section.

**Architecture:** Reuse the existing landing-page route and secondary-action styling. Add one semantic anchor after the process grid and verify its accessible name and destination in the existing landing test.

**Tech Stack:** React, TypeScript, Testing Library, Vitest, CSS.

---

### Task 1: Add and verify the documentation action

**Files:**
- Modify: `web/src/app/LandingPage.test.tsx`
- Modify: `web/src/app/LandingPage.tsx`
- Modify: `web/src/styles/landing.css`

- [x] Add `expect(screen.getByRole('link', { name: /see full documentation/i })).toHaveAttribute('href', '/docs')` to the landing test.
- [x] Run `npm test -- --run src/app/LandingPage.test.tsx` and confirm it fails because the link is absent.
- [x] Add `<a className="landing-secondary landing-docs-link" href="/docs">See full documentation <ArrowRight size={17} /></a>` after the process-card grid.
- [x] Add only the spacing needed for `.landing-docs-link` and preserve the existing responsive layout.
- [x] Run the focused test, full web tests, lint, and production build; all must pass.
- [ ] Commit, push `main`, monitor GitHub Actions, and verify the link on `https://myference.xyz`.
