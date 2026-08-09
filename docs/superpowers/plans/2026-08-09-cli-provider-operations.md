# CLI Provider Operations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the CLI the only model-hosting control plane, add terminal-driven offer and collateral operations with browser wallet approval, and reduce the web provider console to collateral plus repricing of existing account-owned offers.

**Architecture:** Add one account-owned provider projection shared by machine-authenticated CLI reads and browser-authenticated web reads. Refactor the short-lived activation endpoint into typed provider-action drafts whose completion is proven from finalized indexed chain state. Reuse that service from CLI commands and Bubble Tea screens, then synchronize compatible web-published price versions into running providers without exposing or importing wallet keys.

**Tech Stack:** Go 1.25, PostgreSQL projections, existing Monad indexer and Go Ethereum hashes, Bubble Tea v2, React 19, TanStack Query, viem, Vitest.

---

### Task 1: Account-owned provider projection

**Files:**
- Create: `server/internal/store/provider_account.go`
- Create: `server/internal/store/provider_account_integration_test.go`
- Modify: `server/internal/chain/client.go`
- Modify: `server/cmd/myference-server/runtime.go`

- [ ] **Step 1: Write the failing tenant-isolation integration test**

Create two accounts, machines, routing rows, and colliding plaintext offer IDs. Insert finalized chain offers for each wallet and require `ProviderAccount` to return only the requested account and `MachineOfferVersions` to return only the authenticated machine.

```go
first, err := repository.ProviderAccount(ctx, "account-one", ProviderAccountConfig{ChainID: 10143, ContractAddress: contract, MinimumBondWei: "5000000000000000000"})
if err != nil || len(first.Offers) != 1 || first.Offers[0].OfferID != "local-qwen" {
	t.Fatalf("provider account=%+v err=%v", first, err)
}
versions, err := repository.MachineOfferVersions(ctx, "machine-one", "account-one", 10143, contract)
if err != nil || versions["local-qwen"] != 2 {
	t.Fatalf("versions=%v err=%v", versions, err)
}
```

- [ ] **Step 2: Run the focused test and verify RED**

Run: `go test ./server/internal/store -run 'TestProviderAccount' -count=1 -v`

Expected: FAIL because `ProviderAccount` and its types do not exist.

- [ ] **Step 3: Implement the projection**

Define focused response types:

```go
type EditableOffer struct {
	OfferID, Model, MeteringMode string
	Capabilities []string
	Version uint64
	InputPerMillionWei, OutputPerMillionWei, ComputePerSecondWei string
}

type ProviderAccount struct {
	ChainID uint64
	ContractAddress, WalletAddress, MinimumBondWei string
	ProviderBondWei, ClaimableWei, ProviderEarningsWei string
	BondExitAvailableAt uint64
	Offers []EditableOffer
}
```

Join `provider_routing_state` to `machines` with `m.account_id=$1`, then join `chain_offers` using the account wallet, offer hash, model hash, and capability hash. Select the newest compatible version per plaintext offer ID. Implement `MachineOfferVersions` with both machine ID and account ID in the ownership predicate.

- [ ] **Step 4: Read the contract minimum bond once at server startup**

Extend `chain.ReceiptTerms` with `MinimumBond *big.Int`, load `contract.MinimumBond`, and pass its decimal string into the provider-account API configuration. Do not add a database migration for a contract constant.

- [ ] **Step 5: Verify and commit**

Run: `gofmt -w server/internal/store/provider_account*.go server/internal/chain/client.go server/cmd/myference-server/runtime.go && go test ./server/internal/store ./server/internal/chain -count=1`

Commit: `git add server/internal/store server/internal/chain/client.go server/cmd/myference-server/runtime.go && git commit -m "feat(server): project account-owned provider offers"`

### Task 2: Chain-verified provider-action drafts

**Files:**
- Create: `server/internal/api/provider_action.go`
- Create: `server/internal/api/provider_action_test.go`
- Delete: `server/internal/api/provider_activation.go`
- Delete: `server/internal/api/provider_activation_test.go`
- Modify: `server/cmd/myference-server/main.go`
- Modify: `server/cmd/myference-server/main_test.go`

- [ ] **Step 1: Write failing draft-store tests**

