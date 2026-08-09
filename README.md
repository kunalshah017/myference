# Myference

Myference is a Monad-native AI inference marketplace that turns unused computers into paid model providers. Customers use familiar OpenAI- and Anthropic-compatible APIs; provider machines connect outbound, execute requests, stream results, and settle signed usage receipts through native-MON escrow.

## Repository status

The repository contains the shared Go provider CLI, broker, Monad contracts, web client, and preserved Windows host implementation. The working vertical slice uses browser wallet login, delegated machine receipt signers, indexed native-MON spending sessions, real provider streaming, EIP-712 co-signing, durable batch settlement, and realtime account/request projections.

```text
cli/cmd/myference/             Shared Windows/macOS provider CLI
cli/internal/backend/ollama/   Real loopback Ollama adapter
cli/internal/backend/openai/   OpenAI-compatible cloud adapter
cli/internal/backend/codex/    Model-only native Codex CLI adapter
cli/internal/backend/claude/   Model-only native Claude CLI adapter
cli/internal/backend/command/  Disposable image-based Codex/Claude/Kimi runner
cli/internal/provider/         Authenticated outbound provider daemon
cli/internal/platform/windows/ Native Windows provider lifecycle
cli/internal/platform/darwin/  Native launchd lifecycle
cli/platform/windows/legacy/   Preserved Windows PowerShell and Go CLI
server/internal/settlement/    Receipt signing and settlement coordinator
web/                           Wallet, marketplace, billing, and provider console
docs/superpowers/specs/        Approved system design
```

See the [marketplace design](docs/superpowers/specs/2026-08-02-myference-marketplace-design.md) for the broker, relay, pricing, collateral, receipt, settlement, security, and real end-to-end verification requirements.

## Provider CLI

Install the latest checksum-verified CLI with one command:

```powershell
irm https://myference.xyz/install.ps1 | iex
```

```bash
curl -fsSL https://myference.xyz/install.sh | sh
```

The Windows installer supports AMD64, atomically installs the CLI, Linux container proxy sidecar, and lifecycle script, and updates the user PATH. The macOS installer detects Intel or Apple Silicon and installs into `/usr/local/bin`. Manual release artifacts and `SHA256SUMS` remain available on GitHub; releases are not yet code-signed or notarized.

Run `myference` with no arguments for the recommended full-screen hosting interface. It opens browser sign-in when needed, discovers every installed Ollama model plus Codex and Claude on `PATH`, supports OpenAI and OpenAI-compatible model catalogs, lets you select several providers, and shows live foreground status. Existing commands remain available for automation and recovery:

```text
myference login --server https://api.myference.xyz
myference backend add --kind ollama --name local-qwen --model qwen2.5:0.5b
myference backend add --kind openai --name cloud-model --model provider-model --url https://provider.example --secret "$PROVIDER_KEY"
myference backend add --kind codex --name codex-cli-terra --model gpt-5.6-terra
myference backend list
myference backend remove --name <retired-backend>
myference capacity
myference service install
myference service start
```

`myference host` remains the non-interactive Ollama shortcut. It discovers installed models, records the runtime digest, opens the provider workspace for collateral and price activation, and serves in the foreground. Use `--model <name>` to choose a particular installed model or `--setup-only` before installing the background service.

Machine, backend, and EIP-712 signer secrets are loaded from Windows Credential Manager or macOS Keychain and never stored in JSON. Browser approval submits `setProviderSigner` on Monad before the machine can become routable. `backend start`, `backend stop`, and `backend remove` are detected by the running daemon and update advertised capacity without disconnecting other backends. Removing a credential-backed backend also deletes its vault credential.

Ollama must use loopback. Cloud adapters require HTTPS except for loopback providers. Native Codex requires `codex` on `PATH` and an existing `codex login`; native Claude similarly reuses the installed Claude CLI login. Neither needs Docker or a separate API key. Both run in model-only mode with empty temporary workspaces and reject workspace input; Codex uses its deny-all hook and event filter, while Claude uses safe mode, strict empty MCP configuration, and an empty tool list. Only final text and reported usage reach marketplace clients. Use `backend add --replace` to migrate an existing offer in place without changing its name or price version.

Kimi and explicitly image-backed Codex or Claude backends still require Docker Desktop, a digest-pinned agent image, and `--secret`; for example, `myference backend add --kind codex --name codex-image --model <supported-model> --image ghcr.io/kunalshah017/myference-codex@sha256:... --secret "$OPENAI_API_KEY"`. On Windows provider startup, Myference starts Docker Desktop when needed, waits up to two minutes for its Linux engine, pulls only missing immutable images, and verifies them before advertising capacity. Each image agent runs in an ephemeral, read-only, capability-dropped container on a unique internal Docker network and mounts only the disposable workspace—never the host home or Docker socket. A separately packaged Linux proxy sidecar is the only dual-homed peer: it permits only the configured upstream model and inference endpoints within the job's cumulative output-token budget. The agent sees only a random job token; the long-lived credential is mounted only into the proxy sidecar.

Marketplace prices are displayed as MON with a cached, informational USD reference. Billing and settlement always use the exact immutable integer MON rates published on-chain. Ollama, compatible APIs, and native Codex meter observed input, output, and compute usage. Image-based CLI agents are compute-only unless trustworthy upstream usage is available.

When publishing a later immutable offer version for a price or runtime-digest change, select it on the running machine with `myference backend version --name <backend> --price-version <version>`. The daemon reloads this change without interrupting other backends.

