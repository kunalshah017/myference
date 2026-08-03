# One-command CLI Installers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish checksum-verified one-command installers for Windows AMD64 and macOS AMD64/ARM64 and make them the primary installation path in product docs.

**Architecture:** Serve two dependency-light scripts from the existing Vite static site. Both resolve a release, select the platform artifact, verify `SHA256SUMS`, install the CLI and colocated agent proxy, and support deterministic local-fixture testing without a checksum bypass.

**Tech Stack:** POSIX shell, PowerShell 5+, GitHub Releases, Vite public assets, Vitest, GitHub Actions.

---

### Task 1: Lock the public installation contract

**Files:**
- Modify: `web/src/app/App.test.tsx`
- Create: `scripts/test-installers.sh`

- [ ] Add failing docs assertions for `irm https://myference.xyz/install.ps1 | iex` and `curl -fsSL https://myference.xyz/install.sh | sh`.
- [ ] Add a failing shell integration test expecting macOS AMD64 and ARM64 artifact selection, checksum rejection, and colocated CLI/proxy installation.
- [ ] Run both tests and confirm failure because the scripts and docs commands do not exist.

### Task 2: Implement and test both installers

**Files:**
- Create: `web/public/install.sh`
- Create: `web/public/install.ps1`
- Modify: `.github/workflows/verify.yml`

- [ ] Implement macOS detection, latest-tag resolution, archive/checksum download, exact verification, extraction, and atomic installation.
- [ ] Implement Windows AMD64 validation, GitHub release resolution, archive/checksum download, exact verification, extraction, installation, and persistent user PATH update.
- [ ] Add `sh -n`, fixture integration tests, and Windows PowerShell parser validation to CI.
- [ ] Run focused tests until green.

### Task 3: Make one-command installation primary in docs

**Files:**
- Modify: `web/src/app/DocsPage.tsx`
- Modify: `README.md`

- [ ] Replace manual extraction as the primary path with the two copyable installer commands.
- [ ] Retain manual release downloads and checksum instructions as the fallback path.
- [ ] Explain architecture detection, installation directories, PATH behavior, and the absence of code signing/notarization.
- [ ] Run web tests, lint, build, installer tests, and `git diff --check`.
- [ ] Commit, push `main`, monitor GitHub Actions and Render, then verify both public installer URLs.
