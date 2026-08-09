# Interactive CLI Hosting Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Provide a full-screen `myference` hosting interface that discovers local and API-backed providers, configures several models, hands wallet approval to the browser, and controls a foreground provider session without requiring memorized commands.

**Architecture:** Put discovery and idempotent configuration in a reusable `cli/internal/host` service, keep native model runners in focused backend packages, and implement the terminal as a Bubble Tea v2 state machine. The no-argument command launches the TUI only on a real terminal; existing commands remain compatible and use the same host operations where practical.

**Tech Stack:** Go 1.24, standard library HTTP/process/terminal primitives, Bubble Tea v2.0.3, existing Ollama/OpenAI/config/credential/provider packages, Claude Code JSON output, React/Vitest only for the browser activation handoff.

---

### Task 1: Concurrent provider catalog

**Files:**
- Create: `cli/internal/host/discovery.go`
- Create: `cli/internal/host/discovery_test.go`

- [ ] **Step 1: Write failing discovery tests**

Define fake detectors and require results to arrive independently, preserve successful candidates when another detector fails, and generate stable identities.

```go
func TestDiscoverKeepsIndependentResults(t *testing.T) {
	results := Discover(context.Background(), []Detector{
		DetectorFunc(func(context.Context) Result { return Result{Source: "ollama", Candidates: []Candidate{{Kind: "ollama", Model: "qwen"}}} }),
		DetectorFunc(func(context.Context) Result { return Result{Source: "claude", Err: errors.New("login required")} }),
	})
	if got := collect(results); len(got) != 2 || len(got[0].Candidates)+len(got[1].Candidates) != 1 { t.Fatalf("results=%+v", got) }
}
```

- [ ] **Step 2: Run the focused test and verify RED**

Run: `go test ./cli/internal/host -run TestDiscover -v`

Expected: FAIL because the package does not exist.

- [ ] **Step 3: Implement the minimal catalog**

```go
type Candidate struct { ID, Kind, Name, URL, Model, Digest, State, Hint string; Size int64 }
type Result struct { Source string; Candidates []Candidate; Err error }
type Detector interface { Detect(context.Context) Result }

func Discover(ctx context.Context, detectors []Detector) <-chan Result {
	out := make(chan Result)
	var wg sync.WaitGroup
	for _, detector := range detectors { wg.Add(1); go func(d Detector) { defer wg.Done(); select { case out <- d.Detect(ctx): case <-ctx.Done(): } }(detector) }
	go func() { wg.Wait(); close(out) }()
	return out
}
```

Implement Ollama detection through the existing client, CLI detection through injected `exec.LookPath`, Codex presets, and Claude presets `sonnet`, `opus`, and `haiku`. Use a five-second timeout per detector.

- [ ] **Step 4: Add OpenAI model-catalog tests and implementation**

Use the existing OpenAI-compatible client for both the fixed `https://api.openai.com` path and custom base URLs. Sort and deduplicate returned model IDs. Return a typed `ErrModelCatalogUnavailable` for compatible providers so the TUI can offer manual entry; authentication errors remain fatal.

- [ ] **Step 5: Verify GREEN and commit**

Run: `gofmt -w cli/internal/host/*.go && go test ./cli/internal/host -v`

Expected: PASS.

Commit: `git add cli/internal/host && git commit -m "feat(cli): discover hosting providers"`

### Task 2: Idempotent multi-provider configuration

**Files:**
- Create: `cli/internal/host/configure.go`
- Create: `cli/internal/host/configure_test.go`
- Modify: `cli/cmd/myference/main.go`
- Modify: `cli/cmd/myference/main_test.go`

- [ ] **Step 1: Write failing staging tests**

Require selected identities to create or update enabled backends, unselected existing backends to become disabled, existing names and price versions to survive, API credentials to use `machineID/backendName`, and new credentials to be deleted if config saving fails.

```go
selection := []Selection{{Candidate: Candidate{Kind:"openai", Name:"OpenAI", URL:"https://api.openai.com", Model:"gpt-test"}, Secret:"secret"}}
updated, err := Apply(context.Background(), current, selection, Store{Save: save, Delete: remove}, failSave)
if err == nil || updated.Backends[0] != current.Backends[0] || removed != "machine/openai-gpt-test" { t.Fatalf("updated=%+v removed=%q err=%v", updated, removed, err) }
```

