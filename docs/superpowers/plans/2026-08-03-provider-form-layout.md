# Provider Form Layout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the malformed provider controls with accessible, aligned collateral and immutable-offer cards without changing transaction behavior.

**Architecture:** Keep transaction state and contract calls in the existing `ProviderConsole` and `Offers` components. Introduce semantic card, fieldset, field-wrapper, helper-text, unit, action-row, and form-footer markup, then style those stable classes with responsive grids derived from the existing dashboard tokens.

**Tech Stack:** React 19, TypeScript, Testing Library, Vitest, CSS, viem, Vite

---

### Task 1: Specify provider-form semantics in tests

**Files:**
- Modify: `web/src/features/provider/provider.test.tsx`

- [ ] **Step 1: Add a failing structure and explanation test**

Add assertions after the provider console renders:

```tsx
expect(screen.getByRole('region', { name: /provider collateral/i })).toBeVisible()
expect(screen.getByRole('group', { name: /model identity/i })).toBeVisible()
expect(screen.getByRole('group', { name: /usage pricing/i })).toBeVisible()
expect(screen.getByLabelText(/bond amount/i)).toHaveAccessibleDescription(/minimum 5 mon/i)
expect(screen.getByLabelText(/input tokens/i)).toHaveAccessibleDescription(/one million input tokens/i)
expect(screen.getByLabelText(/output tokens/i)).toHaveAccessibleDescription(/one million generated tokens/i)
expect(screen.getByLabelText(/compute time/i)).toHaveAccessibleDescription(/each second/i)
expect(screen.getByText(/new immutable on-chain price version/i)).toBeVisible()
```

- [ ] **Step 2: Run the focused test and verify RED**

Run: `npm run test:run -- src/features/provider/provider.test.tsx` from `web/`

Expected: FAIL because the provider collateral region, fieldsets, and accessible descriptions do not exist.

- [ ] **Step 3: Commit the failing test**

```bash
git add web/src/features/provider/provider.test.tsx
git commit -m "test: specify provider form layout"
```

### Task 2: Rebuild collateral and offer markup

**Files:**
- Modify: `web/src/features/provider/ProviderConsole.tsx`
- Modify: `web/src/features/provider/Offers.tsx`
- Test: `web/src/features/provider/provider.test.tsx`

- [ ] **Step 1: Implement the collateral card**

In `ProviderConsole`, replace `bond-control` with an `aria-labelledby` region containing:

```tsx
<section className="provider-card collateral-card" aria-labelledby="collateral-title">
  <div className="provider-card-heading">
    <div><p className="eyebrow">Collateral</p><h3 id="collateral-title">Secure your provider</h3></div>
    <div className="provider-balance"><span>Bonded</span><strong>{formatUnits(BigInt(operations.data.provider_bond_wei), 18)} MON</strong><code>{operations.data.provider_bond_wei} wei</code></div>
  </div>
  <p className="provider-card-copy">Collateral backs requests served by your machines. It remains yours unless a proven violation is slashed.</p>
  <form className="collateral-form" onSubmit={deposit}>
    <div className="provider-field">
      <label htmlFor="bond-mon">Bond amount</label>
      <div className="input-with-unit"><input id="bond-mon" inputMode="decimal" aria-describedby="bond-help" value={bond} onChange={(event)=>setBond(event.target.value)} required/><span>MON</span></div>
      <small id="bond-help">Minimum 5 MON. Deposits become active after chain confirmation.</small>
    </div>
    <div className="provider-actions"><button type="submit">Deposit collateral</button><button type="button">Request bond exit</button></div>
  </form>
  <p className="provider-note">Bond exits use the contract delay before funds can be withdrawn.</p>
</section>
```

Retain the existing conditional exit label and writer call in the secondary button.

- [ ] **Step 2: Implement the immutable-offer card**

In `Offers`, keep the existing offer ledger and state, but wrap controls as:

