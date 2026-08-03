# Native Codex CLI Provider Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Serve model-only text through the Codex CLI already installed and authenticated on a provider machine, without Docker or an OpenAI API key.

**Architecture:** Add a native Codex backend beside the existing OpenAI, Ollama, and isolated-command backends. It runs `codex exec --json` with a private provider home, empty job directory, deny-all pre-tool hook, disabled hosted tools, event filtering, buffered final text, and Codex-reported token usage. CLI replacement semantics migrate the existing offer in place while preserving the legacy digest-pinned Codex image path when explicitly configured.

**Tech Stack:** Go standard library, Codex CLI JSONL protocol, Codex `PreToolUse` hooks, Windows Credential Manager only for unrelated backends, existing provider WebSocket protocol.

## Global Constraints

- Native Codex uses the existing local Codex login and requires no OpenAI API key or Docker.
- Native Codex rejects workspace input and never exposes command, file, MCP, web, app, plugin, skill, or agent tools to marketplace clients.
- OpenAI, Ollama, Claude, Kimi, and explicit digest-pinned Codex image behavior remains unchanged.
- The existing public offer ID, model, price version, and pricing remain stable during migration.
- No long-lived authentication value may appear in config, logs, command arguments, status, test fixtures, or client output.

---

### Task 1: Native Codex runner and security boundary

**Files:**
- Create: `cli/internal/backend/codex/codex.go`
- Create: `cli/internal/backend/codex/codex_test.go`

**Interfaces:**
- Consumes: `backend.Request`, configured model, resolved `codex` executable, source `auth.json`, private provider state directory, and Myference hook executable.
- Produces: `codex.New(model string, timeout time.Duration) (backend.Backend, error)` and a `backend.Backend` implementation returning final text plus `backend.Usage`.

- [ ] **Step 1: Write failing parser tests**

Use literal Codex JSONL fixtures containing `thread.started`, `turn.started`, completed `agent_message`, and `turn.completed` usage. Require buffered text `ok`, input tokens `11`, output tokens `4`, and rejection of fixtures containing `command_execution`, `file_change`, `mcp_tool_call`, `web_search`, `turn.failed`, missing completion, missing usage, empty output, or usage above the request reservation.

- [ ] **Step 2: Verify parser tests fail**

Run: `go test ./cli/internal/backend/codex -run 'TestParse' -v`

Expected: FAIL because the package/parser does not exist.

- [ ] **Step 3: Implement the minimal JSONL parser**

Decode one JSON object per line with `json.Decoder`, collect only completed `agent_message.text`, capture `turn.completed.usage.input_tokens` and `output_tokens`, and return a dedicated tool-attempt error for every non-text tool item. Do not emit text until completion and reservation checks pass.

- [ ] **Step 4: Write failing isolation tests**

Inject a fake process executable that records argv, environment, cwd, stdin, and writes a literal successful JSONL stream. Require `exec --ephemeral --json --ignore-rules --dangerously-bypass-hook-trust --sandbox read-only --skip-git-repo-check --model <model> -`, an empty job cwd, `CODEX_HOME`, `HOME`, and `USERPROFILE` set to the private provider home, disabled web search/agents in generated config, no inherited API keys, prompt on stdin, rejection of nonempty workspace, and cleanup of the job directory.

- [ ] **Step 5: Verify isolation tests fail**

Run: `go test ./cli/internal/backend/codex -run 'TestRunner|TestProviderHome' -v`

Expected: FAIL because the runner and provider-home initializer are absent.

- [ ] **Step 6: Implement native execution and provider-home initialization**

Resolve `codex`, the current user's `.codex/auth.json`, `%LOCALAPPDATA%/Myference/codex-provider`, and the current Myference executable. Seed the private auth only when missing; write restrictive `config.toml` with `approval_policy="never"`, `sandbox_mode="read-only"`, `web_search="disabled"`, `agents.enabled=false`, hooks enabled, and a wildcard `PreToolUse` hook invoking the hidden Myference hook command. Run each request in a new private empty directory, pass a minimal environment, buffer JSONL, check the per-job marker, validate usage, emit final text once, report elapsed milliseconds, and clean up.

- [ ] **Step 7: Verify the native runner passes**

Run: `gofmt -w cli/internal/backend/codex/*.go && go test ./cli/internal/backend/codex -v`

Expected: PASS.

