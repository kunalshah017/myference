# Cross-device Offer Attachment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fetch wallet-owned offers in the hosting TUI and let a local backend attach to a compatible offer created by another machine without renaming the backend or publishing a duplicate.

**Architecture:** Add an optional public `offer_id` to local backend configuration with a backward-compatible fallback to `name`. Centralize compatibility and attachment in `providerops.Service`, expose it through both `offer attach` and the TUI, and use the effective offer identity when advertising capacity while retaining local names for credentials and lifecycle commands.

**Tech Stack:** Go, Bubble Tea, existing Myference account API/config/provider packages, standard `flag` CLI parsing, Go tests.

---

### Task 1: Separate local backend and public offer identities

**Files:**
- Modify: `cli/internal/config/config.go`
- Modify: `cli/internal/config/config_test.go`
- Modify: `cli/cmd/myference/main.go`
- Modify: `cli/cmd/myference/main_test.go`

- [ ] **Step 1: Write failing configuration and capacity tests**

Add tests proving an absent offer ID falls back to the local name and a distinct offer ID is advertised without changing backend lookup or credentials:

```go
func TestBackendEffectiveOfferIDFallsBackToName(t *testing.T) {
	if got := (Backend{Name: "local"}).EffectiveOfferID(); got != "local" {
		t.Fatalf("effective offer=%q", got)
	}
	if got := (Backend{Name: "local", OfferID: "wallet-offer"}).EffectiveOfferID(); got != "wallet-offer" {
		t.Fatalf("effective offer=%q", got)
	}
}
```

Extend the capacity test with `Name: "ollama-qwen"`, `OfferID: "local-qwen"`, and assert `OfferID == "local-qwen"` plus the hash of `local-qwen`.

- [ ] **Step 2: Run the focused tests and verify RED**

Run: `go test ./cli/internal/config ./cli/cmd/myference -run 'TestBackendEffectiveOfferID|TestOfferCapacity' -count=1`

Expected: FAIL because `OfferID`, `EffectiveOfferID`, and distinct advertisement do not exist.

- [ ] **Step 3: Implement the minimal configuration identity**

Add:

```go
type Backend struct {
	Name         string `json:"name"`
	OfferID      string `json:"offer_id,omitempty"`
	Kind         string `json:"kind"`
	URL          string `json:"url"`
	Model        string `json:"model"`
	PriceVersion uint64 `json:"price_version,omitempty"`
	Enabled      bool   `json:"enabled"`
	Image        string `json:"image,omitempty"`
}

func (b Backend) EffectiveOfferID() string {
	if b.OfferID != "" {
		return b.OfferID
	}
	return b.Name
}
```

Update `offerCapacity` to use `item.EffectiveOfferID()` for `OfferID` and `OfferHash`. Update `discoverBackends` to key the daemon backend map by the effective offer ID after credentials and the runtime backend have been resolved using the unchanged local name.

- [ ] **Step 4: Run focused and package tests and verify GREEN**

Run: `go test ./cli/internal/config ./cli/cmd/myference -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the identity boundary**

```bash
git add cli/internal/config/config.go cli/internal/config/config_test.go cli/cmd/myference/main.go cli/cmd/myference/main_test.go
git commit -m "feat(cli): separate backend and offer identities"
```

### Task 2: Validate and persist account-owned offer attachment

**Files:**
- Modify: `cli/internal/providerops/service.go`
- Modify: `cli/internal/providerops/service_test.go`

- [ ] **Step 1: Write failing service tests**

Cover successful attachment and each immutable mismatch with the real account types:

```go
func TestAttachPersistsCompatibleWalletOffer(t *testing.T) {
	backend := config.Backend{Name: "ollama-qwen2-5", Kind: "ollama", Model: "qwen2.5:0.5b", Enabled: true}
	offer := account.EditableOffer{OfferID: "local-qwen", Model: backend.Model, BackendKind: "ollama", Capabilities: []string{"stream", "text"}, MeteringMode: "tokens_and_compute", Version: 3}
	api := &providerAPIStub{account: account.ProviderAccount{Offers: []account.EditableOffer{offer}}}
	cfg := config.Config{MachineID: "machine", Backends: []config.Backend{backend}}
	var saved config.Config
	service := Service{API: api, Token: "token", LoadConfig: func() (config.Config, error) { return cfg, nil }, SaveConfig: func(value config.Config) error { saved = value; return nil }}
	if err := service.Attach(context.Background(), backend.Name, offer.OfferID); err != nil {
		t.Fatal(err)
	}
	if saved.Backends[0].Name != backend.Name || saved.Backends[0].OfferID != offer.OfferID || saved.Backends[0].PriceVersion != 3 {
		t.Fatalf("saved=%+v", saved.Backends[0])
	}
}
```

Table-test model, kind, capabilities, and metering mismatches and assert the config saver is not called.

- [ ] **Step 2: Run the providerops tests and verify RED**

Run: `go test ./cli/internal/providerops -run 'TestAttach' -count=1`

Expected: FAIL because `Attach` and compatibility validation do not exist.

- [ ] **Step 3: Implement attachment and shared compatibility**

Add:

```go
func Compatible(backend config.Backend, offer account.EditableOffer) bool {
	capabilities, metering := offerShape(backend)
	want := append([]string(nil), capabilities...)
	got := append([]string(nil), offer.Capabilities...)
	slices.Sort(want)
	slices.Sort(got)
	return backend.Model == offer.Model && backend.Kind == offer.BackendKind &&
		slices.Equal(want, got) && metering == offer.MeteringMode && offer.Version > 0
}

