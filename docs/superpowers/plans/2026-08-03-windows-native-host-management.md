# Provider-Owned Windows Host Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the existing multi-backend provider daemon own safe Windows host preparation and recovery while preserving on-demand `backend add|start|stop|remove` behavior and removing obsolete legacy gateway code only after real acceptance.

**Architecture:** The shared `config.Config.Backends` list remains the only source of backend/model truth. One `serve` process advertises every enabled backend as a separate offer, prepares enabled local Ollama models before initial or live capacity publication, applies one journaled Windows tuning session around the daemon, and restores it on shutdown. Windows-only doctor, status, dashboard, focus, headless, and emergency recovery commands are adapters around that same provider lifecycle; none create a LAN listener or separate provider service.

**Tech Stack:** Go standard library, current provider WebSocket protocol, loopback Ollama HTTP API, Windows power/process/CIM/Scheduled Tasks APIs, PowerShell 5+ only for fixed non-interpolated Windows queries and installer tasks, GitHub Actions Windows runner.

## Global Constraints

- One `serve` process owns every enabled backend on the laptop; Windows must not create one service per backend.
- `backend add|list|start|stop|remove|version` and the shared provider config remain cross-platform and authoritative.
- An enabled backend is never advertised until discovery and platform preparation succeed; a failed live reload retains the last good daemon capacity.
- Windows host policy contains no backend name, model, credential, price, LAN, firewall, endpoint-discovery, or listener fields.
- Provider tuning is applied once per `serve` session and restored on every graceful exit path.
- Focus never stops an unconfigured process/service and never stops Explorer; only explicit headless mode may replace Explorer.
- No command disables Defender, Windows Update, networking, drivers, authentication, logging, scheduling, or security services.
- All privileged mutations are journaled before execution, use explicit executable/argument arrays, roll back in reverse order, and preserve the journal after failed mandatory recovery.
- `myference windows restore` is idempotent and removes only Myference-owned tasks/state.
- No new HTTP/TCP listener is opened for Windows management, telemetry, or dashboard data.
- The public one-command installer remains `irm https://myference.xyz/install.ps1 | iex` and must install every runtime asset needed by Windows service/headless management beside the CLI.
- Legacy removal is forbidden until automated Windows CI passes and the real-machine acceptance document is fully checked off.

---

### Task 1: Align the Windows command and host-policy contract

**Files:**
- Modify: `cli/internal/platform/windows/config.go`
- Modify: `cli/internal/platform/windows/config_test.go`
- Modify: `cli/cmd/myference/main.go`
- Modify: `cli/cmd/myference/main_test.go`
- Modify: `cli/cmd/myference/platform_windows.go`

**Interfaces:**
- Consumes: `config.Config.Backends` and existing `runBackend` enable/disable behavior.
- Produces: `windows.Config`, `windows.ParseCommand([]string)`, and public actions `doctor|status|models|test|dashboard|focus|headless|restore`.

- [ ] Add failing tests proving Windows policy has no singular `PreloadModel`, backend/model ownership, LAN, or firewall fields; `ParseCommand` accepts `focus` and rejects `optimize`, `exclusive`, and legacy LAN actions.
- [ ] Run `go test ./cli/internal/platform/windows ./cli/cmd/myference -run 'Config|ParseCommand|WindowsCommand' -count=1` and confirm the new contract fails against the current code.
- [ ] Remove `PreloadModel` and the public `optimize` action, keep machine-level context/concurrency/power/priority and focus allowlists, and update CLI usage strings.
- [ ] Run the focused tests and `go test ./... -count=1`.
- [ ] Commit with `refactor: align Windows controls with provider lifecycle`.

### Task 2: Extend the atomic journal for provider sessions and focus overlays

**Files:**
- Modify: `cli/internal/platform/windows/journal.go`
- Modify: `cli/internal/platform/windows/journal_test.go`
- Modify: `cli/internal/platform/windows/journal_rename_windows.go`
- Modify: `cli/internal/platform/windows/journal_rename_other.go`

**Interfaces:**
- Consumes: `RecoveryJournal`, `JournalStore.Save`, `CompleteStage`, and `CompleteRecovery`.
- Produces: `JournalStore.Update(func(*RecoveryJournal) error) error`, `RemoveStages(prefixes ...string)`, `SessionKind`, `OwnerPID`, shell-presence state, nullable original Ollama environment values, stopped process paths/services, and installed Myference task names.

- [ ] Add failing real-filesystem tests for atomic update, failed-update preservation, provider-session ownership, focus data appended without replacing the active provider journal, prefix stage removal, nullable environment round-trip, and repeated recovery.
- [ ] Run `go test ./cli/internal/platform/windows -run Journal -count=1` and confirm failures are caused by missing update/overlay behavior.
- [ ] Implement atomic update with temp file, sync, Windows write-through replacement, strict Myference task validation, and state removal only after mandatory restoration succeeds.
- [ ] Run journal tests, the Windows package, and `git diff --check`.
- [ ] Commit with `feat: journal provider sessions and focus overlays`.

