# Windows Provider Acceptance

Run this checklist on the exact release commit and a physical Windows 11 AMD64 laptop. Record the commit, release tag, tester, date, GPU, Ollama version, and sanitized evidence links. Do not check an item from unit-test output alone.

## Automated prerequisites

- [ ] The `verify` Windows job is green for the exact commit, including Go tests, vet, CLI/proxy builds, PowerShell parsing, installer rollback fixtures, and non-destructive acceptance.
- [ ] `irm https://myference.xyz/install.ps1 | iex` installs `myference.exe`, Linux `myference-agent-proxy`, and `install-windows.ps1` together under `%LOCALAPPDATA%\Programs\Myference`.
- [ ] `scripts/windows-acceptance.ps1 -Config <absolute-config>` passes and its before-state report is retained without secrets.

## Multi-provider lifecycle

- [ ] Two enabled laptop backends/models appear as distinct offers and both serve real broker requests.
- [ ] `backend stop --name <one>` removes only that offer; `backend start` restores it without disconnecting the other provider.
- [ ] `backend remove --name <one>` permanently deletes only that configured offer and removes its vault credential when applicable.
- [ ] A failed model preload or invalid live config preserves the last good advertised capacity.
- [ ] Relay interruption reconnects without losing accumulated local status counters.

## Native Codex model-only provider

- [ ] With `codex` installed and logged in, `backend add --kind codex --name codex-cli-terra --model gpt-5.6-terra` succeeds without Docker, `--image`, `--secret`, or an OpenAI API key.
- [ ] A real native Codex request returns only final model text and reports nonzero input, output, and compute usage.
- [ ] A prompt explicitly demanding shell, file, MCP, web, app, plugin, skill, or agent use fails with a local blocked-tool diagnostic and returns no tool output to the client.
- [ ] The native Codex process uses the private Myference provider home and an empty disposable job directory; the user's normal config, projects, global skills, plugins, MCP servers, apps, and rules are not loaded.
- [ ] Replacing an existing offer with `backend add --replace --kind codex` preserves its name and price version and removes only that offer's obsolete backend credential.

## Host controls and recovery

- [ ] `windows doctor`, `models`, `test`, `status --json`, and `dashboard` report real values; Q closes only the dashboard viewer.
- [ ] Starting on battery is refused; the documented explicit battery exception is tested only when safe.
- [ ] During `serve`, High Performance, keep-awake, Ollama environment/concurrency, and High priority are observable.
- [ ] Ctrl+C/service stop restores the original power plan, Ollama environment/priority, and removes the recovery journal.
- [ ] Forced process termination leaves the journal; elevated `windows restore` restores the exact original state and is safe to repeat.
- [ ] Focus stops only configured optional apps/services, never Explorer, and focus restore leaves provider tuning active.

## Headless lifecycle

- [ ] `windows headless install` records original AC/DC lid actions and shell state before creating its task or mutating the host.
- [ ] With an enabled digest-pinned command agent, `windows headless status` reports Docker/image readiness without starting Docker or pulling; provider startup starts Docker hidden when stopped, reaches Linux mode within two minutes, and pulls only a missing exact digest.
- [ ] The published Codex image runs as a non-root user; a real request returns model text but exposes no tool-call API, host home, Docker socket, MCP configuration, or long-lived credential.
- [ ] `windows headless start` signs out; the next login starts the same multi-backend provider and dashboard without Explorer.
- [ ] Lid-close behavior, provider reconnect, status/dashboard, and on-demand backend start/stop work in headless mode.
- [ ] Graceful headless stop and emergency `windows restore` restore the exact shell presence/value, lid actions, power plan, and remove only Myference-owned headless tasks.
- [ ] From Task Manager → Run new task (Administrator), emergency restore returns the normal desktop even when optional apps are missing.

## Legacy removal gate

- [ ] Every checkbox above is complete for the exact commit.
- [ ] Sanitized acceptance evidence is reviewed and the exact Windows CI commit is green.
- [ ] Only then may `cli/platform/windows/legacy`, compatibility commands, and legacy release assets be deleted.