```tsx
<section className="provider-card offer-operations" aria-labelledby="offers-title">
  <div className="provider-card-heading"><div><p className="eyebrow">Pricing</p><h3 id="offers-title">Publish an offer</h3></div></div>
  <p className="provider-card-copy">Each publish creates a new immutable on-chain price version. Existing sessions keep the version they already selected.</p>
  <form className="offer-form" onSubmit={(event)=>void publish(event)}>
    <fieldset><legend>Model identity</legend><div className="provider-field-grid">
      <div className="provider-field"><label htmlFor="offer-id">Offer name</label><input id="offer-id" aria-describedby="offer-id-help" value={id} onChange={(event)=>setID(event.target.value)} required/><small id="offer-id-help">A stable name used by your CLI backend.</small></div>
      <div className="provider-field"><label htmlFor="offer-model">Model</label><input id="offer-model" aria-describedby="offer-model-help" value={model} onChange={(event)=>setModel(event.target.value)} required/><small id="offer-model-help">Must exactly match the model advertised by the machine.</small></div>
    </div></fieldset>
    <fieldset><legend>Usage pricing</legend><div className="provider-field-grid provider-pricing-grid">
      <div className="provider-field"><label htmlFor="offer-input">Input tokens</label><div className="input-with-unit"><input id="offer-input" inputMode="numeric" aria-describedby="offer-input-help" value={input} onChange={(event)=>setInput(event.target.value)} required/><span>wei / 1M</span></div><small id="offer-input-help">Charge in wei for one million input tokens.</small></div>
      <div className="provider-field"><label htmlFor="offer-output">Output tokens</label><div className="input-with-unit"><input id="offer-output" inputMode="numeric" aria-describedby="offer-output-help" value={output} onChange={(event)=>setOutput(event.target.value)} required/><span>wei / 1M</span></div><small id="offer-output-help">Charge in wei for one million generated tokens.</small></div>
      <div className="provider-field"><label htmlFor="offer-compute">Compute time</label><div className="input-with-unit"><input id="offer-compute" inputMode="numeric" aria-describedby="offer-compute-help" value={compute} onChange={(event)=>setCompute(event.target.value)} required/><span>wei / sec</span></div><small id="offer-compute-help">Charge in wei for each second of measured compute.</small></div>
    </div></fieldset>
    <label className="workspace-option" htmlFor="offer-workspace"><input id="offer-workspace" type="checkbox" checked={workspace} onChange={(event)=>setWorkspace(event.target.checked)}/><span><strong>Allow temporary workspace</strong><small>Enable only for isolated CLI coding agents that accept disposable project files.</small></span></label>
    <div className="provider-form-footer"><p>Publishing creates a new immutable on-chain price version.</p><button type="submit">Publish offer version</button></div>
  </form>
</section>
```

Use `aria-describedby`, numeric input modes, exact wei units, and the helper copy required by Task 1. Preserve `publishOffer` arguments and capabilities unchanged.

- [ ] **Step 3: Run the focused test and verify GREEN**

Run: `npm run test:run -- src/features/provider/provider.test.tsx` from `web/`

Expected: PASS, including the existing collateral conversion and workspace-capability assertions.

- [ ] **Step 4: Commit the semantic implementation**

```bash
git add web/src/features/provider/ProviderConsole.tsx web/src/features/provider/Offers.tsx web/src/features/provider/provider.test.tsx
git commit -m "feat: structure provider transaction forms"
```

### Task 3: Align and responsively style provider controls

**Files:**
- Modify: `web/src/styles/global.css`

- [ ] **Step 1: Replace obsolete provider layout rules**

Add focused styles for `.provider-setup-grid`, `.provider-card`, `.provider-card-heading`, `.provider-balance`, `.provider-field-grid`, `.provider-field`, `.input-with-unit`, `.provider-actions`, `.workspace-option`, and `.provider-form-footer`. Use existing paper, node-mist, line, circuit-ink, relay-violet, proof-mint, and font tokens. Keep cards square-cornered and information-dense to match the dashboard.

- [ ] **Step 2: Add responsive behavior**

At `max-width: 760px`, make `.provider-field-grid`, `.provider-card-heading`, `.collateral-form`, and `.provider-form-footer` single-column; make `.provider-actions` stretch its buttons. Do not use element-order selectors for field placement.

- [ ] **Step 3: Run focused and full verification**

Run from `web/`:

```bash
npm run test:run -- src/features/provider/provider.test.tsx
npm run test:run
npm run lint
npm run build
```

Expected: all provider tests and the complete suite pass, lint reports no errors, and Vite produces `dist/` successfully.

- [ ] **Step 4: Commit styling**

```bash
git add web/src/styles/global.css
git commit -m "fix: align provider setup forms"
```

### Task 4: Visual verification and deployment

**Files:**
- No source changes expected unless browser verification exposes a concrete defect.

- [ ] **Step 1: Run the web client locally and inspect desktop layout**

Run: `npm run dev -- --host 127.0.0.1` from `web/`. Open the provider screen and verify labels remain attached to their controls, cards align, units are visible, and transaction explanations fit without overlap.

- [ ] **Step 2: Inspect mobile layout**

Use a 390 px viewport. Verify the two-column grids collapse, actions remain reachable, no horizontal overflow occurs, and the checkbox copy wraps cleanly.

- [ ] **Step 3: Push and deploy**

Push `main`, monitor the Render static-site deploy to `live`, and verify `https://myference-web.onrender.com/app` returns HTTP 200.

- [ ] **Step 4: Resume the real transaction flow**

Reconnect or preserve the provider browser session, enter `5` MON in the redesigned collateral field, submit the transaction, leave MetaMask approval to the user, then verify the transaction receipt and indexed bonded balance before continuing machine authorization.