### Task 3: Prepare multiple enabled backends before capacity publication

**Files:**
- Modify: `cli/internal/platform/windows/ollama.go`
- Modify: `cli/internal/platform/windows/ollama_test.go`
- Modify: `cli/cmd/myference/main.go`
- Modify: `cli/cmd/myference/main_test.go`
- Modify: `cli/cmd/myference/platform_windows.go`
- Modify: `cli/cmd/myference/platform_windows_test.go`
- Modify: `cli/cmd/myference/platform_darwin.go`
- Modify: `cli/cmd/myference/platform_other.go`

**Interfaces:**
- Consumes: enabled `config.Backend` entries and `provider.Daemon.UpdateBackends`.
- Produces: `preparePlatformBackends(context.Context, config.Config) error`, `reloadBackends(context.Context, config.Config, *provider.Daemon) error`, `OllamaHostClient.InstalledModels`, `Preload`, and `GenerateTest`.

- [ ] Add failing loopback integration tests with two enabled Ollama offers proving both configured models are checked/prepared; add a live-reload test proving preparation runs before `UpdateBackends` and a failed preparation leaves the previous capacity unchanged.
- [ ] Add failing tests proving stopped backends are skipped and OpenAI/command-agent backends retain existing discovery behavior.
- [ ] Run `go test ./cli/cmd/myference ./cli/internal/platform/windows -run 'Prepare|Reload|Ollama|Backend' -count=1` and confirm expected failures.
- [ ] Refactor initial startup and `watchBackendConfig` through the same prepare-then-discover-then-update function; never advertise a failed new backend.
- [ ] Run focused tests, `go test ./... -count=1`, and Windows/macOS CLI builds.
- [ ] Commit with `feat: prepare live multi-backend capacity`.

### Task 4: Own Windows tuning for the lifetime of `serve`

**Files:**
- Create: `cli/internal/platform/windows/tuning.go`
- Create: `cli/internal/platform/windows/tuning_test.go`
- Create: `cli/internal/platform/windows/focus.go`
- Create: `cli/internal/platform/windows/focus_test.go`
- Create: `cli/internal/platform/windows/runner_windows.go`
- Modify: `cli/cmd/myference/main.go`
- Modify: `cli/cmd/myference/platform_windows.go`
- Modify: `cli/cmd/myference/platform_darwin.go`
- Modify: `cli/cmd/myference/platform_other.go`

**Interfaces:**
- Consumes: Task 2 journal and Task 3 backend preparation.
- Produces: `startPlatformProviderSession(context.Context, config.Config, io.Writer) (func() error, error)`, `StartProviderTuning`, `RestoreProviderTuning`, `StartFocus`, `RestoreFocus`, and a Windows runner using explicit argument arrays.

- [ ] Add failing planning tests for High Performance, native keep-awake, Ollama environment/concurrency, priority, AC policy, protected services, allowlist-only focus, no Explorer operation, and reverse rollback.
- [ ] Add failing orchestration tests proving the journal exists before the first mutation, a later failure restores earlier stages, graceful cleanup restores once, and an existing crash journal blocks a new session with an actionable `windows restore` message.
- [ ] Implement the native runner: `powercfg.exe` argument arrays, fixed CIM process discovery, native execution-state and priority APIs, `sc.exe`/`taskkill.exe` arrays, recorded executable restart, and nullable environment restoration.
- [ ] Wrap `runServe` with platform session start/cleanup; macOS and other platforms return a no-op cleanup. Implement `windows focus start|status|restore` as an overlay on an active provider journal.
- [ ] Run `go test ./cli/internal/platform/windows ./cli/cmd/myference -count=1`, full tests, vet, and Windows/macOS builds.
- [ ] Commit with `feat: own Windows tuning with provider sessions`.

### Task 5: Add live provider telemetry, status, and dashboard

**Files:**
- Modify: `cli/internal/provider/daemon.go`
- Modify: `cli/internal/provider/daemon_test.go`
- Create: `cli/internal/platform/windows/telemetry.go`
- Create: `cli/internal/platform/windows/telemetry_test.go`
- Create: `cli/internal/platform/windows/dashboard.go`
- Create: `cli/internal/platform/windows/dashboard_test.go`
- Modify: `cli/cmd/myference/main.go`
- Modify: `cli/cmd/myference/platform_windows.go`

**Interfaces:**
- Produces: `provider.StatusSnapshot` with connection, start time, requests, input/output tokens, compute milliseconds, and per-offer health; atomic `<config>.status.json`; Windows host telemetry parsers; `windows status [--json]`; and `windows dashboard` polling the status file without a listener.

