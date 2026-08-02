# Myference Marketplace Design

**Date:** 2026-08-02
**Status:** Approved for implementation

## 1. Product definition

Myference is a brokered marketplace that turns unused Windows and macOS computers into paid AI inference providers. Developers call stable OpenAI-compatible and Anthropic-compatible APIs. Myference routes each request to an eligible provider machine, streams the result, meters usage, and settles the payment in native MON through Monad smart contracts.

The provider may independently enable or stop:

- Local inference backends such as Ollama
- Commercial cloud API backends
- Sandboxed CLI agents such as Codex, Claude Code, and Kimi Code

Myference brokers discovery, routing, relay, metering, evidence, and settlement. Provider machines make outbound connections and never require a public IP, router configuration, or inbound marketplace port.

## 2. Goals

The system must:

1. Convert an unused Windows or macOS machine into an authenticated inference provider.
2. Give customers familiar OpenAI and Anthropic API surfaces with streaming.
3. Keep marketplace funds non-custodial through native-MON escrow.
4. Require providers to lock native-MON collateral before receiving work.
5. Lock prices, fees, capabilities, and maximum spend before dispatch.
6. Produce replay-protected signed usage receipts and batch-settle them on Monad.
7. Pay providers and Myference automatically from successful settlements.
8. Reflect machine, request, balance, offer, and settlement changes in realtime.
9. Avoid persisting prompt and response content.
10. Prove the full flow with real infrastructure and real testnet transactions.

## 3. Non-goals for the first hackathon delivery

The first submission will not include:

- Peer-to-peer customer-to-provider networking
- Public inbound ports on provider machines
- Customer-uploaded coding workspaces
- Subjective on-chain judging of model quality
- Automatic collateral slashing for outages or poor responses
- Mainnet deployment before testnet verification and contract review
- Kubernetes, Redis, NATS, or another message broker before measured load requires them

These exclusions do not remove cloud and CLI-agent adapters from the product architecture. They keep the submission focused on one complete, real Ollama-to-Monad transaction loop before additional backends are enabled.

## 4. Delivery principle: no fake product paths

The submission is accepted only when a real customer request completes through:

1. A real Myference API key and funded Monad testnet spending session
2. A real broker and PostgreSQL database
3. A real authenticated outbound provider relay
4. A real Windows provider machine
5. A real Ollama model invocation
6. A real streamed response
7. Real EIP-712 signatures
8. A real Monad testnet settlement transaction
9. Explorer-visible provider and Myference balance changes

There will be no fake marketplace data, simulated balances, mock provider in the demo, hard-coded successful response, or off-chain-only payment presented as a blockchain settlement. Unit test fixtures may isolate failure conditions, but they never replace the live end-to-end acceptance run.

## 5. Chosen architecture

```text
Customer application
        |
        | OpenAI/Anthropic-compatible HTTPS and SSE
        v
API gateway -> reservation and router -> outbound relay
        |                                    |
        |                                    v
        |                              Provider daemon
        |                              |- Ollama/local
        |                              |- Cloud API
        |                              `- Sandboxed CLI agent
        v
PostgreSQL control plane and realtime events
        |
        v
