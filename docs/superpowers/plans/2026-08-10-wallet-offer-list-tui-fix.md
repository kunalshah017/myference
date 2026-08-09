# Wallet Offer List TUI Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Show every wallet-owned offer in the hosting TUI and prevent selection of an existing offer from opening the new-offer pricing flow.

**Architecture:** Reuse the account data already fetched by `Model.accountCommand` and the existing `providerops.Compatible` predicate. Build one small flattened selection model for local-backend and wallet-offer rows so rendering and keyboard handling use the same ordering; keep repricing behind an explicit `e` key.

**Tech Stack:** Go, Bubble Tea, existing `account`, `config`, and `providerops` packages.

---

### Task 1: Render the complete wallet offer list

**Files:**
- Modify: `cli/internal/tui/model_test.go`
- Modify: `cli/internal/tui/model.go`

- [ ] **Step 1: Write the failing rendering test**

Add `TestOffersRenderEveryWalletOffer` with one attached Ollama backend and multiple wallet offers. Assert that `ViewText` contains the **This machine** and **Wallet offers** headings, the attached `local-qwen` offer, and an incompatible `gpt-5.6-terra` offer marked `No matching local provider`.

- [ ] **Step 2: Run the focused test and verify RED**

Run: `GOCACHE=/private/tmp/myference-go-cache go test ./cli/internal/tui -run TestOffersRenderEveryWalletOffer -count=1`

Expected: FAIL because `ScreenOffers` only iterates `currentBackends()`.

- [ ] **Step 3: Implement the two-section rendering**

Add a minimal `offerRows()` helper that returns local backend rows followed by wallet rows. Render all `model.account.Offers`; derive wallet state by scanning local backends with `providerops.Compatible` and `EffectiveOfferID()`.

- [ ] **Step 4: Run the focused test and verify GREEN**

Run: `GOCACHE=/private/tmp/myference-go-cache go test ./cli/internal/tui -run TestOffersRenderEveryWalletOffer -count=1`

Expected: PASS.

### Task 2: Correct selection and repricing behavior

**Files:**
- Modify: `cli/internal/tui/model_test.go`
- Modify: `cli/internal/tui/model.go`

- [ ] **Step 1: Write failing interaction tests**

Add tests proving: Enter on an attached machine or wallet row stays on `ScreenOffers` and does not return a pricing command; Enter on an attachable wallet row returns the existing attachment command; Enter on an unavailable wallet row remains on the screen with `Configure a matching provider first`; and `e` on a public machine row opens `ScreenPricing`.

- [ ] **Step 2: Run the interaction tests and verify RED**

Run: `GOCACHE=/private/tmp/myference-go-cache go test ./cli/internal/tui -run 'TestAttachedOfferSelectionDoesNotOpenPricing|TestWalletOfferSelection|TestExplicitRepricing' -count=1`

Expected: FAIL because cursor movement only covers machine rows and Enter on every published backend opens pricing.

- [ ] **Step 3: Implement row-based keyboard handling**

Move the Offers cursor over the flattened rows. Keep machine-row attachment for unpublished backends, attach a wallet row when it has exactly one compatible unattached backend, show a status message for attached, unavailable, or ambiguous wallet rows, and call `openPricing` only for an unpublished backend with no reusable offer or when `e` is pressed on a machine row.

- [ ] **Step 4: Run TUI tests and verify GREEN**

Run: `GOCACHE=/private/tmp/myference-go-cache go test ./cli/internal/tui -count=1`

Expected: PASS.

### Task 3: Verify and release the correction

**Files:**
- Modify only if verification exposes a defect in the files above.

- [ ] **Step 1: Run repository verification**

Run: `GOCACHE=/private/tmp/myference-go-cache make verify`

Expected: all Go, contract, web, lint, build, and shell checks pass.

- [ ] **Step 2: Run Windows and installer verification**

Run: `GOCACHE=/private/tmp/myference-go-cache GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -exec=/usr/bin/true ./cli/... -count=1`

Run: `bash scripts/test-installers.sh`

Expected: both commands pass.

- [ ] **Step 3: Commit, push, and publish the next alpha**

Commit the spec, plan, tests, and implementation; push `main`; build the next unused alpha tag with `scripts/build-release.sh`; verify `SHA256SUMS` and embedded `VERSION`; publish the release; install the exact macOS arm64 artifact; and confirm the TUI shows all wallet offers without opening pricing for `local-qwen`.