- [ ] Add failing daemon tests for concurrent counters, backend reload health, and immutable status snapshots.
- [ ] Add failing parser/render tests for CPU/RAM, battery, NVIDIA CSV, loaded Ollama models, scheduled provider state, provider status JSON, and dashboard fields for all enabled offers.
- [ ] Increment counters only from measured completed work, atomically write the local status snapshot, and remove stale status on shutdown.
- [ ] Implement status/dashboard polling with bounded refresh and `Q` closing only the viewer.
- [ ] Run focused tests, full tests, race tests for provider/platform packages, and builds.
- [ ] Commit with `feat: expose multi-backend Windows telemetry`.

### Task 6: Add provider-owned headless mode and universal recovery

**Files:**
- Create: `cli/internal/platform/windows/headless.go`
- Create: `cli/internal/platform/windows/headless_test.go`
- Modify: `cli/internal/platform/windows/runner_windows.go`
- Modify: `cli/cmd/myference/platform_windows.go`
- Modify: `packaging/windows/install.ps1`

**Interfaces:**
- Produces: `windows headless install|start|status|restore`, Myference-owned Scheduled Task definitions using absolute executable/config paths, and `windows restore` ordering focus → provider process → lid → shell → tuning → tasks.

- [ ] Add failing tests for absolute task actions calling `serve --config`, shell presence/value, AC/DC lid state, active-session refusal, Myference-only task deletion, reverse restore, missing optional app warnings, mandatory shell/power failure retention, and repeat restore.
- [ ] Implement preflight/elevation checks before mutation; never change shell or lid state when their originals cannot be read and journaled.
- [ ] Ensure headless runs the same shared multi-backend `serve` provider and never references the legacy gateway.
- [ ] Update the colocated Windows lifecycle script with fixed parameterized Scheduled Task operations; keep all user paths as arguments, not interpolated command text.
- [ ] Run package tests, full tests, vet, Windows build, and macOS cross-build.
- [ ] Commit with `feat: run the multi-backend provider headlessly`.

### Task 7: Ship and verify the complete Windows installation path

**Files:**
- Modify: `scripts/build-release.sh`
- Modify: `web/public/install.ps1`
- Modify: `.github/workflows/verify.yml`
- Create: `scripts/windows-acceptance.ps1`
- Create: `docs/windows-acceptance.md`
- Modify: `README.md`

**Interfaces:**
- The release archive contains `myference.exe`, `myference-agent-proxy.exe`, and `install-windows.ps1`; the public installer checksum-verifies and atomically installs all three under `%LOCALAPPDATA%\Programs\Myference`.

- [ ] Add failing release/installer fixture tests proving every required asset is archived, checksum-verified, installed together, and preserved on failed update.
- [ ] Add a Windows Actions job running `go test ./...`, `go vet ./...`, CLI/proxy builds, PowerShell parser checks, installer fixtures, and non-destructive acceptance.
- [ ] Add acceptance automation that captures power, lid, shell, tasks, and provider status; configures two fixture backends; exercises backend start/stop and verifies offer changes; runs doctor/models/test/status/restore; and proves host state matches afterward.
- [ ] Document manual keep-awake, priority, focus, graceful stop, forced crash recovery, dashboard, headless sign-out/login, provider reconnect, and Task Manager emergency restore checks without machine-specific values or secrets.
- [ ] Run all local Go checks, installer tests, PowerShell parsing, Windows release build, macOS release builds, and `git diff --check`.
- [ ] Commit with `ci: verify provider-owned Windows management`.

### Task 8: Remove the legacy CLI only after acceptance passes

**Files:**
- Delete after gate: `cli/platform/windows/legacy/`
- Delete after gate: `cli/internal/platform/windows/lifecycle.go`
- Modify after gate: `cli/cmd/myference/main.go`
- Modify after gate: `cli/cmd/myference/platform_windows.go`
- Modify after gate: `README.md`
- Modify after gate: `scripts/build-release.sh`

**Gate:** Every checkbox in `docs/windows-acceptance.md` is completed on a real Windows machine and the Windows Actions job is green for the exact commit.

- [ ] Add failing tests rejecting `legacy-start`, `legacy-status`, `legacy-stop`, LAN, firewall, and removed `optimize` commands while accepting all final Windows commands and the existing multi-backend controls.
- [ ] Delete legacy scripts/dashboard/gateway code, compatibility lifecycle aliases, legacy packaging, and documentation references only after the gate is evidenced.
- [ ] Run `go test ./... -count=1`, `go vet ./...`, provider/platform race tests, Windows AMD64 release build, macOS AMD64/ARM64 release builds, installer fixtures, PowerShell parser validation, and `git diff --check`.
- [ ] Commit with `refactor: remove accepted Windows legacy CLI`.
