# Myference

Myference is a Monad-native AI inference marketplace that turns unused computers into paid model providers. Customers use familiar OpenAI- and Anthropic-compatible APIs; provider machines connect outbound, execute requests, stream results, and settle signed usage receipts through native-MON escrow.

## Repository status

The repository contains the shared Go provider CLI, broker, Monad contracts, web client, and preserved Windows host implementation. The working vertical slice uses browser wallet login, delegated machine receipt signers, indexed native-MON spending sessions, real provider streaming, EIP-712 co-signing, durable batch settlement, and realtime account/request projections.

```text
cli/cmd/myference/             Shared Windows/macOS provider CLI
cli/internal/backend/ollama/   Real loopback Ollama adapter
cli/internal/backend/openai/   OpenAI-compatible cloud adapter
cli/internal/backend/command/  Disposable Codex/Claude/Kimi runner
cli/internal/provider/         Authenticated outbound provider daemon
cli/internal/platform/windows/ Windows lifecycle bridge
cli/internal/platform/darwin/  Native launchd lifecycle
cli/platform/windows/legacy/   Preserved Windows PowerShell and Go CLI
server/internal/settlement/    Receipt signing and settlement coordinator
web/                           Wallet, marketplace, billing, and provider console
docs/superpowers/specs/        Approved system design
```

See the [marketplace design](docs/superpowers/specs/2026-08-02-myference-marketplace-design.md) for the broker, relay, pricing, collateral, receipt, settlement, security, and real end-to-end verification requirements.

## Provider CLI

Download the binary for the machine, connect it to a wallet-bound account in the browser, configure one or more independently controlled backends, and start the outbound provider:

```text
myference login --server https://api.example.com
myference backend add --kind ollama --name local-qwen --model qwen2.5:0.5b
myference backend add --kind openai --name cloud-model --model provider-model --url https://provider.example --secret "$PROVIDER_KEY"
myference backend add --kind codex --name codex-agent --model codex --image ghcr.io/example/codex-agent@sha256:... --secret "$OPENAI_API_KEY"
myference backend list
myference capacity
myference service install
myference service start
```

Machine, backend, and EIP-712 signer secrets are loaded from Windows Credential Manager or macOS Keychain and never stored in JSON. Browser approval submits `setProviderSigner` on Monad before the machine can become routable. `backend start` and `backend stop` are detected by the running daemon and update advertised capacity without disconnecting other backends.

Ollama must use loopback. Cloud adapters require HTTPS except in loopback integration tests. Codex, Claude, and Kimi require Docker Desktop and a digest-pinned agent image. Each agent runs in a read-only, capability-dropped container on a unique internal Docker network and mounts only the disposable workspace. A separately packaged proxy sidecar is the only dual-homed peer: it permits only the configured upstream model and inference endpoints within the job's cumulative output-token budget. The agent sees only a random job token; the long-lived credential is mounted only into the proxy sidecar. Their receipts can bill measured compute time without inventing unavailable token counts.

`myference service install|start|stop|status|uninstall` uses a Windows Scheduled Task or a per-user macOS LaunchAgent. A foreground `serve` process stops cleanly with Ctrl+C. `legacy-start`, `legacy-status`, and `legacy-stop` remain available only for the preserved Windows LAN host.

## Broker server

Apply the SQL migrations in numeric order and configure the actual settlement runtime:

```text
MYFERENCE_DATABASE_URL=postgres://...
MYFERENCE_RPC_URL=https://testnet-rpc.monad.xyz
MYFERENCE_CONTRACT_ADDRESS=0x...
MYFERENCE_SETTLEMENT_PRIVATE_KEY=...  # deployment secret manager, never source control
MYFERENCE_CHAIN_START_BLOCK=...
MYFERENCE_WEB_ORIGIN=https://app.example.com
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