- [ ] **Step 2: Verify RED**

Run: `go test ./cli/internal/host -run 'TestApply|TestStable' -v`

Expected: FAIL because configuration staging is absent.

- [ ] **Step 3: Implement stable selection and rollback**

Expose `StableID(Candidate)`, `BackendName(Candidate)`, and `Apply`. Reuse config atomic saving and the existing credential service callbacks. Do not persist a secret in `config.Backend`.

- [ ] **Step 4: Route legacy Ollama host setup through the service**

Keep `configureLocalHost` as a compatibility wrapper and preserve its current errors and default-first behavior for explicit `myference host` scripts.

- [ ] **Step 5: Verify GREEN and commit**

Run: `gofmt -w cli/internal/host/*.go cli/cmd/myference/*.go && go test ./cli/internal/host ./cli/cmd/myference -v`

Expected: PASS.

Commit: `git add cli/internal/host cli/cmd/myference && git commit -m "feat(cli): configure multiple hosting backends"`

### Task 3: Native model-only Claude backend

**Files:**
- Create: `cli/internal/backend/claude/claude.go`
- Create: `cli/internal/backend/claude/claude_test.go`
- Modify: `cli/cmd/myference/main.go`
- Modify: `cli/cmd/myference/main_test.go`

- [ ] **Step 1: Write failing runner tests**

Inject a fake executable and require arguments equivalent to:

```text
claude -p --model sonnet --output-format json --no-session-persistence --safe-mode --strict-mcp-config --tools "" --disallowedTools mcp__* --max-turns 1
```

Require an empty temporary working directory, prompt on stdin, rejection of workspace files, final result text only, token usage from JSON, timeout handling, and no inherited API-key environment variables.

- [ ] **Step 2: Verify RED**

Run: `go test ./cli/internal/backend/claude -v`

Expected: FAIL because the package does not exist.

- [ ] **Step 3: Implement minimal native execution**

Parse one bounded JSON result object containing `result`, `usage.input_tokens`, and `usage.output_tokens`. Reject empty output, missing usage, over-limit output, nonzero exit, oversized output, and any reported tool-use content. Report elapsed compute milliseconds.

- [ ] **Step 4: Register image-less Claude**

Route `config.Backend{Kind:"claude", Image:""}` to the native runner without loading a backend credential. Preserve the existing digest-pinned image path when `Image` and `Secret` are supplied.

- [ ] **Step 5: Verify GREEN and commit**

Run: `gofmt -w cli/internal/backend/claude/*.go cli/cmd/myference/*.go && go test ./cli/internal/backend/claude ./cli/cmd/myference -v`

Expected: PASS.

Commit: `git add cli/internal/backend/claude cli/cmd/myference && git commit -m "feat(cli): add native Claude provider"`

### Task 4: Testable full-screen hosting TUI

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Create: `cli/internal/tui/model.go`
- Create: `cli/internal/tui/model_test.go`
- Create: `cli/internal/tui/run.go`

- [ ] **Step 1: Add Bubble Tea v2**

Run: `go get charm.land/bubbletea/v2@v2.0.3 charm.land/bubbles/v2@v2.0.0`

Expected: modules are added without changing the Go language version.

- [ ] **Step 2: Write failing state-machine tests**

Drive typed messages and key messages directly. Require Home → Providers → Review → Status navigation, multi-selection with Space, API form routing, browser-pending state, resize support, back/cancel behavior, masked secrets, and quit confirmation while running.

```go
m := NewModel(Dependencies{}, []host.Candidate{{ID:"ollama/qwen", Name:"Ollama", Model:"qwen"}})
m, _ = update(m, Key("enter"))
m, _ = update(m, Key("space"))
if !m.Selected("ollama/qwen") || strings.Contains(m.ViewText(), "api-secret") { t.Fatalf("model=%+v", m) }
```

- [ ] **Step 3: Verify RED**

Run: `go test ./cli/internal/tui -v`

Expected: FAIL because the package does not exist.

- [ ] **Step 4: Implement the reducer and views**

Use a small `Screen` enum and a single model. Render Home, Providers, API Setup, Review & Start, and Live Status. Use Bubble text input only for forms; implement provider checkbox navigation directly to keep selection behavior explicit. Every view includes key help and renders as an alternate-screen `tea.View`.