func (s Service) Attach(ctx context.Context, backendName, offerID string) error {
	if s.LoadConfig == nil || s.SaveConfig == nil {
		return errors.New("configuration persistence is unavailable")
	}
	cfg, err := s.LoadConfig()
	if err != nil { return err }
	backendIndex := slices.IndexFunc(cfg.Backends, func(item config.Backend) bool { return item.Name == backendName })
	if backendIndex < 0 { return fmt.Errorf("backend %q not found", backendName) }
	providerAccount, err := s.Account(ctx)
	if err != nil { return err }
	offerIndex := slices.IndexFunc(providerAccount.Offers, func(item account.EditableOffer) bool { return item.OfferID == offerID })
	if offerIndex < 0 { return fmt.Errorf("wallet offer %q not found", offerID) }
	if !Compatible(cfg.Backends[backendIndex], providerAccount.Offers[offerIndex]) {
		return fmt.Errorf("wallet offer %q is incompatible with backend %q", offerID, backendName)
	}
	cfg.Backends[backendIndex].OfferID = offerID
	cfg.Backends[backendIndex].PriceVersion = providerAccount.Offers[offerIndex].Version
	return s.SaveConfig(cfg)
}
```

Change `Publish` to submit and read versions using `backend.EffectiveOfferID()`. Change `SyncVersions` to look up each backend by its effective offer ID.

- [ ] **Step 4: Run providerops tests and verify GREEN**

Run: `go test ./cli/internal/providerops -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the attachment service**

```bash
git add cli/internal/providerops/service.go cli/internal/providerops/service_test.go
git commit -m "feat(cli): attach compatible wallet offers"
```

### Task 3: Add command-line attachment parity

**Files:**
- Modify: `cli/cmd/myference/main.go`
- Modify: `cli/cmd/myference/main_test.go`

- [ ] **Step 1: Write failing command tests**

Extend offer command usage to include `attach`. Add dependency-injected tests asserting:

```text
myference offer attach --backend ollama-qwen2-5-0-5b --offer local-qwen
```

calls `Service.Attach`, prints `Attached local-qwen to ollama-qwen2-5-0-5b.`, and rejects missing flags before any API call.

- [ ] **Step 2: Run the command tests and verify RED**

Run: `go test ./cli/cmd/myference -run 'TestOfferAttach|TestCommandUsage' -count=1`

Expected: FAIL with unknown offer action or old usage.

- [ ] **Step 3: Implement `offer attach`**

Add an `--offer` flag and branch:

```go
case "attach":
	if *backendName == "" || *offerID == "" {
		return errors.New("offer attach requires --backend and --offer")
	}
	if err := service.Attach(ctx, *backendName, *offerID); err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "Attached %s to %s.\n", *offerID, *backendName)
	return err
```

Update root and offer usage strings to `offer <publish|attach|list|sync>`.

- [ ] **Step 4: Run command tests and verify GREEN**

