# Myference

Myference is a Monad-native AI inference marketplace that turns unused computers into paid model providers. Customers use familiar OpenAI- and Anthropic-compatible APIs; provider machines connect outbound, execute requests, stream results, and settle signed usage receipts through native-MON escrow.

## Repository status

The repository contains the shared Go provider CLI, broker services, Monad contracts, web client, and the preserved Windows Ollama host implementation. The provider CLI now discovers and streams real Ollama inference over an authenticated outbound TLS WebSocket; the preserved Windows lifecycle remains available for reversible power, firewall, startup, and desktop-state management.

```text
cli/cmd/myference/             Shared Windows/macOS provider CLI
cli/internal/backend/ollama/   Real loopback Ollama adapter
cli/internal/provider/         Authenticated outbound provider daemon
cli/internal/platform/windows/ Windows lifecycle bridge
cli/platform/windows/legacy/   Preserved Windows PowerShell and Go CLI
docs/superpowers/specs/        Approved system design
```

See the [marketplace design](docs/superpowers/specs/2026-08-02-myference-marketplace-design.md) for the broker, relay, pricing, collateral, receipt, settlement, security, and real end-to-end verification requirements.

## Provider CLI

Configure an Ollama model, inspect the capacity that will be published, and start the outbound provider:

```text
myference backend add --name local-qwen --model qwen2.5:0.5b
myference backend list
myference capacity
myference serve
```

The machine token is loaded from Windows Credential Manager or macOS Keychain and is never stored in the JSON configuration. `serve` refuses missing models, non-loopback Ollama endpoints, unsupported backend types, and machines with no enabled backends.

On Windows, `legacy-start`, `legacy-status`, and `legacy-stop` expose the preserved reversible host lifecycle while migration continues. The general `stop` command also restores that lifecycle state; a foreground `serve` process stops cleanly with Ctrl+C.

## Broker server

Apply the SQL migrations in numeric order, set `MYFERENCE_DATABASE_URL`, and run:

```text
go run ./server/cmd/myference-server
```

The server binds to `127.0.0.1:8080` by default and exposes `/relay` plus the OpenAI-compatible `/v1/chat/completions` streaming endpoint. Set both `MYFERENCE_TLS_CERT` and `MYFERENCE_TLS_KEY` when terminating TLS in the process; deployments may instead keep the server on loopback behind a TLS reverse proxy. Reservations, capacity, request transitions, receipt proposals, and realtime outbox events are persisted in PostgreSQL.

## Preserved Windows CLI

The existing CLI is documented in [`cli/platform/windows/legacy/README.md`](cli/platform/windows/legacy/README.md). It runs Ollama on loopback and exposes the original private-LAN streaming gateway. That gateway is preserved for migration and recovery; marketplace traffic uses the authenticated outbound provider daemon above.

## Hackathon proof

The submission is complete only when a real Windows provider serves a real Ollama request through the broker and settles its signed receipt on Monad testnet. Fake balances, mocked providers, hard-coded model responses, and simulated settlement are not accepted product paths.