Cover publish, deposit, exit request, and exit finalization. Require account binding, immutable values, baseline state, fifteen-minute expiry, bounded offer batches, and cross-account reads that do not remove the original draft.

```go
draft, err := store.Create(ActionInput{Kind: ActionPublishOffer, MachineID: "machine", AccountID: "account", WalletAddress: wallet, Offers: offers}, baseline)
if err != nil || draft.Status != ActionPendingWallet { t.Fatalf("draft=%+v err=%v", draft, err) }
if _, err := store.GetForAccount(draft.ID, "other"); !errors.Is(err, ErrActionNotFound) { t.Fatal(err) }
if _, err := store.GetForAccount(draft.ID, "account"); err != nil { t.Fatal("cross-account read removed draft") }
```

- [ ] **Step 2: Verify RED**

Run: `go test ./server/internal/api -run 'TestProviderAction' -count=1 -v`

Expected: FAIL because provider actions are absent.

- [ ] **Step 3: Implement typed drafts and validation**

Use these public kinds and states:

```go
const (
	ActionPublishOffer = "publish_offer"
	ActionDepositCollateral = "deposit_collateral"
	ActionRequestCollateralExit = "request_collateral_exit"
	ActionFinalizeCollateralExit = "finalize_collateral_exit"
	ActionPendingWallet = "pending_wallet"
	ActionPendingChain = "pending_chain"
	ActionConfirmed = "confirmed"
)
```

Publish inputs contain exact offer identity and decimal wei rates. Deposit inputs contain one positive decimal wei amount. Exit actions accept no amount or offers. Store only public transaction hashes supplied by the browser; never store credentials or keys.

- [ ] **Step 4: Add repository-backed evidence verification**

Define a small verifier callback injected into the handler. It compares current indexed state with the creation baseline:

- publish: newest account-wallet version is greater than baseline and every hash/rate equals the draft;
- deposit: provider bond is at least baseline plus the draft amount;
- request exit: baseline exit timestamp was zero and current timestamp is nonzero;
- finalize exit: baseline exit timestamp was nonzero and current exit timestamp and bond are zero.

The browser submission endpoint changes state to `pending_chain`; GET calls the verifier and returns `confirmed` only on matching indexed evidence.

- [ ] **Step 5: Mount the authenticated endpoints**

Mount:

```text
POST /api/provider/actions
GET  /api/provider/actions/{id}
POST /api/provider/actions/{id}/submitted
GET  /api/provider/account
GET  /api/provider/machines/{machine}/offer-versions
```

Machine bearer authentication creates actions and reads machine versions. Matching browser authentication reads/submits actions and reads the provider account. Return not found for ownership mismatches.

- [ ] **Step 6: Remove activation-specific endpoints and verify**

Update the root-routing test and remove the now-unused activation store instead of maintaining two draft systems.

Run: `gofmt -w server/internal/api server/cmd/myference-server && go test ./server/internal/api ./server/cmd/myference-server -count=1`

Commit: `git add server/internal/api server/cmd/myference-server && git commit -m "feat(server): verify provider wallet actions from chain state"`

### Task 3: CLI provider operations service and commands

**Files:**
- Replace: `cli/internal/account/activation.go`
- Replace: `cli/internal/account/activation_test.go`
- Create: `cli/internal/providerops/service.go`
- Create: `cli/internal/providerops/service_test.go`
- Modify: `cli/cmd/myference/main.go`
- Modify: `cli/cmd/myference/main_test.go`

- [ ] **Step 1: Write failing API-client and money tests**

Use a custom `http.RoundTripper` so tests require no listener. Verify bearer authentication, browser action URL generation, bounded polling, and that JSON never contains a backend secret.

Test exact MON parsing:

```go
for input, want := range map[string]string{
	"1": "1000000000000000000",
	"0.000000000000000001": "1",
	"5.25": "5250000000000000000",
} {
	got, err := ParseMON(input)
	if err != nil || got != want { t.Fatalf("ParseMON(%q)=%q err=%v", input, got, err) }
}
```

Reject negatives, exponents, more than eighteen decimals, empty values, and values beyond 256 bits.

- [ ] **Step 2: Verify RED**

