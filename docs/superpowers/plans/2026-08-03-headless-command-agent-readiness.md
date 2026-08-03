# Headless Command-Agent Readiness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reliably prepare Docker-backed command agents in Windows foreground and headless sessions and ship a runnable Codex provider through the one-command installer.

**Architecture:** A testable Windows Docker adapter owns engine discovery/start/wait/image preparation and read-only status. The shared Windows backend-preparation hook invokes it for every unique enabled command-agent image, while packaging supplies the Linux proxy binary and a pinned Codex container image.

**Tech Stack:** Go 1.24, Windows process APIs, Docker Desktop CLI, PowerShell installer, Docker Buildx/GHCR, GitHub Actions.

## Global Constraints

- Public API consumers receive model output only; no agent tool surface, Docker socket, host home, MCP configuration, or host filesystem access is exposed.
- Command-agent images stored in config remain immutable `@sha256:` references.
- Doctor and status commands are read-only; only provider startup may start Docker or pull a missing pinned image.
- The public Windows install command remains `irm https://myference.xyz/install.ps1 | iex`.
- Existing multi-provider configuration, live enable/disable behavior, Ollama preparation, and recovery journals remain authoritative and unchanged.

---

### Task 1: Add test-driven Windows Docker readiness

**Files:**
- Create: `cli/internal/platform/windows/docker.go`
- Create: `cli/internal/platform/windows/docker_test.go`
- Create: `cli/internal/platform/windows/process_windows.go`
- Create: `cli/internal/platform/windows/process_other.go`

**Interfaces:**
- Produces: `DockerRuntime`, `DockerStatus`, `DiscoverDockerRuntime()`, `Status(context.Context, []string) DockerStatus`, and `Prepare(context.Context, []string, time.Duration) error`.

- [ ] Write table-driven tests whose literal command traces prove ready engines do not start Docker, stopped engines start once and poll, timeouts return the last error, Windows-container mode is rejected, present images are not pulled, and absent pinned images are pulled and re-inspected.
- [ ] Run `go test ./cli/internal/platform/windows -run Docker -count=1` and confirm the tests fail because the runtime does not exist.
- [ ] Implement the smallest injected runner/starter/waiter adapter and hidden Windows process launcher that passes those branches.
- [ ] Run the focused tests and `go test ./cli/internal/platform/windows -count=1`.
- [ ] Commit with `feat: prepare Docker command agents on Windows`.

### Task 2: Integrate readiness and observable status

**Files:**
- Modify: `cli/cmd/myference/platform_windows.go`
- Modify: `cli/cmd/myference/platform_windows_test.go`
- Modify: `cli/internal/platform/windows/diagnostics.go`
- Modify: `cli/internal/platform/windows/diagnostics_test.go`

**Interfaces:**
- Consumes: `DockerRuntime.Status` and `DockerRuntime.Prepare` from Task 1.
- Produces: enabled-image extraction, startup preparation, Docker engine/image doctor findings, and Docker details in `windows headless status`.

- [ ] Write failing tests for unique enabled command-agent image extraction, preparation before discovery, read-only doctor findings, and headless status rendering.
- [ ] Run the focused CLI/platform tests and confirm failures describe the missing readiness behavior.
- [ ] Invoke `Prepare` from `prepareWindowsBackends`, preserve Ollama preload behavior, and extend structured diagnostics without loading or printing credentials.
- [ ] Run `go test ./cli/cmd/myference ./cli/internal/platform/windows -count=1`.
- [ ] Commit with `feat: report headless Docker readiness`.

### Task 3: Harden Codex to the model-output boundary

**Files:**
- Modify: `cli/cmd/myference/main_test.go`
- Modify: `cli/cmd/myference/main.go`
- Modify: `cli/internal/backend/command/command_test.go`

**Interfaces:**
- Produces: Codex arguments `exec --ephemeral --sandbox read-only --ask-for-approval never --skip-git-repo-check --model MODEL -` and tests that the container has no Docker socket, host home, or long-lived credential.

- [ ] Write a failing test for the exact documented Codex non-interactive safety arguments and strengthen the real Docker argument assertions.
- [ ] Run the focused tests and confirm they fail on missing explicit flags.
- [ ] Add only the documented flags; keep the existing prompt-in/stdout-out Myference boundary.
- [ ] Run `go test ./cli/cmd/myference ./cli/internal/backend/command -count=1`.
- [ ] Commit with `security: constrain Codex command-agent execution`.

### Task 4: Ship the Linux proxy and Codex image

**Files:**
- Create: `packaging/agents/codex/Dockerfile`
- Create: `packaging/agents/codex/README.md`
- Create: `.github/workflows/agent-images.yml`
- Modify: `scripts/build-release.sh`
- Modify: `web/public/install.ps1`
- Modify: `scripts/test-installer-windows.ps1`
- Modify: `scripts/test-installers.sh`
- Modify: `cli/internal/backend/command/command.go`
- Modify: `cli/internal/backend/command/command_test.go`

**Interfaces:**
- Produces: Linux `myference-agent-proxy` in Windows AMD64 packages and `ghcr.io/kunalshah017/myference-codex` release-tag images with immutable published digests.

- [ ] Change installer fixtures first so they fail unless the Linux proxy is required, installed, and restored after an injected failure; add a runner test that selects the Linux sidecar on Windows.
- [ ] Run PowerShell installer and Go tests and confirm expected failures.
- [ ] Build Linux/amd64 proxy for Windows packages, transactionally install it, select it from the Windows CLI, and add a non-root version-pinned Codex image plus tag-triggered Buildx workflow.
- [ ] Run installer tests, `./scripts/build-release.sh` from a clean commit, and Docker-build the image locally.
- [ ] Commit with `build: ship the Docker command-agent runtime`.

### Task 5: Document, verify, publish, and configure the laptop

**Files:**
- Modify: `README.md`
- Modify: `web/src/app/DocsPage.tsx`
- Modify: `web/src/app/App.test.tsx`
- Modify: `docs/windows-acceptance.md`

**Interfaces:**
- Produces: one-command install/update guidance, automatic startup/pull behavior, model-only security language, and real acceptance evidence.

- [ ] Add failing docs/UI tests for the Linux-container sidecar, automatic headless readiness, immutable Codex image, and model-only public boundary.
- [ ] Update documentation and troubleshooting, then run web tests and installer assertions.
- [ ] Run `gofmt`, `go test ./...`, race tests, `go vet ./...`, `go build ./...`, web test/lint/build, PowerShell parsing, installer tests, and local Docker image build.
- [ ] Rebase/merge latest `origin/main`, rerun release verification, merge to main, push, wait for green GitHub Actions, publish the next prerelease and GHCR image, and record the immutable image digest.
- [ ] Rerun `irm https://myference.xyz/install.ps1 | iex`, verify installed hashes/version, start/wait for Docker, add the Codex backend using the existing eligible API credential without printing it, and run backend list, doctor, image inspect, and headless status. Do not sign out or start headless mode; the user will do that.
- [ ] Commit any digest/documentation follow-up, rerun affected checks, push, and wait for green CI before reporting completion.
