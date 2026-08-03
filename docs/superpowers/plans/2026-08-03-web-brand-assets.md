# Web Brand Assets Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish a theme-consistent favicon family, installable-site icons, Open Graph image, manifest, and complete social metadata.

**Architecture:** Keep editable SVG sources in `web/public`, generate deterministic PNG/ICO derivatives, and reference them through static HTML metadata. A small verification script checks dimensions and declarations without adding application dependencies.

**Tech Stack:** SVG, PNG, ICO, HTML metadata, Web App Manifest, Node.js standard library, Vite.

---

### Task 1: Lock the asset and metadata contract

**Files:**
- Create: `scripts/test-brand-assets.mjs`
- Modify: `web/index.html`

- [x] Write failing checks for every required asset, expected dimensions, manifest values, favicon links, canonical URL, Open Graph metadata, Twitter metadata, and alt text.
- [x] Run `node scripts/test-brand-assets.mjs` and confirm it fails because the asset set is absent.

### Task 2: Create the vector identity and raster family

**Files:**
- Create: `web/public/favicon.svg`
- Create: `web/public/app-icon.svg`
- Create: `web/public/og-image.svg`
- Create: `web/public/brand/providers/*.png`
- Create: `scripts/inline-og-logos.mjs`
- Create: `web/public/favicon-16x16.png`
- Create: `web/public/favicon-32x32.png`
- Create: `web/public/apple-touch-icon.png`
- Create: `web/public/icon-192.png`
- Create: `web/public/icon-512.png`
- Create: `web/public/mstile-150x150.png`
- Create: `web/public/favicon.ico`
- Create: `web/public/safari-pinned-tab.svg`

- [x] Draw the geometric `M/` favicon using only local SVG paths and established color tokens.
- [x] Compose a 1200×630 social card using the landing-page grid, typography, routing nodes, and concise product message.
- [x] Render deterministic raster sizes and package the 32px PNG into an ICO fallback.
- [x] Inspect the 16px, 180px, 512px, and 1200×630 outputs for clipping, legibility, contrast, and unintended artifacts.

### Task 3: Wire metadata and verify deployment

**Files:**
- Create: `web/public/site.webmanifest`
- Modify: `web/index.html`
- Modify: `.github/workflows/verify.yml`

- [x] Add favicon, Apple icon, manifest, canonical, theme color, Open Graph, and Twitter declarations.
- [x] Add brand-asset verification to the web CI job.
- [x] Run asset checks, web tests, lint, and production build; confirm Vite copies every asset.
- [ ] Commit and push `main`, monitor GitHub Actions and Render, then verify public assets and metadata.
