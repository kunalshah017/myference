# Windows Provider Acceptance

Run this checklist on the exact release commit and a physical Windows 11 AMD64 laptop. Record the commit, release tag, tester, date, GPU, Ollama version, and sanitized evidence links. Do not check an item from unit-test output alone.

## Automated prerequisites

- [ ] The `verify` Windows job is green for the exact commit, including Go tests, vet, CLI/proxy builds, PowerShell parsing, installer rollback fixtures, and non-destructive acceptance.
- [ ] `irm https://myference.xyz/install.ps1 | iex` installs `myference.exe`, Linux `myference-agent-proxy`, and `install-windows.ps1` together under `%LOCALAPPDATA%\Programs\Myference`.
- [ ] `scripts/windows-acceptance.ps1 -Config <absolute-config>` passes and its before-state report is retained without secrets.
- [ ] Removed Windows and compatibility command names return the normal unknown-command error.

## Shared CLI hosting flow

- [ ] Running `myference` opens the provider terminal UI and discovers installed Ollama models.
- [ ] OpenAI and OpenAI-compatible providers can be configured through the TUI without exposing their credentials in configuration, status, or logs.
- [ ] Collateral deposit and exit actions open the exact browser wallet approval, and the terminal resumes only after indexed chain confirmation.
- [ ] A new offer can be priced, published, synchronized, and made routable entirely from the CLI.
- [ ] `myference status --json` reports the real release version, commit, machine, capacity, and offer evidence.

## Multi-provider lifecycle

- [ ] Two enabled laptop backends/models appear as distinct offers and both serve real broker requests.
- [ ] `backend stop --name <one>` removes only that offer; `backend start` restores it without disconnecting the other provider.
- [ ] `backend remove --name <one>` permanently deletes only that configured offer and removes its vault credential when applicable.
- [ ] A failed model preload or invalid live config preserves the last good advertised capacity.
- [ ] Relay interruption reconnects without losing accumulated local status counters.

## Automatic Windows host preparation and recovery

- [ ] Starting `myference serve` on battery is refused; `--allow-battery` works when intentionally tested under safe conditions.
- [ ] Provider startup preloads every enabled Ollama model before advertising it.
- [ ] With an enabled digest-pinned command agent, provider startup starts Docker Desktop hidden when stopped, reaches Linux mode within two minutes, pulls only a missing exact digest, and verifies it before advertising capacity.
- [ ] During `serve`, High Performance, keep-awake, Ollama environment/concurrency, and High process priority are observable.
- [ ] Ctrl+C restores the original power plan and Ollama environment/priority and removes the recovery journal.
- [ ] A failed tuning stage rolls back previously applied stages rather than advertising the provider with partial host state.

## Background service lifecycle

- [ ] `myference service install` creates one limited, per-user `Myference Provider` scheduled task running `serve --config <path>`.
- [ ] `service start`, `status`, and `stop` control that task without a Windows-specific command namespace.
- [ ] `service uninstall` removes only the Myference provider task.
- [ ] A background provider reconnects, synchronizes compatible offer versions, and responds to backend start/stop changes.
- [ ] Graceful service stop restores provider tuning and leaves the normal Windows desktop unchanged.

## Release gate

- [ ] Every checkbox above is complete for the exact commit.
- [ ] A hosted model is visible on the live website and serves a real paid inference request through the broker.
- [ ] Sanitized acceptance evidence is reviewed and the exact Windows CI commit is green.
