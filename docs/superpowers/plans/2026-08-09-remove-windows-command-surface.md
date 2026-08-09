# Remove Windows Command Surface Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete the Windows-only and legacy CLI commands while preserving Windows hosting through the shared TUI, `host`, `serve`, and `service` commands.

**Architecture:** The shared router becomes the only public command boundary. Windows keeps a small platform adapter for service lifecycle, automatic provider-session tuning, backend preparation, and browser opening; command-specific doctor, dashboard, focus, headless, model-test, recovery, and compatibility handlers are removed. Installer, acceptance, README, and web documentation use only shared commands.

**Tech Stack:** Go 1.24 CLI and tests, PowerShell installer scripts, React/TypeScript documentation, GitHub Actions cross-compilation.

---

### Task 1: Reject removed top-level commands

**Files:**
- Modify: `cli/cmd/myference/main_test.go`
- Modify: `cli/cmd/myference/main.go`

- [ ] **Step 1: Replace Windows dispatch tests with a failing public-surface test**

```go
func TestRemovedWindowsAndLegacyCommandsAreUnknown(t *testing.T) {
	for _, command := range []string{"windows", "legacy-start", "legacy-status", "legacy-stop", "stop"} {
		err := run([]string{command}, &bytes.Buffer{})
		if err == nil || err.Error() != fmt.Sprintf("unknown command %q", command) {
			t.Fatalf("run(%q) error=%v", command, err)
		}
	}
}
```

- [ ] **Step 2: Run the focused test and verify RED**

Run: `go test ./cli/cmd/myference -run TestRemovedWindowsAndLegacyCommandsAreUnknown -count=1`

Expected: FAIL because `windows` and the compatibility commands still reach `runPlatformCommand`.

- [ ] **Step 3: Remove the router cases and correct the usage text**

Delete the `case "windows"` dispatch and the `case "stop", "legacy-start", "legacy-stop", "legacy-status"` dispatch from `run`. Change the empty-argument usage string to:

```go
return errors.New("usage: myference login | host | backend <add|list|start|stop|remove|version> | offer <publish|list|sync> | collateral <status|deposit|request-exit|finalize-exit> | capacity | status | serve | service <install|start|stop|status|uninstall>")
```

- [ ] **Step 4: Run the focused test and verify GREEN**

Run: `go test ./cli/cmd/myference -run 'TestRemovedWindowsAndLegacyCommandsAreUnknown|TestProviderCommandsRequireExplicitActionInputs|TestServeFlagsPreserveConfigAndBatteryOverride' -count=1`

Expected: PASS.

### Task 2: Delete Windows command handlers without deleting Windows hosting

**Files:**
- Modify: `cli/cmd/myference/platform_windows.go`
- Modify: `cli/cmd/myference/platform_windows_test.go`
- Modify: `cli/internal/platform/windows/optimize.go`
- Modify: `cli/internal/platform/windows/optimize_test.go`
- Modify: `cli/internal/platform/windows/config.go`
- Delete: `cli/internal/platform/windows/lifecycle.go`
- Delete: `cli/internal/platform/windows/headless.go`
- Delete: `cli/internal/platform/windows/headless_test.go`

- [ ] **Step 1: Remove command-handler tests and retain hosting tests**

Delete `TestWindowsStatusAndDashboardReadLocalProviderSnapshot` and `TestWindowsModelsAndTestUseLoopbackOllama`. Keep the tests for `prepareWindowsBackends`, command-agent image preparation, and last-good capacity.

- [ ] **Step 2: Record a clean Windows hosting baseline**

Run: `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -exec=/usr/bin/true ./cli/cmd/myference -run TestRemovedWindowsAndLegacyCommandsAreUnknown -count=1`

Expected: PASS before deletion, proving the retained Windows hosting path starts clean.

- [ ] **Step 3: Reduce the Windows adapter to shared hosting hooks**

Keep only these responsibilities in `platform_windows.go`:

```go
func runPlatformCommand(command string, args []string, output io.Writer) error
func startPlatformProviderSession(context.Context, config.Config, io.Writer) (func() error, error)
func preparePlatformBackends(context.Context, config.Config) error
func prepareWindowsBackends(context.Context, config.Config, platform.Config, *http.Client) error
func commandAgentImages(config.Config) []string
func openBrowser(string) error
```

`runPlatformCommand` accepts only `service` and its five actions. Delete Windows namespace parsing and all doctor, status, dashboard, focus, headless, restore, models, test, telemetry, AC-status, and legacy lifecycle handlers. Delete the unreachable legacy lifecycle bridge and headless installer orchestration. Retain `prepareWindowsDocker`, Ollama preload, provider tuning, recovery-on-stop, and the scheduled-service implementation.

Remove `ParseCommand` and its `Command` type from `cli/internal/platform/windows/config.go`. Change the active-journal error in `optimize.go` so it does not direct users to a removed command:

```go
return fmt.Errorf("%w; stop the existing provider session before starting another", err)
```

- [ ] **Step 4: Run Windows package tests and cross-build**

Run: `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -exec=/usr/bin/true ./cli/cmd/myference ./cli/internal/platform/windows -count=1`

Run: `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./cli/cmd/myference`

Expected: both commands PASS and the retained Windows host builds.

### Task 3: Remove the obsolete headless installer path