Run: `go test ./cli/cmd/myference -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the command**

```bash
git add cli/cmd/myference/main.go cli/cmd/myference/main_test.go
git commit -m "feat(cli): add offer attach command"
```

### Task 4: Fetch and attach wallet offers in the TUI

**Files:**
- Modify: `cli/internal/tui/model.go`
- Modify: `cli/internal/tui/model_test.go`
- Modify: `cli/cmd/myference/main.go`

- [ ] **Step 1: Write failing TUI tests**

Add tests that enter Offers from Home, execute the returned account command, apply its message, and assert a compatible wallet offer appears:

```go
func TestOffersFetchAndAttachCompatibleWalletOffer(t *testing.T) {
	backend := config.Backend{Name: "ollama-qwen", Kind: "ollama", Model: "qwen2.5:0.5b", Enabled: true}
	attached := ""
	model := NewModel(Dependencies{
		Backends: []config.Backend{backend},
		Account: func(context.Context) (account.ProviderAccount, error) {
			return account.ProviderAccount{Offers: []account.EditableOffer{{OfferID: "local-qwen", Model: backend.Model, BackendKind: backend.Kind, Capabilities: []string{"stream", "text"}, MeteringMode: "tokens_and_compute", Version: 1}}}, nil
		},
		Attach: func(_ context.Context, _, offerID string) error { attached = offerID; return nil },
	}, nil)
	model.cursor = 1
	var command tea.Cmd
	model, command = model.HandleKey("enter")
	message := command().(accountMsg)
	model.applyAccount(message)
	model, command = model.HandleKey("enter")
	result := command().(providerOperationMsg)
	if result.err != nil || attached != "local-qwen" {
		t.Fatalf("attached=%q err=%v", attached, result.err)
	}
}
```

Also test multiple compatible offers require `ScreenOfferAttach`, attached rows open repricing, incompatible account offers remain visible as unavailable, and fetch errors are retryable without losing local rows.

- [ ] **Step 2: Run TUI tests and verify RED**

Run: `go test ./cli/internal/tui -run 'TestOffersFetch|TestOfferAttach' -count=1`

Expected: FAIL because the Offers screen neither fetches the account nor supports attachment selection.

- [ ] **Step 3: Implement account-backed offer states and chooser**

Add `Attach func(context.Context, string, string) error` to `Dependencies`, `ScreenOfferAttach`, and helpers that derive compatible account offers via `providerops.Compatible`.

When Home opens Offers, set `busy` and return `accountCommand()`. Render each backend as one of:

```text
ollama-qwen  qwen2.5:0.5b  Attach local-qwen v1
ollama-qwen  qwen2.5:0.5b  Public as local-qwen v1
ollama-qwen  qwen2.5:0.5b  Not published
```

For one compatible unattached offer, Enter calls `Attach`. For several, Enter opens `ScreenOfferAttach` and a second Enter attaches the selected offer. On success, reload backends and account state; on failure, retain the current screen and show the error. Keep Enter on an already attached/public backend mapped to the existing repricing form.

Wire `Dependencies.Attach` to `providerService.Attach` in `runHostingUI`.

- [ ] **Step 4: Run TUI and CLI tests and verify GREEN**

Run: `go test ./cli/internal/tui ./cli/cmd/myference -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the TUI flow**

```bash
git add cli/internal/tui/model.go cli/internal/tui/model_test.go cli/cmd/myference/main.go
git commit -m "feat(tui): reuse wallet offers across devices"
```

### Task 5: Verify, document, release, and install

**Files:**
- Modify: `README.md`
- Modify: `web/src/app/DocsPage.tsx`

- [ ] **Step 1: Update user-facing command documentation**

Document `myference offer attach --backend <local-name> --offer <wallet-offer-id>` and explain that attachment reuses an immutable compatible wallet offer without a transaction.

- [ ] **Step 2: Run formatting and focused regression tests**

Run: `gofmt -w cli/internal/config/config.go cli/internal/config/config_test.go cli/internal/providerops/service.go cli/internal/providerops/service_test.go cli/internal/tui/model.go cli/internal/tui/model_test.go cli/cmd/myference/main.go cli/cmd/myference/main_test.go`

Run: `go test ./cli/internal/config ./cli/internal/providerops ./cli/internal/tui ./cli/cmd/myference -count=1`

Expected: PASS.

- [ ] **Step 3: Run full release verification**

Run: `make verify`

Expected: Go tests/vet/build, 20 Forge tests, 58 web tests, web lint/build, and shell syntax checks all pass.

- [ ] **Step 4: Verify cross-platform compilation**

Run: `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -exec=/usr/bin/true ./cli/...`

Expected: PASS.

- [ ] **Step 5: Commit documentation, push, and publish the next alpha**

Commit any documentation changes, push `main`, build release artifacts with the next available tag after `v0.2.0-alpha.7`, verify checksums and embedded commit, push the annotated tag, and publish all platform archives plus `SHA256SUMS`.

- [ ] **Step 6: Install and run production smoke tests on this Mac**

Install the exact new tag into `/Users/kunal/.local/bin`. Verify the binary is Mach-O ARM64, its embedded version and commit match the release, `myference offer list` returns wallet offers, and the TUI shows `local-qwen v1` as attachable to local backend `ollama-qwen2-5-0-5b` without creating a provider action.
