# Windows Native Host Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port the non-LAN Windows legacy machine-management features into the current Go CLI and safely delete the legacy CLI after real Windows verification.

**Architecture:** Add a journaled Windows host manager behind `myference windows`, reuse the current provider and service commands, and keep every privileged mutation explicit and reversible. Pure planning and parsing code is separated from Windows command execution so most behavior is deterministic under tests, while a Windows Actions job and physical-machine checklist cover platform integration.

**Tech Stack:** Go standard library, Windows PowerShell/CIM/Scheduled Tasks/powercfg, Ollama loopback API, GitHub Actions Windows runner.

---

### Task 1: Lock the command and safety contract

**Files:**
- Create: `cli/internal/platform/windowshost/config.go`
- Create: `cli/internal/platform/windowshost/config_test.go`
- Modify: `cli/cmd/myference/platform_windows.go`
- Modify: `cli/cmd/myference/main.go`

- [ ] Write failing table tests for the `windows` actions, default safe configuration, protected-service rejection, AC-power policy, and rejection of LAN/firewall fields.
- [ ] Run `go test ./cli/internal/platform/windowshost ./cli/cmd/myference` and confirm failures for missing `windowshost.Config` and dispatch.
- [ ] Implement a typed config with `PreloadModel`, `KeepAlive`, `ContextLength`, `MaxLoadedModels`, `NumParallel`, `FlashAttention`, `KVCacheType`, `PerformancePowerPlan`, `ProcessPriority`, `RequireACPower`, `StopProcesses`, and `StopServices`.
- [ ] Add `myference windows <doctor|status|models|test|optimize|dashboard|headless|restore>` dispatch without changing `host`, `serve`, or `service`.
- [ ] Run the focused tests and commit with `feat: define native Windows host controls`.

### Task 2: Add the atomic recovery journal

**Files:**
- Create: `cli/internal/platform/windowshost/journal.go`
- Create: `cli/internal/platform/windowshost/journal_test.go`

- [ ] Write failing tests that save/load a journal, refuse replacement of an active journal, preserve the last good file across a failed write, and make completed recovery idempotent.
- [ ] Run `go test ./cli/internal/platform/windowshost -run Journal` and verify the expected failures.
- [ ] Implement atomic temp-file, sync, rename persistence with mode `0600`; record power scheme, AC/DC lid actions, shell value, stopped process executable paths, stopped services, task names, and applied stages.
- [ ] Implement stage completion and recovery completion methods that remove state only after all mandatory restoration succeeds.
- [ ] Run focused and package tests; commit with `feat: journal reversible Windows changes`.

### Task 3: Implement diagnostics, models, and preload

**Files:**
- Create: `cli/internal/platform/windowshost/diagnostics.go`
- Create: `cli/internal/platform/windowshost/diagnostics_test.go`
- Create: `cli/internal/platform/windowshost/ollama.go`
- Create: `cli/internal/platform/windowshost/ollama_test.go`

- [ ] Write failing tests for doctor findings, `ollama ps` model parsing, automatic installed-model selection, explicit missing-model failure, and the exact preload request (`stream:false`, configured `keep_alive`, `num_ctx`).
- [ ] Run focused tests and verify failures are caused by missing implementations.
- [ ] Implement command discovery and loopback Ollama calls with bounded timeouts; never bind a listener or create a firewall rule.
- [ ] Print actionable doctor results and make preload failure prevent provider readiness.
- [ ] Run focused tests; commit with `feat: add Windows provider diagnostics and preload`.

### Task 4: Implement optimization and focus mode

**Files:**
- Create: `cli/internal/platform/windowshost/optimize.go`
- Create: `cli/internal/platform/windowshost/optimize_test.go`
- Create: `cli/internal/platform/windowshost/runner_windows.go`

- [ ] Write failing command-planning tests for power plan, keep-awake, Ollama environment, priority, focus allowlists, exclusive Explorer handling, and reverse-order rollback.
- [ ] Add tests proving protected services and unconfigured processes are never stopped.
- [ ] Implement a Windows runner using explicit argument arrays—never interpolated shell text—for `powercfg`, process priority, optional process/service stops, and restart from recorded paths.
- [ ] Implement keep-awake with the Windows execution-state API or a narrowly scoped PowerShell process owned by the journal.
- [ ] Apply each mutation only after its original value is recorded; rollback on any subsequent failure.
- [ ] Run `GOOS=windows GOARCH=amd64 go test ./cli/internal/platform/windowshost` on Windows and cross-build on macOS; commit with `feat: add reversible Windows optimization`.

