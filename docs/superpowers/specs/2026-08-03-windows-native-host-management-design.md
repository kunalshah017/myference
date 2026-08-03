# Windows Native Host Management Design

## Scope

Build Windows host management around the current outbound marketplace provider. Reuse only legacy machine controls that improve the new provider lifecycle; do not reproduce the legacy CLI command-for-command. The private-LAN Ollama gateway, inbound firewall rule, LAN endpoint discovery, LAN checks, and other obsolete gateway behavior are excluded. Remove the legacy implementation only after the provider-owned migration and Windows acceptance checks pass.

## Command surface

The normal marketplace path remains unchanged and outbound-only:

```text
myference host
myference serve
myference service install|start|stop|status|uninstall
myference backend add|list|start|stop|version
```

One `serve` process owns every enabled backend on the laptop. Each enabled backend/model is advertised as a separate offer; `backend start|stop --name NAME` changes availability on demand through the existing live config reload and capacity heartbeat. Windows does not create a separate service per backend.

Windows-only machine controls live under one explicit namespace:

```text
myference windows doctor
myference windows status [--json]
myference windows models
myference windows test [--model NAME]
myference windows dashboard
myference windows focus start|status|restore [--allow-battery]
myference windows headless install|start|status|restore
myference windows restore
```

There are no `lan`, `firewall`, or local network listener commands.

## Architecture

`cli/internal/platform/windows` is the single Windows implementation. The cross-platform provider configuration remains the source of truth for backend names, kinds, models, pricing versions, credentials, and enabled state. Windows host policy contains only machine-level tuning and optional focus/headless allowlists; it does not duplicate backend or model ownership. Pure configuration, state validation, command planning, and telemetry formatting remain testable without privileged mutation. A narrow runner invokes built-in Windows tools (`powercfg`, Scheduled Tasks, CIM, process APIs, and narrowly scoped PowerShell) only after preflight validation.

The provider lifecycle owns safe baseline tuning once per `serve` session. Before the daemon advertises capacity, Windows validates every enabled backend and prepares each enabled local Ollama model. The same preparation runs before a live config reload adds a newly enabled backend. If preparation fails, that capacity update is rejected and the last good provider set remains active. OpenAI and command-agent backends retain their existing cross-platform discovery and credential behavior.

Power-plan, keep-awake, Ollama concurrency/environment, and process-priority changes are applied once for the provider session, journaled before mutation, and restored when `serve` stops. An active journal from a crashed session blocks destructive replacement and remains recoverable through `myference windows restore`. Focus and headless are explicit overlays on the provider session, not alternate provider implementations.

Every mutation is journaled atomically under `%LOCALAPPDATA%\Myference\state` before it is applied. The journal records the active power scheme, lid actions, shell policy, stopped processes/services, Ollama process settings, and installed task names. Restore is idempotent: it applies recorded values, removes only Myference-owned tasks, then deletes the journal only after successful recovery.

## Migrated behavior

- Doctor checks Windows version, every enabled backend and configured model, Docker when command-agent backends are enabled, credential storage, AC/battery state, service installation, and config readability.
- Status and dashboard show provider connection, uptime, requests, input/output tokens, compute time, CPU, RAM, battery/AC, NVIDIA data when `nvidia-smi` exists, loaded Ollama models, and backend health. The dashboard reads the current CLI/provider status; it does not open a LAN proxy.
- Enabled local Ollama backends are verified and prepared through loopback HTTP before their offers become ready, both on initial startup and live enable. Model identity comes from the shared backend configuration; there is no separate singular Windows preload model.
- Provider-owned tuning may enable High Performance, keep the system awake, set Ollama priority, and apply Ollama memory/concurrency settings for the lifetime of `serve`.
- Focus is an explicit optional overlay that stops only configured nonessential apps/services. It never changes Explorer.
- Headless mode installs Myference-owned Scheduled Tasks, records and changes the current-user shell, optionally changes lid actions to `Do nothing`, starts the current marketplace provider, and restores the prior desktop on exit or emergency recovery.
- No command disables Defender, Windows Update permanently, networking, drivers, authentication, or security services.

## Failure and recovery rules

- Administrator-only actions fail before mutation with a precise elevation instruction.
- Battery-required policy is checked before provider-owned tuning, focus, or headless operation unless the applicable explicit override allows battery use.
- A second provider-tuning, focus, or headless session refuses to overwrite an active recovery journal.
- Partial setup rolls back in reverse order.
- `myference windows restore` is safe to repeat and is available from Task Manager's elevated “Run new task” flow.
- Service and process restoration uses recorded executable/service identities; missing optional items are warnings, not fatal recovery failures.
- Graceful `serve` shutdown restores provider-owned tuning. A crash leaves the journal intact for startup diagnostics and emergency restore.

## Testing and removal gate

Pure tests cover configuration, multi-backend preparation, live enable/disable, command planning, journal round trips, idempotent recovery, telemetry parsing, loopback Ollama requests, and forbidden service protection. GitHub Actions adds a Windows job for `go test`, `go vet`, and a Windows CLI build. A real Windows acceptance checklist validates multiple simultaneously enabled offers, on-demand backend start/stop, elevation, power/lid restoration, keep-awake, process priority, focus, headless sign-out/login, emergency restore, dashboard telemetry, provider reconnect, and graceful/crash recovery.

Only after those checks pass may the migration delete `cli/platform/windows/legacy`, remove `legacy-start|legacy-status|legacy-stop`, and remove the README legacy note.