**Files:**
- Modify: `scripts/test-installers.sh`
- Modify: `scripts/test-service-installer-windows.ps1`
- Modify: `packaging/windows/install.ps1`

- [ ] **Step 1: Add a failing portable assertion for the removed headless path**

Add these checks to `scripts/test-installers.sh`:

```sh
service_installer="$root/packaging/windows/install.ps1"
! grep -F 'Headless' "$service_installer" >/dev/null
! grep -F 'windows headless' "$service_installer" >/dev/null
```

- [ ] **Step 2: Run the portable installer test and verify RED**

Run: `sh scripts/test-installers.sh`

Expected: FAIL because the service installer still exposes the obsolete headless branch.

- [ ] **Step 3: Change the Windows fixture to require one normal provider task**

Remove the `Headless` switch from `Invoke-ServiceInstaller`, delete the headless invocation/assertion, and assert that the normal task runs:

```powershell
$normal = Invoke-ServiceInstaller
if ($normal.RunLevel -ne 'Limited') { throw "Provider task run level was '$($normal.RunLevel)', expected 'Limited'" }
if ($normal.Argument -notmatch '^serve --config ') { throw "Provider task argument was '$($normal.Argument)'" }
```

- [ ] **Step 4: Simplify the service installer**

Remove the `Headless` parameter and every conditional branch. Use:

```powershell
$taskName = 'Myference Provider'
$argument = 'serve --config "{0}"' -f $configPath
$principal = New-ScheduledTaskPrincipal -UserId $env:USERNAME -LogonType Interactive -RunLevel Limited
```

- [ ] **Step 5: Run portable installer tests and parse PowerShell when available**

Run: `sh scripts/test-installers.sh`

Expected: PASS.

Run: `pwsh -NoProfile -Command '$null = [scriptblock]::Create((Get-Content -Raw packaging/windows/install.ps1)); $null = [scriptblock]::Create((Get-Content -Raw scripts/test-service-installer-windows.ps1))'`

Expected: PASS when PowerShell is installed. The repository's Windows CI runs the full scheduled-task mock fixture.

### Task 4: Update acceptance and user documentation

**Files:**
- Modify: `scripts/windows-acceptance.ps1`
- Modify: `docs/windows-acceptance.md`
- Modify: `README.md`
- Modify: `web/src/app/DocsPage.tsx`
- Test: `web/src/app/App.test.tsx`

- [ ] **Step 1: Add a failing web documentation assertion**

In `serves complete public documentation for using and hosting inference`, add:

```tsx
expect(screen.queryByText(/myference windows/i)).not.toBeInTheDocument()
expect(screen.queryByText(/legacy-(start|status|stop)/i)).not.toBeInTheDocument()
expect(screen.getAllByText(/myference service install/i).length).toBeGreaterThan(0)
```

- [ ] **Step 2: Run the focused web test and verify RED**

Run: `cd web && npm test -- --run src/app/App.test.tsx`

Expected: FAIL because the page still documents Windows headless and doctor commands.

- [ ] **Step 3: Replace command-specific operational guidance**

Use shared commands everywhere:

```text
myference host --model qwen2.5:0.5b --setup-only
myference service install
myference service start
myference status --json
myference backend list
```

Remove legacy, doctor, dashboard, focus, headless, and standalone restore claims. Keep the statements that Windows provider startup prepares Docker/Ollama and applies reversible tuning automatically. Update the Windows acceptance script to call `status --json` and `backend list`. Update the acceptance checklist to verify the shared TUI, foreground `serve`, scheduled service, automatic tuning, and graceful restoration.

- [ ] **Step 4: Run documentation tests and stale-command search**

Run: `cd web && npm test -- --run src/app/App.test.tsx`

Run: `rg -n 'myference windows|legacy-start|legacy-status|legacy-stop|windows (doctor|dashboard|focus|headless|restore)' README.md docs/windows-acceptance.md scripts/windows-acceptance.ps1 web/src packaging/windows scripts/test-service-installer-windows.ps1`

Expected: web test PASS and search returns no matches.

### Task 5: Verify the complete change

**Files:**
- Modify only if verification reveals a defect in the preceding tasks.

- [ ] **Step 1: Format and validate diffs**

Run: `gofmt -w cli/cmd/myference/main.go cli/cmd/myference/main_test.go cli/cmd/myference/platform_windows.go cli/cmd/myference/platform_windows_test.go cli/internal/platform/windows/config.go cli/internal/platform/windows/optimize.go`

Run: `git diff --check`

Expected: PASS.

- [ ] **Step 2: Run all Go verification**

Run: `go test ./... -count=1`

Run: `go vet ./...`

Run: `go build ./...`

Expected: PASS.

- [ ] **Step 3: Cross-compile retained Windows hosting**

Run: `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -exec=/usr/bin/true ./cli/cmd/myference ./cli/internal/platform/windows -count=1`

Run: `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build ./cli/cmd/myference`

Expected: PASS.

- [ ] **Step 4: Run the web verification suite**

Run: `cd web && npm test -- --run`

Run: `cd web && npm run lint`

Run: `cd web && npm run build`

Expected: PASS.

- [ ] **Step 5: Commit implementation**

```bash
git add cli packaging scripts README.md docs/windows-acceptance.md web/src docs/superpowers/plans/2026-08-09-remove-windows-command-surface.md
git commit -m "refactor(cli): remove Windows command namespace"
```
