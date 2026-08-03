# Provider Onboarding and Pricing UX Design

## Goal

Make Myference understandable without exposing users to wei, copied model names, or long provider forms. Hosting should begin in the CLI, require only wallet-authorized on-chain actions in the browser, and report model and metering evidence honestly.

## Principles

- Native MON remains the only settlement asset and the only value trusted by the contract.
- USD values are estimates based on a timestamped mainnet MON reference quote; testnet MON itself has no market value.
- Wallet private keys never enter the Myference CLI.
- Ordinary laptops cannot cryptographically prove model weights. The product reports evidence levels instead of using an unqualified “verified” label.
- A backend is charged only for usage dimensions it can measure reliably.
- Raw wei remains available for technical inspection but is not the default interface unit.

## Hosting Journey

The primary entry point is `myference host`. It authenticates the machine when needed, discovers supported local backends, and prints a browser activation URL. Discovery selects installed Ollama models rather than requiring exact model-name entry. Existing explicit `backend add` commands remain available for automation and cloud credentials.

The browser activation page loads the authenticated machine and its discovered backends. The provider selects a backend, enters friendly USD target rates, reviews the converted MON rates, and signs the required bond, signer authorization, and immutable offer publication transactions. Offer identifiers and model names are derived from the selected backend. The dashboard remains the place for on-chain authorization because the provider wallet key must not be stored by the CLI.

The CLI waits for activation, then serves and automatically reconnects after relay restarts. Re-running `myference host` is idempotent: it reuses the machine account and configured backends rather than creating duplicates.

## Pricing and Money Presentation

The server exposes a public cached MON/USD reference quote with price, source, and update time. The quote is informational and never participates in settlement or receipt signing. A failed or stale quote hides USD estimates and leaves MON actions available.

Provider rate fields accept USD targets for:

- one million input tokens;
- one million output tokens;
- one compute minute.

Before publication the web client converts each target into an integer wei rate, shows the exact MON equivalent, and states that the MON amount is locked for that offer version. Compute-per-minute input is converted to the contract’s compute-per-second rate with integer-safe rounding.

Balances, bonds, earnings, fees, allowances, usage totals, marketplace prices, and maximum-spend fields use a shared money formatter. The primary value is compact MON, the secondary value is estimated USD when a quote is available, and raw wei is placed in a technical detail or copy affordance. Inputs that control on-chain value accept MON or USD as appropriate and convert without JavaScript floating-point arithmetic.

## Model Identity Evidence

Backend discovery returns an evidence record:

- `ollama_digest`: model name and immutable digest reported by the local Ollama daemon;
- `upstream_model`: model identifier returned by an OpenAI-compatible provider;
- `runtime_image`: digest-pinned container image used for a command agent;
- `provider_claimed`: no stronger evidence is available.

The machine includes this evidence in capacity heartbeats. The server stores it with the live backend and exposes the evidence level in marketplace responses. If a pinned digest changes, the existing offer is not routed until a new offer version is activated. The UI uses labels such as “Ollama digest pinned” and “Provider claimed,” never a blanket “verified model.”

The authenticated machine and immutable on-chain offer create accountability, not proof of honest execution. Hardware remote attestation and zkML are explicitly outside this iteration because Myference must work on ordinary Windows and macOS machines.

## Metering

Ollama usage comes from `prompt_eval_count`, `eval_count`, and `total_duration`. OpenAI-compatible usage comes from the provider’s final streaming usage object and requests fail closed when generated output lacks usage. Both send input tokens, output tokens, and compute milliseconds in the terminal output chunk.

Command agents currently provide reliable wall-clock compute but may not expose complete token usage. Their offers therefore use compute-only pricing unless the credential proxy observes upstream usage. Token rates are disabled for compute-only backends. The server measures route duration independently and rejects usage that exceeds the reservation bounds; provider-reported usage remains part of the provider- and settlement-signed receipt.

## Consumer UX

The playground loads live marketplace inventory and uses a native model select containing only non-stale models with available capacity. Loading, empty, and failure states explain why no model can be selected.

API keys are entered in a normal text control with visual masking and a show/hide toggle. The control opts out of autocomplete and common password-manager capture attributes. The key remains React state only and is never stored in local storage, session storage, a URL, or analytics.

## Failure Handling

- Discovery failures identify the missing dependency or unreachable local service and leave existing configuration untouched.
- USD quote failure never blocks MON pricing or transactions.
- Conversion rejects zero, negative, malformed, and over-precision values before opening the wallet.
- A changed model digest marks capacity unavailable instead of silently serving a different runtime.
- A backend that omits required usage fails the request; it cannot settle a token-priced receipt.
- Browser activation clearly separates completed, pending, rejected, and failed wallet transactions.

## Testing

- Unit tests cover decimal-safe USD/MON/wei conversion and display formatting.
- HTTP tests cover cached quote success, stale fallback, and upstream failure.
- CLI tests cover discovery, idempotent host setup, evidence generation, and existing explicit configuration compatibility.
- Protocol and server tests cover evidence validation, persistence, and marketplace output.
- Component tests cover model selection, password-manager-safe API key input, USD offer entry, MON preview, and compute-only pricing.
- Existing Go, Solidity, and web regression suites remain green.
- A final local smoke test runs a real Ollama backend and, when installed/authenticated, a real Codex CLI agent through the provider path without printing credentials.