Run: `go test ./cli/internal/account ./cli/internal/providerops -count=1 -v`

Expected: FAIL because the provider action client and service do not exist.

- [ ] **Step 3: Implement the minimal service**

The account client exposes `ProviderAccount`, `CreateProviderAction`, `ProviderAction`, and `MachineOfferVersions`. The provider operations service accepts config, machine credential loading, config saving, browser opening, and time functions as callbacks.

Implement:

```go
func (s Service) Publish(ctx context.Context, backend config.Backend, rates Rates) error
func (s Service) Deposit(ctx context.Context, amountMON string) error
func (s Service) RequestExit(ctx context.Context) error
func (s Service) FinalizeExit(ctx context.Context) error
func (s Service) Account(ctx context.Context) (account.ProviderAccount, error)
func (s Service) SyncVersions(ctx context.Context) (bool, error)
```

Mutating methods create a draft, open `/provider/approve?action=<id>`, poll until indexed confirmation, and atomically apply confirmed offer versions. A browser-open error returns the full copyable URL.

- [ ] **Step 4: Add command parsing**

Add `offer` and `collateral` to the existing command switch. Require explicit backend/offer and pricing flags for publication. `--no-browser` prints the URL while continuing to poll. Keep exact usage errors and avoid interactive prompts in command mode.

- [ ] **Step 5: Verify and commit**

Run: `gofmt -w cli/internal/account cli/internal/providerops cli/cmd/myference && go test ./cli/internal/account ./cli/internal/providerops ./cli/cmd/myference -count=1`

Commit: `git add cli/internal/account cli/internal/providerops cli/cmd/myference && git commit -m "feat(cli): manage offers and collateral"`

### Task 4: Offers and collateral TUI screens

**Files:**
- Modify: `cli/internal/tui/model.go`
- Modify: `cli/internal/tui/model_test.go`
- Modify: `cli/cmd/myference/main.go`

- [ ] **Step 1: Write failing reducer tests**

Require Home to expose Providers, Offers & Pricing, Collateral, Live Status, and Quit. Cover offer selection, metering-aware disabled fields, rate review, collateral deposit, exit-state actions, pending wallet/chain messages, cancel behavior, and masked secrets.

```go
model := NewModel(deps, candidates)
model, _ = model.HandleKey("down")
model, _ = model.HandleKey("enter")
if model.Screen() != ScreenOffers || !strings.Contains(model.ViewText(), "Not published") {
	t.Fatalf("screen=%v view=%q", model.Screen(), model.ViewText())
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./cli/internal/tui -run 'TestOffer|TestCollateral|TestHome' -count=1 -v`

Expected: FAIL because the new screens and dependencies are absent.

- [ ] **Step 3: Extend the state machine without duplicating services**

Add `ScreenOffers`, `ScreenPricing`, and `ScreenCollateral`. Inject account loading and provider-action callbacks. Forms collect presentation strings only; parsing and validation stay in `providerops.Service`. Render exact MON review values returned by the service and one actionable error at a time.

- [ ] **Step 4: Wire interactive dependencies**

Construct the provider operations service once in `runInteractive`. Refresh provider account state when entering Offers or Collateral and after a confirmed action. New offer publication uses configured backend identity, then the existing foreground start path.

- [ ] **Step 5: Verify and commit**

Run: `gofmt -w cli/internal/tui cli/cmd/myference && go test ./cli/internal/tui ./cli/cmd/myference -count=1`

Commit: `git add cli/internal/tui cli/cmd/myference && git commit -m "feat(cli): add offer and collateral screens"`

### Task 5: Compatible version synchronization

**Files:**
- Modify: `cli/internal/providerops/service.go`
- Modify: `cli/internal/providerops/service_test.go`
- Modify: `cli/cmd/myference/main.go`
- Modify: `cli/cmd/myference/main_test.go`

- [ ] **Step 1: Write failing synchronization tests**

Require a newer compatible server version to update only the matching enabled backend, preserve every other field, save atomically, and trigger daemon reload through the existing config watcher. Require equal, older, missing, and identity-incompatible versions to leave config unchanged.

- [ ] **Step 2: Verify RED**

Run: `go test ./cli/internal/providerops ./cli/cmd/myference -run 'TestSync|TestWatchOffer' -count=1 -v`