Monad escrow, provider bonds, offers, receipts, settlement, and fees
```

The brokered design is selected over pure P2P because it provides one stable customer endpoint, protects provider addresses, works behind NAT, centralizes rate limits and abuse controls, preserves stream ordering, and makes consistent metering and failover possible. Monad remains authoritative for economic state.

## 6. Repository organization

```text
myference/
|- cli/
|  |- cmd/myference/          # Shared Go CLI entry point
|  |- internal/               # Configuration, relay, adapters, metering, TUI
|  `- platform/
|     |- windows/             # Windows services, power, firewall, startup
|     |  `- legacy/           # Preserved current implementation during migration
|     `- macos/               # launchd, power assertions, Keychain
|- server/                    # API gateway, router, relay, indexer, control plane
|- protocol/                  # Versioned relay and receipt schemas
|- contracts/                 # Solidity contracts and Foundry tests
|- migrations/                # PostgreSQL migrations
|- docs/                      # Design, plans, operations, and demo material
|- scripts/                   # Build, package, deploy, and real E2E scripts
`- README.md
```

The current Windows scripts and Go dashboard move first to `cli/platform/windows/legacy/` without functional edits. The checked-in Windows executable is preserved initially, then replaced by reproducible release artifacts after the shared Go CLI can build it.

## 7. Technology choices

- Go for the shared CLI, provider daemon, terminal UI, gateway, router, relay, indexer, and control plane
- PostgreSQL for durable operational state, idempotency, reservations, health, and realtime outbox events
- Solidity with Foundry and audited OpenZeppelin primitives for Monad contracts
- TLS WebSocket for the outbound provider relay
- HTTP Server-Sent Events for customer model streaming and authenticated control-plane events
- EIP-712 typed data for usage receipts and protocol evidence
- Windows Credential Manager and macOS Keychain for machine credentials and backend secrets

The project will not add infrastructure merely to anticipate scale. A single broker deployment with PostgreSQL is the first production shape. Horizontal relay coordination is introduced only when multiple broker instances are required.

## 8. Accounts, machines, and authentication

`myference login` uses browser-based device authorization. The user signs in, binds a Monad wallet, and approves the machine. The CLI receives a revocable machine-scoped credential and stores it in the operating-system credential store. Wallet private keys never enter the Myference configuration file or broker.

Each registered machine has:

- A unique machine ID
- Account and payout-wallet ownership
- Revocable authentication credentials
- Operating system and daemon version
- Declared backend capabilities
- Health, concurrency, and routing status
- Reputation and quarantine state

API keys are hashed at rest, scoped, individually revocable, and limited by spending session, models, endpoints, rate, concurrency, and maximum request cost.

## 9. Provider CLI

The shared commands are:

```text
myference login
myference doctor
myference machine register
myference backend add <type>
myference backend list
myference backend start <name>
myference backend stop <name>
myference offer set <backend> <model> <prices>
myference bond deposit <amount>
myference bond status
myference serve
myference status
myference earnings
myference logs
myference logout
```

Stopping one backend does not stop the daemon or other backends. A graceful stop drains accepted work; a forced stop explicitly cancels leases. Windows and macOS expose the same commands while their power, startup, and credential implementations remain isolated behind platform interfaces.

## 10. Backend adapter contract

Every backend adapter implements discovery, health, capability declaration, job acceptance, streaming execution, cancellation, usage reporting, and graceful shutdown.

Capabilities are explicit and include:

- Canonical model identifier and provider-specific model identifier
- Text, image, structured-output, and tool support
- Maximum input and output sizes
- Context window
- Streaming support
- Workspace support
- Concurrency
- Usage fields the adapter can report reliably

The router never infers compatibility from a model name.

Ollama binds to loopback. Cloud API credentials remain on the provider machine. CLI agents run in a fresh disposable sandbox with no access to provider home files, Git credentials, browser sessions, Myference secrets, or unrelated processes.

Codex integration uses supported non-interactive execution, ephemeral sessions, JSONL events, explicit sandbox settings, and an eligible business/API authentication method. Commercial adapters require the provider to attest that its account and applicable terms permit serving end users. A technical adapter is not a promise that a personal subscription may be resold.

## 11. Customer APIs

Myference exposes separate native compatibility surfaces:

- OpenAI-compatible models, chat completions, responses, and SSE shapes required by the implemented adapters
- Anthropic-compatible models/messages and Anthropic SSE event shapes

The broker normalizes each request into an internal job envelope without discarding provider-specific features. Unsupported fields fail explicitly before a reservation is created. Model aliases resolve to pinned canonical versions for repeatability.

Customers normally select a model or capability. Automatic routing ranks eligible providers by locked price, health, available concurrency, recent latency, success rate, region, and reputation. Advanced customers may pin a provider. Pinning never bypasses health, balance, capability, collateral, or policy checks.

## 12. Relay protocol

The provider opens one authenticated TLS WebSocket and periodically advertises enabled offers and available capacity. The protocol is versioned.

The broker sends a leased job containing:

- Request ID
- Required backend/model/capabilities
- Immutable offer and fee versions
- Maximum spend
- Deadline and lease expiry
- Input limits and cancellation token

The provider accepts or rejects the lease before receiving the full prompt. Output chunks contain request ID and monotonically increasing sequence number. Acknowledgements, reconnect cursors, deadlines, and idempotency keys prevent duplicate execution and duplicated output.

The broker may retry another provider only before the first output chunk. After streaming begins, the request is bound to that provider and output is never combined with another execution.

## 13. Pricing

Providers freely set prices within protocol-wide safety bounds. Prices are integer values:

- Wei per million input tokens
- Wei per million output tokens
- Wei per compute second for agent work

Every price update creates a new immutable version and applies only to jobs accepted afterward. Integer multiplication uses full-width intermediate arithmetic, deterministic rounding, and an explicit maximum cost. Floating-point values never enter billing or contracts.

Normal inference uses input/output token billing. CLI-agent jobs use input/output tokens when reliably reported plus capped verified compute time. Marketplace prices are all-inclusive.

## 14. Platform fee

The initial Myference fee is 500 basis points, or 5%, of successful settlement value. A listed charge of 1 MON pays 0.95 MON to the provider and 0.05 MON to Myference.

The contract enforces a 1,500 basis-point, or 15%, maximum. Fee changes require a timelock, emit events, and apply only to requests accepting the new fee version. No rejected job or pre-billable failure pays a platform fee.

## 15. Monad contract

The hackathon vertical slice uses one focused `MyferenceMarket.sol` contract to keep economic invariants auditable. It uses native MON and implements:

- Customer deposits and pull-based withdrawals
- Provider registration and collateral deposits
- Configurable minimum bond
- Delayed bond withdrawal
- Versioned provider offer commitments
- Customer spending sessions with locked allowance and expiry
- Delayed session closure for pending settlement
- EIP-712 usage receipt verification
- Provider and Myference co-signatures
- Batch settlement
- Provider/platform fee distribution
- Receipt nonce and request replay protection
- Objective double-signing evidence and collateral slashing
- Timelocked fee changes with the 15% ceiling
- Emergency pause that cannot permanently trap earned or deposited funds

Before mainnet, administrative ownership moves from a deployer wallet to a multisig and the contract receives an independent security review.

## 16. Receipt schema

Every EIP-712 receipt contains:

- Chain ID and verifying contract
- Request ID and spending-session ID
- Customer, provider, and Myference settlement addresses
- Offer ID and price version
- Model and capability hashes
- Input and output token counts
- Verified compute milliseconds
- Customer-approved maximum spend
- Recomputed total charge
- Applied fee basis points and fee version
- Completion status and timestamp
- Salted input and output hashes
- Monotonic receipt nonce

The contract recalculates the charge and rejects invalid signatures, duplicate nonces, exhausted allowances, expired sessions, stale price/fee versions, mismatched totals, and charges above maximum spend.

The Myference broker independently counts observable input and output using the canonical tokenizer when available and cross-checks adapter-reported usage. Provider refusal to sign after streaming results in no provider payout and a reputation penalty; the broker cannot fabricate the provider signature.

## 17. Collateral and slashing

Collateral is required on testnet so the demo exercises the mainnet economic design. Providers become routable only after the indexer confirms a sufficient bond.

Automatic slashing is limited to cryptographically provable violations, initially two conflicting provider-signed receipts for the same request. Outages, timeouts, low-quality answers, process crashes, and network failure affect routing weight, quarantine, and account status rather than collateral because contracts cannot objectively distinguish their causes.

Bond withdrawal starts a delay long enough for unsettled receipts and evidence to surface.

## 18. Request state machine

```text
created -> reserved -> offered -> accepted -> streaming -> completed -> signed -> settled
                    `-> rejected/expired              `-> cancelled/failed
