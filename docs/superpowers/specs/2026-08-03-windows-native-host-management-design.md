# Windows Native Host Management Design

## Scope

Migrate every useful Windows machine-management feature from `cli/platform/windows/legacy` into the current Go CLI, except the private-LAN Ollama gateway, inbound firewall rule, LAN endpoint discovery, and LAN checks. Remove the legacy implementation only after the migration matrix and Windows acceptance checks pass.

## Command surface

The normal marketplace path remains unchanged and outbound-only:

```text
myference host
myference serve
myference service install|start|stop|status|uninstall
```

Windows-only machine controls live under one explicit namespace:

```text
myference windows doctor
myference windows status [--json]
myference windows models
myference windows test [--model NAME]
myference windows optimize start [--focus|--exclusive] [--allow-battery]
myference windows optimize restore
myference windows dashboard
myference windows headless install|start|status|restore
myference windows restore
```

There are no `lan`, `firewall`, or local network listener commands.

## Architecture

`cli/internal/platform/windows` becomes the single Windows implementation. Pure configuration, state validation, PowerShell argument construction, and telemetry formatting are kept testable without privileged mutation. A narrow command runner invokes built-in Windows tools (`powercfg`, Scheduled Tasks, CIM, process APIs, and PowerShell) only after preflight validation.

Every mutation is journaled atomically under `%LOCALAPPDATA%\Myference\state` before it is applied. The journal records the active power scheme, lid actions, shell policy, stopped processes/services, Ollama process settings, and installed task names. Restore is idempotent: it applies recorded values, removes only Myference-owned tasks, then deletes the journal only after successful recovery.

## Migrated behavior

- Doctor checks Windows version, Ollama availability, configured model, Docker when agent backends are enabled, credential storage, AC/battery state, service installation, and config readability.
- Status and dashboard show provider connection, uptime, requests, input/output tokens, compute time, CPU, RAM, battery/AC, NVIDIA data when `nvidia-smi` exists, loaded Ollama models, and backend health. The dashboard reads the current CLI/provider status; it does not open a LAN proxy.
- Model preload calls loopback Ollama with the configured context and keep-alive before the provider becomes ready.
- Optimization can enable High Performance, keep the system awake, set Ollama priority, apply Ollama memory/concurrency environment settings, and optionally stop an allowlist of nonessential apps/services.
- Focus stops configured optional apps/services. Exclusive additionally replaces Explorer only inside an explicitly requested headless session.
- Headless mode installs Myference-owned Scheduled Tasks, records and changes the current-user shell, optionally changes lid actions to `Do nothing`, starts the current marketplace provider, and restores the prior desktop on exit or emergency recovery.
- No command disables Defender, Windows Update permanently, networking, drivers, authentication, or security services.

## Failure and recovery rules

- Administrator-only actions fail before mutation with a precise elevation instruction.
- Battery-required policy is checked before optimization unless `--allow-battery` is explicit.
- A second optimization/headless start refuses to overwrite an active recovery journal.
- Partial setup rolls back in reverse order.
- `myference windows restore` is safe to repeat and is available from Task Manager's elevated “Run new task” flow.
- Service and process restoration uses recorded executable/service identities; missing optional items are warnings, not fatal recovery failures.

## Testing and removal gate

Pure tests cover configuration, command planning, journal round trips, idempotent recovery, telemetry parsing, preload requests, and forbidden service protection. GitHub Actions adds a Windows job for `go test`, `go vet`, and a Windows CLI build. A real Windows acceptance checklist validates elevation, power/lid restoration, keep-awake, process priority, focus, headless sign-out/login, emergency restore, dashboard telemetry, and provider reconnect.

Only after those checks pass may the migration delete `cli/platform/windows/legacy`, remove `legacy-start|legacy-status|legacy-stop`, and remove the README legacy note.