### Task 2: CLI hook and native backend registration

**Files:**
- Modify: `cli/cmd/myference/main.go`
- Modify: `cli/cmd/myference/main_test.go`

**Interfaces:**
- Consumes: `backend add --kind codex`, optional `--image`, optional `--secret`, and `--replace`.
- Produces: native Codex backend selection, explicit legacy image selection, atomic replacement, and hidden `internal codex-deny-tool` hook behavior.

- [ ] **Step 1: Write failing CLI tests**

Require native Codex add to accept no image/secret, explicit image mode to keep requiring both, duplicate names to fail without `--replace`, replacement to preserve the requested name/model/price version while deleting the obsolete backend credential only after config save, and `configuredBackend` to select native Codex when `Image == ""`.

- [ ] **Step 2: Write the failing hook behavior test**

Set a marker path under a test temporary directory, pass representative `PreToolUse` JSON on stdin to the hidden command, and require that it creates the marker and prints the literal deny decision without echoing tool input or secrets.

- [ ] **Step 3: Verify CLI tests fail**

Run: `go test ./cli/cmd/myference -run 'TestBackend.*Codex|TestCodexDenyTool' -v`

Expected: FAIL because native selection, replacement, and the hidden hook command are absent.

- [ ] **Step 4: Implement CLI integration**

Route image-less `kind=codex` to `codex.New`; route explicit image mode to the existing command runner after validating its secret. Add `--replace`, delete stale credentials only after a successful save, remove the obsolete `--ask-for-approval` argument from image-mode Codex, and implement `internal codex-deny-tool` as a narrow marker-and-denial command.

- [ ] **Step 5: Verify CLI integration passes**

Run: `gofmt -w cli/cmd/myference/*.go && go test ./cli/cmd/myference ./cli/internal/backend/command -v`

Expected: PASS.

### Task 3: Provider documentation and regression verification

**Files:**
- Modify: `README.md`
- Modify: `docs/windows-acceptance.md`

**Interfaces:**
- Consumes: installed/authenticated `codex` CLI.
- Produces: documented native add/replace commands, prerequisites, text-only boundary, and diagnostics.

- [ ] **Step 1: Update documentation**

Replace the default Codex example with:

```powershell
myference backend add --kind codex --name codex-cli --model gpt-5.3-codex
```

Document `--image` plus `--secret` as the explicit legacy isolated-image form, state that native mode reuses local Codex login, and state that Myference blocks all Codex tools and workspace access.

- [ ] **Step 2: Run repository verification**

Run: `go test ./...`, `go test -race ./cli/internal/backend/codex ./cli/internal/provider ./cli/internal/platform/windows`, `go vet ./...`, `go build ./...`, and `git diff --check`.

Expected: PASS.

### Task 4: Live Windows migration and release

**Files:**
- Modify outside repository through CLI: `%APPDATA%/myference/config.json`
- Install outside repository: `%LOCALAPPDATA%/Programs/Myference/myference.exe`

**Interfaces:**
- Consumes: existing `gpt-5.3-codex` offer and laptop Codex login.
- Produces: the same public offer backed by native `codex exec`, one healthy provider process, pushed `main`, and green CI.

- [ ] **Step 1: Build and install the tested CLI**

Build `./cli/cmd/myference`, stop the Scheduled Task, restore any confirmed stale provider recovery journal, replace the installed executable after hash verification, and restart exactly one task process.

- [ ] **Step 2: Migrate the existing offer in place**

Run `myference backend add --replace --kind codex --name gpt-5.3-codex --model gpt-5.3-codex --price-version 1` and restart the provider once.

- [ ] **Step 3: Execute live text-only and adversarial requests**

Run a harmless `Reply with exactly: ok` request and require nonzero input/output usage. Run a prompt demanding a shell command and require provider failure, a recorded tool-attempt marker, and no command output delivered to the client.

- [ ] **Step 4: Verify provider health**

Require relay connected, all enabled offers healthy after a successful Codex request, Scheduled Task `Running`, exactly one `myference.exe`, and no leftover Codex job directories or authentication files outside the private provider home.

- [ ] **Step 5: Commit, rebase, push, and monitor CI**

Commit implementation and docs, fetch/rebase onto `origin/main` without force, rerun full verification, push `main`, and monitor the verify workflow through Go, race, PostgreSQL, contracts, web, release, and Windows-installer jobs.