```

Transitions are append-only and idempotent. Terminal states cannot reopen. An accepted request has exactly one provider, offer version, fee version, spending cap, and lease.

## 19. Failure and billing rules

- Provider rejection, lease expiry, or failure before billable work releases the reservation without provider payment.
- Customer cancellation bills accepted input, delivered output, and verified compute time within the maximum spend.
- No retry occurs after the first streamed output chunk.
- A provider that reconnects may resume acknowledgements and sign the receipt observed by the broker.
- Invalid order, duplicate chunks, forged signatures, replayed receipts, price mismatch, and cap overrun fail closed.
- Poor model output is not automatically refundable because inference quality is subjective.
- Repeated malformed output, model mismatch, timeouts, or unsigned completions reduce reputation and can quarantine the machine.
- Backend shutdown drains or explicitly cancels accepted work before withdrawing capacity.
- Missed heartbeats remove the machine from routing eligibility.
- Bounded queues and acknowledgements apply backpressure at customer, broker, relay, and backend boundaries.
- Failed native-MON transfers remain claimable and do not block an entire settlement batch.

## 20. Privacy and abuse controls

Prompt and response content exists only in memory for routing and execution and is not persisted. Durable evidence contains metadata, usage, status, and salted content hashes. Customers are explicitly told that the selected provider receives plaintext prompts; trusted execution environments are not claimed.

Trust boundaries enforce:

- TLS in transit
- Hashed API credentials and rotated machine credentials
- OS credential stores for backend secrets
- Request body, context, output, duration, concurrency, and spend limits
- Rate limiting by account, key, machine, model, and network identity
- Sandboxed CLI-agent execution
- No secret-bearing environment inherited by untrusted jobs
- Minimal structured logs with content redaction
- Immediate key, machine, backend, offer, and session revocation

## 21. Operational and chain state

PostgreSQL is authoritative for accounts, API keys, machine presence, health, current capacity, request lifecycle, off-chain reservations, pending receipts, reputation, and the realtime outbox.

Monad is authoritative for deposits, withdrawals, provider bonds, offers, spending sessions, settlement, fees, and slashing.

The indexer identifies events by chain ID, contract, block hash, transaction hash, and log index. Processing is idempotent. It waits for the configured confirmation/finality policy and can rewind and replay on an RPC reorganization. The CLI and future web client consume one authenticated event stream derived from the PostgreSQL outbox and confirmed Monad events.

## 22. Verification

### Contract verification

- Unit tests cover deposits, withdrawals, bonds, delayed exits, offers, sessions, receipt settlement, fee distribution, pause, and slashing.
- Fuzz tests cover price arithmetic, rounding, batch totals, and spending caps.
- Invariant tests prove MON accounting conservation, single settlement per request, allowance enforcement, and the fee ceiling.
- Static analysis and gas reports run on every contract change.

### Go verification

- Protocol tests cover reconnects, duplicate messages, chunk ordering, lease expiry, cancellation, and backpressure.
- Router tests cover collateral, capability, capacity, health, price locking, reputation, pinning, and retry boundaries.
- Adapter tests execute actual locally installed backends in opt-in integration jobs.
- Security tests exercise oversized requests, malicious prompts, credential isolation, path traversal, command escape attempts, and forged receipts.

### Real end-to-end verification

The release gate runs against a fresh installation, real PostgreSQL, real deployed Monad testnet contract, real funded wallets, real Windows provider daemon, and real Ollama model. It records the customer deposit/session transaction, provider bond/offer transaction, streamed API response, signed receipt, settlement transaction, and final balances. Transaction hashes are included in the submission evidence.

## 23. Hackathon demonstration

The demo performs this visible sequence:

1. Register the Windows provider and display its wallet.
2. Deposit provider collateral on Monad testnet.
3. Discover a real installed Ollama model and publish its versioned offer.
4. Deposit customer MON and open a capped spending session.
5. Start the outbound provider relay.
6. Send a real OpenAI-compatible streaming request with `curl` or an official SDK.
7. Show the provider accept and Ollama generate the response.
8. Show the signed usage receipt and batch settlement transaction.
9. Open the transaction in a Monad explorer.
10. Show decreased customer balance, increased provider earnings, and the 5% Myference fee.

Required submission evidence includes the public repository, public gateway/health link, deployed contract address, explorer transactions, installation instructions, architecture explanation, automated test output, and concise demo video.

## 24. Delivery sequence

1. Preserve and reorganize the existing Windows implementation.
2. Establish the shared Go module, protocol types, and reproducible builds.
3. Implement and deploy the tested Monad market contract.
4. Implement PostgreSQL schema, chain indexer, account/device authentication, and API keys.
5. Implement broker reservations, router, outbound relay, and OpenAI streaming endpoint.
6. Adapt the Windows Ollama lifecycle into the provider daemon.
7. Complete and record the real Windows/Ollama/Monad end-to-end flow.
8. Package, publish, document, deploy, rehearse, and submit.
9. Add macOS lifecycle support.
10. Add native Anthropic API compatibility, cloud API adapters, and isolated CLI-agent adapters.

Every delivery step must leave a runnable, verified state. A feature is not described as working until its real integration path has been executed successfully.

## 25. Current external constraints

- Monad is EVM-compatible, so Solidity, EIP-712, and Ethereum tooling are appropriate. The current Monad testnet uses native MON and chain ID 10143.
- OpenAI documents `codex exec` for non-interactive automation with explicit sandboxes, ephemeral execution, JSONL events, and usage reporting. It also advises that API keys are the default for programmatic automation and warns against exposing Codex execution directly to untrusted public environments.
- Anthropic recommends its native Messages API for production Claude features rather than treating its OpenAI compatibility layer as a complete long-term substitute.

References:

- https://docs.monad.xyz/
- https://developers.monad.xyz/
- https://learn.chatgpt.com/docs/non-interactive-mode.md
- https://openai.com/policies/may-2025-business-terms/
- https://platform.claude.com/docs/en/cli-sdks-libraries/libraries/openai-sdk