Expected: FAIL because the serve lifecycle does not poll versions.

- [ ] **Step 3: Add the bounded watcher**

Start a fifteen-second watcher beside `watchBackendConfig` in `runServe`. It calls `SyncVersions`, logs sanitized transient failures, and relies on atomic config replacement plus the existing watcher to reload daemon backends. Stop it with the serve context and never stop serving the current confirmed version on sync failure.

- [ ] **Step 4: Verify and commit**

Run: `gofmt -w cli/internal/providerops cli/cmd/myference && go test -race ./cli/internal/providerops ./cli/internal/provider ./cli/cmd/myference -count=1`

Commit: `git add cli/internal/providerops cli/cmd/myference && git commit -m "feat(cli): synchronize published offer versions"`

### Task 6: Focused web account console and approval page

**Files:**
- Modify: `web/src/lib/api.ts`
- Create: `web/src/features/provider/ProviderApproval.tsx`
- Modify: `web/src/features/provider/ProviderConsole.tsx`
- Replace: `web/src/features/provider/Offers.tsx`
- Modify: `web/src/features/provider/provider.test.tsx`
- Modify: `web/src/app/App.tsx`
- Modify: `web/src/app/DashboardShell.tsx`
- Modify: `web/src/features/onboarding/OnboardingFlow.tsx`
- Modify: `web/src/styles/global.css`

- [ ] **Step 1: Write failing normal-console tests**

Render a provider account containing one editable offer and assert that collateral and locked offer pricing are present while machine names, backend/model dropdowns, discovered models, deployment controls, and new-offer controls are absent. With no offers, require the `myference` instruction.

- [ ] **Step 2: Write failing approval-page tests**

For each action kind, render exact immutable values, reject a connected wallet different from `wallet_address`, submit transaction hashes, poll status, and show terminal-return copy only after server confirmation. Verify a multi-offer draft sequences one wallet transaction per offer.

- [ ] **Step 3: Verify RED**

Run: `npm test --prefix web -- --run src/features/provider/provider.test.tsx`

Expected: FAIL because the current console still renders discovered backends and the approval page is absent.

- [ ] **Step 4: Add typed APIs and focused components**

Add `ProviderAccountAPI` and `ProviderActionAPI`. Refactor `Offers` into an existing-offer price editor that locks identity and calls `publishOffer` only for a selected returned offer. `ProviderConsole` loads provider-account data rather than flattening `operations.machines`.

Route `/provider/approve?action=<id>` directly to `ProviderApproval`. Use the existing `ViemMarketWriter`; before simulation require the injected wallet address to equal the draft wallet address. The page has no hosting configuration controls.

- [ ] **Step 5: Remove web hosting controls and update copy**

Remove the Machines component from provider pages and remove all discovered-backend language from dashboard, onboarding, and empty states. Keep collateral, earnings, existing-offer history, and repricing.

- [ ] **Step 6: Verify and commit**

Run: `npm test --prefix web -- --run && npm run build --prefix web`

Commit: `git add web/src && git commit -m "feat(web): focus provider console on account operations"`

### Task 7: Documentation and full verification

**Files:**
- Modify: `README.md`
- Modify: `web/src/app/DocsPage.tsx`
- Modify: `docs/superpowers/plans/2026-08-09-cli-provider-operations.md`

- [ ] **Step 1: Document the final split**

Make bare `myference` the provider entry point. Document `offer` and `collateral` commands, browser wallet approval, automatic compatible-version synchronization, and that new offers cannot be created in the web client.

- [ ] **Step 2: Run complete verification**

Run:

```text
gofmt -w cli server
go test ./...
go test -race ./cli/internal/account ./cli/internal/providerops ./cli/internal/tui ./cli/internal/provider ./server/internal/api
go vet ./...
go build ./...
npm test --prefix web -- --run
npm run build --prefix web
git diff --check
```

Expected: every command passes with no warnings or leaked secrets.

- [ ] **Step 3: Mark this plan complete and commit**

Commit: `git add README.md web/src/app/DocsPage.tsx docs/superpowers/plans/2026-08-09-cli-provider-operations.md && git commit -m "docs: explain CLI-owned provider workflow"`