- [ ] **Step 5: Implement asynchronous commands**

Convert discovery channel results, login completion, model catalog results, configuration results, browser activation, runtime snapshots, and runtime termination into typed Bubble Tea messages. Clear secret text inputs on success, cancel, and error.

- [ ] **Step 6: Verify GREEN and commit**

Run: `gofmt -w cli/internal/tui/*.go && go test ./cli/internal/tui -v`

Expected: PASS.

Commit: `git add go.mod go.sum cli/internal/tui && git commit -m "feat(cli): add interactive hosting TUI"`

### Task 5: No-argument entry point and foreground lifecycle

**Files:**
- Modify: `cli/cmd/myference/main.go`
- Modify: `cli/cmd/myference/main_test.go`
- Modify: `README.md`
- Modify: `web/src/app/DocsPage.tsx`

- [ ] **Step 1: Write failing entry tests**

Extract and test `entryMode(args, stdinTTY, stdoutTTY)`. Empty interactive input returns TUI mode, empty non-interactive input returns usage mode, and any argument returns command mode. Inject a TUI runner to prove `main` does not enter the command switch.

- [ ] **Step 2: Verify RED**

Run: `go test ./cli/cmd/myference -run 'TestEntry|TestNoArgument' -v`

Expected: FAIL because entry selection is absent.

- [ ] **Step 3: Wire the TUI**

Use character-device checks for stdin/stdout, call `tui.Run` for interactive no-argument invocation, and provide dependencies that reuse login, discovery, host configuration, browser opening, provider status, and `runServe`. Keep all existing command behavior.

- [ ] **Step 4: Document both interaction styles**

Make bare `myference` the recommended host path. Keep exact `myference host`, `backend`, `serve`, `status`, and service examples for automation and recovery.

- [ ] **Step 5: Verify GREEN and commit**

Run: `gofmt -w cli/cmd/myference/*.go && go test ./cli/cmd/myference ./cli/internal/tui ./cli/internal/host -v && npm test --prefix web -- --run`

Expected: PASS.

Commit: `git add cli/cmd/myference README.md web/src/app/DocsPage.tsx && git commit -m "feat(cli): launch hosting UI by default"`

### Task 6: Browser activation handoff and final verification

**Files:**
- Create: `server/internal/api/provider_activation.go`
- Create: `server/internal/api/provider_activation_test.go`
- Modify: `server/internal/router/router.go`
- Modify: `web/src/features/provider/ProviderConsole.tsx`
- Modify: `web/src/features/provider/provider.test.tsx`
- Modify: `cli/internal/tui/model.go`
- Modify: `cli/internal/tui/model_test.go`

- [ ] **Step 1: Write failing activation API tests**

Require an authenticated machine to create a short-lived activation draft, an authenticated matching account to read it, and the machine to poll pending/confirmed/failed state. Reject cross-account access and expire unused drafts.

- [ ] **Step 2: Verify RED**

Run: `go test ./server/internal/api -run TestProviderActivation -v`

Expected: FAIL because the endpoint is absent.

- [ ] **Step 3: Implement the bounded in-memory draft store and endpoints**

Add `POST /api/provider/activations`, `GET /api/provider/activations/{id}`, and `POST /api/provider/activations/{id}/complete`. Store only offer metadata and rates, never provider API credentials. Bind drafts to machine and account IDs and expire them after fifteen minutes.

- [ ] **Step 4: Implement browser completion**

Load an activation ID from `/host?activation=<id>`, reuse the existing wallet and `publishOffer` flow, and post confirmed offer versions back to the server. Show pending, rejected, failed, and confirmed states.

- [ ] **Step 5: Resume the TUI after confirmation**

Poll with bounded backoff, preserve a copyable URL if browser opening fails, update backend price versions only from confirmed results, and transition to foreground hosting.

- [ ] **Step 6: Run full verification**

Run: `gofmt -w cli server && go test ./... && go test -race ./cli/internal/host ./cli/internal/tui ./cli/internal/provider && go vet ./... && go build ./... && npm test --prefix web -- --run && npm run build --prefix web && git diff --check`

Expected: PASS with no warnings, leaked secrets, or formatting errors.

- [ ] **Step 7: Commit**

Commit: `git add cli server web README.md go.mod go.sum && git commit -m "feat: complete terminal-first hosting workflow"`