`myference service install|start|stop|status|uninstall` uses a Windows Scheduled Task or a per-user macOS LaunchAgent. A foreground `serve` process stops cleanly with Ctrl+C. `legacy-start`, `legacy-status`, and `legacy-stop` remain available only for the preserved Windows LAN host.

Windows provider management is native and outbound-only: `myference windows doctor|models|test|status|dashboard`, `windows focus start|status|restore`, `windows headless install|start|status|restore`, and the idempotent emergency `windows restore`. One `serve` owns every enabled laptop backend; focus and headless reuse it and never open a LAN management listener. Complete [the physical Windows acceptance checklist](docs/windows-acceptance.md) before removing the preserved legacy implementation.

`myference windows doctor` and `myference windows headless status` inspect Docker readiness without starting it or pulling images. `windows headless start` signs out; the next login launches the same provider task without Explorer, and its startup path performs the bounded Docker start/pull preparation described above.

Windows provider tuning requires AC power by default. For an intentional foreground battery session, use `myference serve --allow-battery` or `myference host --allow-battery`; scheduled service/headless sessions retain the safer AC-only policy.

## Broker server

Apply the SQL migrations in numeric order and configure the actual settlement runtime:

```text
MYFERENCE_DATABASE_URL=postgres://...
MYFERENCE_RPC_URL=https://testnet-rpc.monad.xyz
MYFERENCE_CONTRACT_ADDRESS=0x...
MYFERENCE_SETTLEMENT_PRIVATE_KEY=...  # deployment secret manager, never source control
MYFERENCE_CHAIN_START_BLOCK=...
MYFERENCE_WEB_ORIGIN=https://myference.xyz
MYFERENCE_EXPLORER_URL=https://testnet.monadexplorer.com
```

Then run:

```text
go run ./server/cmd/myference-server
```

The server binds to `127.0.0.1:8080` by default and automatically uses `0.0.0.0:$PORT` on managed hosts such as Render. It exposes `/relay`, `/v1/chat/completions`, native Anthropic `/v1/messages`, `/healthz`, wallet/device authentication, marketplace/account APIs, and ticketed SSE. Set both `MYFERENCE_TLS_CERT` and `MYFERENCE_TLS_KEY` when terminating TLS in-process, or keep it on loopback behind a TLS reverse proxy.

## Render deployment

[`render.yaml`](render.yaml) defines a Singapore-region Go API, React static site, and private managed PostgreSQL database. The API runs every numeric migration before boot. Blueprint secrets are intentionally prompted rather than committed: the Monad RPC URL, deployed contract address and start block, settlement key, web origin, auth domain, explorer URL, and the static site's API URL must all be real values.

Validate the infrastructure definition with `render blueprints validate render.yaml`. The contract must be deployed first because the API verifies the configured chain, bytecode, and settlement signer during startup.

Requests reserve only finalized session allowance. After real output, the broker builds the receipt exclusively from measured usage and indexed immutable prices, asks the authorized headless machine signer to sign the exact Monad EIP-712 domain, co-signs it, and durably queues the batch. The runtime persists the raw transaction before broadcast; only the finality-aware indexer moves a request to `settled` and changes visible balances.

The chain package uses the generated `MyferenceMarket` binding for Monad-compatible EVM RPC. Its indexer persists block hashes and logs, waits for configured confirmations, rewinds projections on canonical-hash disagreement, and resumes idempotently after restart. Only co-signed receipts enter the settlement queue; the signed transaction hash is persisted before broadcast.

Both API dialects are streaming-only and require `X-Myference-Max-Spend` in wei MON. OpenAI clients send a bearer key to `/v1/chat/completions`; Anthropic clients send the same key through `x-api-key` to `/v1/messages` with `anthropic-version: 2023-06-01`. Coding-agent offers may also receive a bounded disposable workspace (at most 64 files and 512 KiB decoded) through the same extension in either dialect:

```json
{
  "model": "codex-agent",
  "stream": true,
  "messages": [{"role": "user", "content": "Review this file."}],
  "myference_workspace": [
    {"path": "src/main.go", "content_base64": "cGFja2FnZSBtYWluCg=="}
  ]
}
```

Workspace paths must be normalized relative paths. The relay rejects traversal, absolute paths, invalid base64, excess files, and excess decoded bytes; cloud and Ollama backends reject workspace jobs. Command backends materialize files with private permissions, run them in the isolated container, remove the container and both job networks on cancellation, and delete the workspace on every exit.

## Preserved Windows CLI

The existing CLI is documented in [`cli/platform/windows/legacy/README.md`](cli/platform/windows/legacy/README.md). It runs Ollama on loopback and exposes the original private-LAN streaming gateway. That gateway is preserved for migration and recovery; marketplace traffic uses the authenticated outbound provider daemon above.

## Hackathon proof

Build downloadable artifacts with `make release`. Run local contracts, Go tests, vet, and builds with `make verify`.

The submission is complete only when a physical Windows provider serves a real Ollama request through the hosted broker and settles its signed receipt on Monad testnet. `scripts/e2e-testnet.sh` rejects localhost/Anvil, missing funded setup transactions, a non-Windows CLI attestation, unavailable routed inventory, non-streamed output, failed receipts, and an unindexed settlement. On success it writes sanitized, explorer-linked evidence to `docs/demo.md`.

Fake balances, mocked demo providers, hard-coded model responses, and simulated settlement are not accepted product paths. Unit/integration fixtures exercise failure conditions; public evidence is generated only from real RPC, broker, PostgreSQL, and provider state.
