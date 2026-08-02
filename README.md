# Myference

Myference is a Monad-native AI inference marketplace that turns unused computers into paid model providers. Customers use familiar OpenAI- and Anthropic-compatible APIs; provider machines connect outbound, execute requests, stream results, and settle signed usage receipts through native-MON escrow.

## Repository status

The repository currently contains the existing Windows Ollama host implementation and the approved marketplace architecture. The Windows code is preserved without behavioral changes while it is migrated into the shared cross-platform provider daemon.

```text
cli/platform/windows/legacy/   Existing Windows PowerShell and Go CLI
docs/superpowers/specs/        Approved system design
```

See the [marketplace design](docs/superpowers/specs/2026-08-02-myference-marketplace-design.md) for the broker, relay, pricing, collateral, receipt, settlement, security, and real end-to-end verification requirements.

## Current Windows CLI

The existing CLI is documented in [`cli/platform/windows/legacy/README.md`](cli/platform/windows/legacy/README.md). It runs Ollama on loopback and exposes the current private-LAN streaming gateway. Marketplace authentication, outbound relay, and Monad settlement are the next implementation phase.

## Hackathon proof

The submission is complete only when a real Windows provider serves a real Ollama request through the broker and settles its signed receipt on Monad testnet. Fake balances, mocked providers, hard-coded model responses, and simulated settlement are not accepted product paths.