### Task 5: Integrate provider lifecycle and telemetry dashboard

**Files:**
- Create: `cli/internal/platform/windowshost/telemetry.go`
- Create: `cli/internal/platform/windowshost/telemetry_test.go`
- Create: `cli/internal/platform/windowshost/dashboard.go`
- Modify: `cli/internal/provider/daemon.go`
- Modify: `cli/cmd/myference/platform_windows.go`

- [ ] Write failing parsers for CPU/RAM, battery, `nvidia-smi`, `ollama ps`, service state, and provider status JSON.
- [ ] Add a failing dashboard rendering test for uptime, requests, tokens, compute, model, GPU, AC/battery, and backend health.
- [ ] Expose provider counters through the existing status path without opening a new HTTP listener.
- [ ] Implement a terminal dashboard with periodic refresh and `Q` exit; `Q` closes only the viewer unless the dashboard owns the foreground optimized session.
- [ ] Run focused tests and commit with `feat: add Windows host telemetry dashboard`.

### Task 6: Implement headless shell mode and emergency recovery

**Files:**
- Create: `cli/internal/platform/windowshost/headless.go`
- Create: `cli/internal/platform/windowshost/headless_test.go`
- Modify: `packaging/windows/install.ps1`
- Modify: `cli/cmd/myference/platform_windows.go`

- [ ] Write failing tests for task definitions, shell/lid journal entries, active-session refusal, restore ordering, and repeat restore.
- [ ] Generate Myference-owned elevated start/restore Scheduled Tasks using absolute executable and config paths.
- [ ] Record the existing current-user shell and lid actions before replacing them; refuse headless activation if either value cannot be read.
- [ ] Start the current outbound `serve` provider in the headless task, not the removed LAN gateway.
- [ ] Restore provider, lid actions, shell, optimization state, and tasks; expose the same path through `myference windows restore` for emergency use.
- [ ] Run package tests and a Windows cross-build; commit with `feat: add recoverable Windows headless mode`.

### Task 7: Add Windows CI and physical acceptance script

**Files:**
- Modify: `.github/workflows/verify.yml`
- Create: `scripts/windows-acceptance.ps1`
- Create: `docs/windows-acceptance.md`

- [ ] Add a Windows Actions job running `go test ./...`, `go vet ./...`, and `go build ./cli/cmd/myference`.
- [ ] Add a non-destructive acceptance script that captures current power/lid/shell/task state, exercises doctor/status/models/test, and compares state after restore.
- [ ] Document manual focus, keep-awake, priority, preload, sign-out/login headless, dashboard, provider reconnect, and emergency Task Manager recovery checks.
- [ ] Run the Windows Actions job and the physical acceptance checklist; save no machine-specific values or secrets.
- [ ] Commit with `ci: verify native Windows host management`.

### Task 8: Remove the legacy CLI after parity passes

**Files:**
- Delete: `cli/platform/windows/legacy/`
- Delete: `cli/internal/platform/windows/lifecycle.go`
- Modify: `cli/cmd/myference/main.go`
- Modify: `cli/cmd/myference/platform_windows.go`
- Modify: `README.md`
- Modify: `scripts/build-release.sh`

- [ ] Check off every item in `docs/windows-acceptance.md` on a real Windows machine.
- [ ] Write failing tests that reject removed `legacy-start`, `legacy-status`, and `legacy-stop` commands while confirming every new `windows` command remains registered.
- [ ] Delete the legacy scripts, dashboard binary/source, compatibility lifecycle wrapper, and legacy command aliases.
- [ ] Remove legacy packaging and documentation references; document only current outbound provider and Windows host controls.
- [ ] Run `go test ./...`, `go vet ./...`, Windows and macOS release builds, and `git diff --check`.
- [ ] Commit with `refactor: remove migrated Windows legacy CLI` and push `main`.
