# Myference Hackathon Vertical Slice Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a real Windows-to-Ollama inference request through the Myference broker and settle its signed receipt with native MON on Monad testnet.

**Architecture:** A shared Go module contains the provider CLI, broker, and versioned protocol. PostgreSQL owns operational state; one Solidity contract owns deposits, bonds, offers, spending sessions, receipt settlement, fees, and objective slashing. The only release acceptance path uses a real Windows provider, real Ollama generation, real PostgreSQL, and explorer-visible Monad testnet transactions.

**Tech Stack:** Go, PostgreSQL, Solidity, Foundry, OpenZeppelin Contracts, TLS WebSocket, Server-Sent Events, EIP-712, Monad testnet.

---

## File map

```text
go.mod                                      Shared Go module
Makefile                                    Repeatable checks and builds
docker-compose.yml                          Local PostgreSQL only
protocol/v1/messages.go                     Relay wire messages
protocol/v1/price.go                        Integer billing arithmetic
protocol/v1/receipt.go                      EIP-712 receipt representation
protocol/v1/*_test.go                       Protocol tests
contracts/foundry.toml                      Foundry configuration
contracts/src/MyferenceMarket.sol           Monad market contract
contracts/test/MyferenceMarket.t.sol         Contract unit/fuzz/invariant tests
contracts/script/Deploy.s.sol                Deterministic deployment script
migrations/000001_control_plane.sql          Operational database schema
server/cmd/myference-server/main.go          Broker entry point
server/internal/store/store.go               PostgreSQL operations
server/internal/auth/device.go               Device authorization and API keys
server/internal/relay/hub.go                 Provider WebSocket sessions
server/internal/router/router.go             Eligibility and provider selection
server/internal/api/openai.go                OpenAI-compatible streaming endpoint
server/internal/chain/client.go              Contract calls and receipt settlement
server/internal/chain/indexer.go             Idempotent Monad event ingestion
server/internal/realtime/events.go           Authenticated account event stream
cli/cmd/myference/main.go                    Provider CLI entry point
cli/internal/config/config.go                Non-secret local configuration
cli/internal/credential/store_windows.go     Windows Credential Manager
cli/internal/credential/store_darwin.go      macOS Keychain
cli/internal/backend/backend.go              Backend interface
cli/internal/backend/ollama/ollama.go         Real Ollama adapter
cli/internal/provider/daemon.go              Capacity, leases, streaming, receipts
cli/internal/platform/windows/lifecycle.go   Existing Windows lifecycle migration
scripts/e2e-testnet.sh                       Real testnet acceptance orchestration
docs/demo.md                                 Reproducible demo and explorer evidence
web/src/app/                                 Router, providers, and boundaries
web/src/features/                            Marketplace, billing, activity, provider
web/src/lib/                                 API, Monad, realtime, formatting
web/src/styles/                              Design tokens and global styles
web/src/test/                                Browser and component test setup
web/package.json                             React/TypeScript/Vite commands
```

### Task 1: Establish reproducible Go and repository checks

**Files:**
- Create: `go.mod`
- Create: `Makefile`
- Create: `docker-compose.yml`
- Create: `protocol/v1/price_test.go`

- [x] **Step 1: Write the failing integer-price test**

```go
func TestChargeRoundsUpAndNeverExceedsMaximum(t *testing.T) {
	price := Price{InputPerMillion: 100, OutputPerMillion: 200, ComputePerSecond: 300}
	charge, err := price.Charge(1, 2, 1, 1_000)
	if err != nil || charge != 3 {
		t.Fatalf("charge=%d err=%v", charge, err)
	}
	charge, err = price.Charge(1_000_000, 1_000_000, 1_000, 600)
	if err != nil || charge != 600 {
		t.Fatalf("full-unit charge=%d err=%v", charge, err)
	}
	if _, err := price.Charge(1_000_000, 1_000_000, 1_000, 1); !errors.Is(err, ErrMaximumExceeded) {
		t.Fatalf("expected ErrMaximumExceeded, got %v", err)
	}
}
```

- [x] **Step 2: Run the test and verify the missing type failure**

Run: `go test ./protocol/v1 -run TestChargeRoundsUpAndNeverExceedsMaximum -v`

Expected: FAIL because `Price` and `ErrMaximumExceeded` are undefined.

- [x] **Step 3: Implement checked integer billing in `protocol/v1/price.go`**

Use `math/bits.Mul64` to reject overflow. Compute each component with ceiling division: `(units*rate + denominator - 1) / denominator`, using denominators `1_000_000` for tokens and `1_000` for compute milliseconds. Sum with checked addition and reject a result above `maxWei`.

- [x] **Step 4: Add repository commands**

`Makefile` targets must run `go test ./...`, `go vet ./...`, `go build ./...`, `forge test --root contracts`, and a combined `make verify`. `docker-compose.yml` must run PostgreSQL 17 with a health check and a named volume; credentials come from documented local-only defaults and environment overrides.

- [x] **Step 5: Verify and commit**

Run: `go test ./protocol/v1 -v && go vet ./... && go build ./...`

Expected: PASS and exit 0.

Commit: `git add go.mod Makefile docker-compose.yml protocol/v1/price.go protocol/v1/price_test.go && git commit -m "build: establish shared Go module and checks"`

### Task 2: Define the versioned relay and receipt protocol

**Files:**
- Create: `protocol/v1/messages.go`
- Create: `protocol/v1/messages_test.go`
- Create: `protocol/v1/receipt.go`
- Create: `protocol/v1/receipt_test.go`

- [x] **Step 1: Write failing round-trip and validation tests**

Test `Hello`, `Capacity`, `JobOffer`, `JobAccept`, `OutputChunk`, `Cancel`, `ReceiptProposal`, and `ReceiptSignature`. Assert protocol version `1`, non-empty request IDs, positive lease deadlines, monotonically increasing chunk sequence, 32-byte hashes, and 20-byte EVM addresses.

- [x] **Step 2: Verify RED**

Run: `go test ./protocol/v1 -run 'TestMessage|TestReceipt' -v`

Expected: FAIL because the message and receipt types do not exist.

- [x] **Step 3: Implement strict JSON envelopes**

Define `Envelope{Version uint16, Type string, ID string, Body json.RawMessage}` and a decoder that uses `json.Decoder.DisallowUnknownFields()`, rejects bodies larger than the configured protocol limit, and calls each message's `Validate()` method. Use UUID strings for transport IDs and `[32]byte` for on-chain request IDs and hashes.

- [x] **Step 4: Implement the EIP-712 receipt fields**

The Go `Receipt` must exactly match the Solidity struct order: request ID, session ID, customer, provider, settlement signer, offer ID, price version, model hash, capability hash, input tokens, output tokens, compute milliseconds, maximum charge, total charge, fee basis points, fee version, status, completed timestamp, input hash, output hash, and nonce.

- [x] **Step 5: Verify and commit**

Run: `go test ./protocol/v1 -v`

Expected: PASS.

Commit: `git add protocol/v1 && git commit -m "feat: define relay and receipt protocol"`

### Task 3: Implement deposits, provider bonds, and offers on Monad

**Files:**
- Create: `contracts/foundry.toml`
- Create: `contracts/src/MyferenceMarket.sol`
- Create: `contracts/test/MyferenceMarket.t.sol`

- [x] **Step 1: Write failing contract tests**

Tests must prove that deposits increase only the sender's balance; withdrawals use pull payments; an unbonded provider cannot publish; a bonded provider can publish a price version; a price update increments the immutable version; and a bond cannot withdraw before its delay.

- [x] **Step 2: Verify RED**

Run: `forge test --root contracts -vvv`

Expected: FAIL because `MyferenceMarket.sol` does not exist.

- [x] **Step 3: Implement the minimum market state**

Use OpenZeppelin `Ownable2Step`, `Pausable`, `ReentrancyGuard`, `EIP712`, `ECDSA`, and `Math`. Store customer balances, claimable withdrawals, provider bond state, and `mapping(address => mapping(bytes32 => Offer))`. Publish native-MON deposit, withdrawal, bond, bond-exit, and versioned-offer events.

- [x] **Step 4: Add fuzz coverage**

Fuzz deposit and withdrawal amounts across non-zero `uint128` values. Prove contract balance equals customer balances plus provider bonds plus locked sessions plus claimable funds.

- [x] **Step 5: Verify and commit**

Run: `forge fmt --root contracts --check && forge test --root contracts -vvv`

Expected: PASS.

Commit: `git add contracts && git commit -m "feat: add Monad deposits bonds and offers"`

### Task 4: Implement spending sessions and signed batch settlement

**Files:**
- Modify: `contracts/src/MyferenceMarket.sol`
- Modify: `contracts/test/MyferenceMarket.t.sol`
- Create: `contracts/test/MyferenceMarketInvariant.t.sol`

- [x] **Step 1: Write failing session tests**

Test that opening a session locks its allowance, expiry is enforced, closing starts a delay, pending settlement can complete during the delay, and unused funds return only after finalization.

- [x] **Step 2: Write failing receipt tests**

Create real EIP-712 provider and Myference signatures with Foundry signing helpers. Assert exact billing, 95/5 distribution at 500 basis points, maximum-spend enforcement, replay rejection, bad signature rejection, stale price rejection, and atomic accounting across a batch.

- [x] **Step 3: Verify RED**

Run: `forge test --root contracts --match-test 'testSession|testSettle' -vvv`

Expected: FAIL because session and settlement functions are absent.

- [x] **Step 4: Implement sessions and settlement**

Lock allowance when the customer opens a session. Verify provider and configured settlement-signer EIP-712 signatures. Recompute token and millisecond charges with `Math.mulDiv(..., Math.Rounding.Ceil)`. Mark request IDs and nonces before crediting provider and fee recipient claimable balances.

- [x] **Step 5: Add invariants**

The invariant handler must attempt arbitrary deposit, session, settlement, and withdrawal sequences. Assert conservation of MON, no request settles twice, no session spends above allowance, and `feeBps <= 1500`.

- [x] **Step 6: Verify and commit**

Run: `forge test --root contracts -vvv`

Expected: PASS including invariant tests.

Commit: `git add contracts/src contracts/test && git commit -m "feat: settle signed inference receipts"`

### Task 5: Add objective slashing and timelocked fees

**Files:**
- Modify: `contracts/src/MyferenceMarket.sol`
- Modify: `contracts/test/MyferenceMarket.t.sol`
- Create: `contracts/script/Deploy.s.sol`

- [x] **Step 1: Write failing governance and evidence tests**

Test that two different provider-signed receipts with the same request ID slash once; identical receipts do not; arbitrary evidence fails; fee proposals below or equal to 1,500 basis points execute only after the timelock; and pausing cannot block mature withdrawals.

- [x] **Step 2: Verify RED**

Run: `forge test --root contracts --match-test 'testSlash|testFee|testPause' -vvv`

Expected: FAIL because the evidence and timelock functions are absent.

- [x] **Step 3: Implement evidence, fee scheduling, and deployment**

Hash both typed receipts, recover the same provider from both signatures, require matching request IDs and different hashes, then slash the configured amount once. Store fee proposal value and executable timestamp. Deployment reads owner, fee recipient, settlement signer, minimum bond, exit delay, and fee delay from environment and logs the deployed address.

- [x] **Step 4: Verify and commit**

Run: `forge fmt --root contracts --check && forge test --root contracts -vvv`

Expected: PASS.

Commit: `git add contracts && git commit -m "feat: add provable slashing and fee timelock"`

### Task 6: Create the PostgreSQL control plane and transactional outbox

**Files:**
- Create: `migrations/000001_control_plane.sql`
- Create: `server/internal/store/store.go`
- Create: `server/internal/store/store_integration_test.go`

- [x] **Step 1: Write a failing real-PostgreSQL integration test**

The test must connect through `MYFERENCE_TEST_DATABASE_URL`, apply the migration, create account/machine/backend/offer/session/request records, perform a valid state transition, reject an invalid terminal-state transition, and read the corresponding outbox event.

- [x] **Step 2: Verify RED against real PostgreSQL**

Run: `docker compose up -d --wait postgres && MYFERENCE_TEST_DATABASE_URL=postgres://myference:myference@localhost:5432/myference_test?sslmode=disable go test ./server/internal/store -v`

Expected: FAIL because the migration and store do not exist.

- [x] **Step 3: Implement schema and store**

Use constraints for unique API-key hashes, machine ownership, immutable offer versions, receipt nonces, request transitions, and chain log identity. Write state change and outbox insertion in one SQL transaction. Use PostgreSQL advisory locks for per-request settlement and `FOR UPDATE SKIP LOCKED` for outbox workers.

- [x] **Step 4: Verify and commit**

Run the integration command from Step 2 again.

Expected: PASS using the running PostgreSQL container.

Commit: `git add migrations server/internal/store && git commit -m "feat: add PostgreSQL control plane"`

### Task 7: Implement device login and scoped API keys

**Files:**
- Create: `server/internal/auth/device.go`
- Create: `server/internal/auth/device_integration_test.go`
- Create: `cli/internal/credential/store_windows.go`
- Create: `cli/internal/credential/store_darwin.go`
- Create: `cli/internal/config/config.go`

- [x] **Step 1: Write failing device-flow tests**

Against real PostgreSQL, create a short-lived device code, poll while pending, approve it with a wallet-bound account, exchange it exactly once, hash the issued machine token, revoke it, and reject subsequent authentication. Test API-key model and spend scopes.

- [x] **Step 2: Verify RED**

Run: `MYFERENCE_TEST_DATABASE_URL=postgres://myference:myference@localhost:5432/myference_test?sslmode=disable go test ./server/internal/auth -v`

Expected: FAIL because device authorization is absent.

- [x] **Step 3: Implement authentication**

Generate device codes and bearer tokens with `crypto/rand`; store only SHA-256 token hashes; compare with constant-time operations; enforce expiry and one-time exchange. Store the resulting machine token in Windows Credential Manager or macOS Keychain. Keep wallet keys outside the CLI.

- [x] **Step 4: Verify and commit**

Run: `go test ./server/internal/auth ./cli/internal/credential/... -v`

Expected: PASS on the current operating system; cross-compile both platform packages.

Commit: `git add server/internal/auth cli/internal/credential cli/internal/config && git commit -m "feat: add device login and scoped credentials"`

### Task 8: Implement the authenticated outbound relay

**Files:**
- Create: `server/internal/relay/hub.go`
- Create: `server/internal/relay/hub_test.go`
- Create: `cli/internal/provider/daemon.go`
- Create: `cli/internal/provider/daemon_test.go`

- [x] **Step 1: Write failing relay tests with real loopback sockets**

Start an actual TLS test server and WebSocket client. Test authentication, heartbeat expiry, capacity updates, one lease acceptance, ordered chunks, duplicate chunk rejection, cancellation, bounded queue backpressure, reconnect cursor, and refusal to retry after the first output chunk.

- [x] **Step 2: Verify RED**

Run: `go test ./server/internal/relay ./cli/internal/provider -v`

Expected: FAIL because the relay hub and daemon do not exist.

- [x] **Step 3: Implement broker and daemon state machines**

Use bounded channels, explicit deadlines, one writer goroutine per WebSocket, ping/pong heartbeats, context cancellation, monotonic sequence checks, and idempotent request maps. Never log prompt or output bodies.

- [x] **Step 4: Verify race safety and commit**

Run: `go test -race ./server/internal/relay ./cli/internal/provider -v`

Expected: PASS with no race reports.

Commit: `git add server/internal/relay cli/internal/provider && git commit -m "feat: add outbound provider relay"`

### Task 9: Migrate the real Windows Ollama backend

**Files:**
- Create: `cli/internal/backend/backend.go`
- Create: `cli/internal/backend/ollama/ollama.go`
- Create: `cli/internal/backend/ollama/ollama_integration_test.go`
- Create: `cli/internal/platform/windows/lifecycle.go`
- Create: `cli/cmd/myference/main.go`
- Preserve: `cli/platform/windows/legacy/*`

- [x] **Step 1: Write a failing Ollama integration test**

Require `MYFERENCE_TEST_OLLAMA_MODEL`. Connect to the real local Ollama endpoint, discover that exact installed model, stream a deterministic short prompt, observe at least one content chunk, cancel a second request, and capture real usage fields.

- [x] **Step 2: Verify RED with real Ollama**

Run on Windows: `go test ./cli/internal/backend/ollama -tags=integration -v`

Expected: FAIL because the adapter does not exist.

Recorded RED on the available macOS development host before implementation. The adapter is platform-neutral; physical Windows execution remains part of Step 4's release gate.

- [x] **Step 3: Implement the adapter and Windows lifecycle**

Use Ollama's loopback HTTP streaming API. Port the existing process, firewall, power, keep-awake, startup, and restoration behavior behind `platform/windows` without changing reversible-state guarantees. The CLI commands must add/list/start/stop the Ollama backend, publish capacity, serve, show status, and stop cleanly.

- [ ] **Step 4: Verify real backend and Windows build**

Run on Windows: `go test -race ./cli/... -tags=integration -v && go build -o dist/myference-windows-amd64.exe ./cli/cmd/myference`

Expected: PASS and a runnable Windows executable.

Current evidence: the race-enabled CLI suite passes against real local Ollama `qwen2.5:0.5b`; the Windows amd64 cross-build is a valid PE32+ executable. Running that executable and integration suite on physical Windows remains an explicit release gate.

The arm64 macOS development host has no Wine or Windows VM installed. Installing an emulator would not validate Credential Manager, PowerShell lifecycle, firewall, or power-management behavior, so physical Windows remains the meaningful gate.

- [x] **Step 5: Commit**

Commit: `git add cli && git commit -m "feat: migrate Windows Ollama provider"`

### Task 10: Implement routing and OpenAI-compatible streaming

**Files:**
- Create: `server/internal/router/router.go`
- Create: `server/internal/router/router_test.go`
- Create: `server/internal/api/openai.go`
- Create: `server/internal/api/openai_integration_test.go`
- Create: `server/cmd/myference-server/main.go`

- [x] **Step 1: Write failing router tests**

Use deterministic provider records to prove filtering by confirmed bond, model capability, health, capacity, session balance, price bound, and optional pin. Prove stable ranking by price, latency, success, and reputation and no post-stream retry.

- [x] **Step 2: Write a failing real relay/API integration test**

Run the HTTP server and provider daemon on loopback sockets, submit `/v1/chat/completions` with `stream:true`, and assert valid SSE order, `[DONE]`, request ID propagation, cancellation, and persisted receipt proposal. The provider side must invoke the configured real Ollama integration backend when the integration tag is enabled.

- [x] **Step 3: Verify RED**

Run: `go test ./server/internal/router ./server/internal/api -v`

Expected: FAIL because router and API handlers do not exist.

- [x] **Step 4: Implement reservation, routing, and SSE**

Authenticate the API key, validate request bounds, reserve from a confirmed spending session, lock offer/fee versions, lease one provider, forward ordered chunks, flush SSE, meter observed content, and persist the terminal transition and receipt proposal transactionally.

- [x] **Step 5: Verify and commit**

Run: `go test -race ./server/... ./protocol/... -v`

Expected: PASS.

Commit: `git add server && git commit -m "feat: route OpenAI streaming inference"`

### Task 11: Integrate Monad deployment, indexing, and settlement

**Files:**
- Create: `server/internal/chain/client.go`
- Create: `server/internal/chain/client_integration_test.go`
- Create: `server/internal/chain/indexer.go`
- Create: `server/internal/chain/indexer_integration_test.go`
- Create: `server/internal/realtime/events.go`

- [x] **Step 1: Write failing local-chain integration tests**

Start Anvil, deploy the actual contract, submit real deposit/bond/offer/session transactions, index their logs into real PostgreSQL, settle a provider/broker-signed receipt, restart the indexer, and prove no duplicate records or balance changes.

- [x] **Step 2: Verify RED**

Run: `MYFERENCE_TEST_DATABASE_URL=postgres://myference:myference@localhost:5432/myference_test?sslmode=disable go test ./server/internal/chain -tags=integration -v`

Expected: FAIL because the chain client and indexer do not exist.

- [x] **Step 3: Implement chain client and reorg-safe indexer**

Generate Go contract bindings from the compiled ABI. Persist chain ID, contract address, block number/hash, transaction hash, and log index. Wait for configured finality, detect parent-hash disagreement, rewind unfinalized events, and replay idempotently. Batch only co-signed receipts and persist the settlement transaction before broadcasting realtime events.

- [x] **Step 4: Verify and commit**

Run the integration command from Step 2 again.

Expected: PASS against real Anvil and PostgreSQL.

Commit: `git add server/internal/chain server/internal/realtime && git commit -m "feat: index Monad and settle receipts"`

### Task 12: Initialize the real-data-only web client

**Files:**
- Create: `web/package.json`
- Create: `web/vite.config.ts`
- Create: `web/src/main.tsx`
- Create: `web/src/app/App.tsx`
- Create: `web/src/app/App.test.tsx`
- Create: `web/src/styles/tokens.css`
- Create: `web/src/styles/global.css`
- Create: `web/src/test/setup.ts`

- [x] **Step 1: Initialize React, TypeScript, and Vite**

Run: `npm create vite@latest web -- --template react-ts`

Then install TanStack Query, Viem, Vitest, Testing Library, `jest-dom`, `user-event`, `jsdom`, and Axe. Use a small typed browser History API route map because current React Router releases have overlapping high-severity advisories. Pin the resolved versions in `web/package-lock.json`.

- [x] **Step 2: Write the failing application-shell test**

```tsx
it("shows an honest disconnected marketplace without fake inventory", () => {
  render(<App />);
  expect(screen.getByRole("heading", { name: /unused machines, useful inference/i })).toBeVisible();
  expect(screen.getByText(/live marketplace data is not connected/i)).toBeVisible();
  expect(screen.queryByText(/mock|demo provider|sample balance/i)).not.toBeInTheDocument();
});
```

- [x] **Step 3: Verify RED**

Run: `npm --prefix web test -- --run src/app/App.test.tsx`

Expected: FAIL because the default Vite application does not implement the Myference shell.

- [x] **Step 4: Implement the responsive shell and tokens**

Build semantic header, navigation, main, routing rail, marketplace boundary, and account action. Use the web design spec's Circuit Ink, Relay Violet, Packet Blue, Proof Mint, Fault Coral, and Node Mist tokens. The routing rail must say it is awaiting a live request; no animation runs without a real event.

- [x] **Step 5: Verify and commit**

Run: `npm --prefix web test -- --run && npm --prefix web run lint && npm --prefix web run build`

Expected: PASS with a production bundle and no default Vite assets or fake marketplace records.

Commit: `git add web docs/superpowers && git commit -m "feat: initialize Myference web client"`

### Task 13: Implement wallet, device authorization, and API-key flows

**Files:**
- Create: `migrations/000004_browser_auth.sql`
- Create: `server/internal/auth/browser.go`
- Create: `server/internal/auth/http.go`
- Create: `server/internal/auth/http_integration_test.go`
- Create: `cli/internal/account/client.go`
- Create: `cli/internal/account/client_test.go`
- Modify: `cli/cmd/myference/main.go`
- Modify: `server/cmd/myference-server/main.go`
- Create: `web/src/features/auth/ConnectWallet.tsx`
- Create: `web/src/features/auth/DeviceApproval.tsx`
- Create: `web/src/features/auth/ApiKeys.tsx`
- Create: `web/src/lib/chain.ts`
- Create: `web/src/lib/api.ts`
- Create: `web/src/features/auth/auth.test.tsx`

- [x] **Step 1: Write failing auth-boundary tests**

Test unsupported-chain refusal, nonce-bound wallet authentication, expired device code, exact machine identity display, one-time API-key reveal, scope display, and revocation. Use a real loopback HTTP test server; controlled wallet signatures are test inputs, not a fake production login path.

- [x] **Step 2: Verify RED**

Run: `npm --prefix web test -- --run src/features/auth/auth.test.tsx`

Expected: FAIL because auth features do not exist.

- [x] **Step 3: Implement secure auth flows**

Define Monad testnet chain ID 10143 with environment-provided RPC and explorer URLs. Request a server nonce, sign domain/chain/origin/nonce/issued/expiry fields, exchange it for a secure HTTP-only session, approve device codes, and create scoped API keys that reveal plaintext once. Never store bearer tokens in local storage.

- [x] **Step 4: Verify and commit**

Run: `npm --prefix web test -- --run src/features/auth/auth.test.tsx && npm --prefix web run build`

Expected: PASS.

Commit: `git add web && git commit -m "feat: connect wallets devices and API keys"`

### Task 14: Implement live marketplace and request activity

**Files:**
- Create: `migrations/000005_marketplace_realtime.sql`
- Create: `migrations/000006_request_submission.sql`
- Create: `server/internal/store/marketplace.go`
- Create: `server/internal/api/marketplace.go`
- Modify: `server/internal/realtime/events.go`
- Modify: `server/internal/chain/settlement.go`
- Modify: `server/internal/chain/indexer.go`
- Modify: `server/cmd/myference-server/main.go`
- Create: `web/src/features/marketplace/ModelList.tsx`
- Create: `web/src/features/marketplace/ModelDetail.tsx`
- Create: `web/src/features/activity/RoutingRail.tsx`
- Create: `web/src/features/activity/Activity.tsx`
- Create: `web/src/lib/realtime.ts`
- Create: `web/src/features/marketplace/marketplace.test.tsx`

- [x] **Step 1: Write failing real-data state tests**

Test loading, empty, available, stale, disconnected, reconnect-gap, and provider-pinned states. Assert all visible prices are all-inclusive MON values and model/provider rows originate from API responses. Test request transitions through reserved, routed, streaming, signed, submitted, and confirmed without skipping or reopening terminal states.

- [x] **Step 2: Verify RED**

Run: `npm --prefix web test -- --run src/features/marketplace/marketplace.test.tsx`

Expected: FAIL because marketplace and realtime features do not exist.

- [x] **Step 3: Implement query and realtime reconciliation**

Fetch models/offers through TanStack Query. Obtain a short-lived stream ticket, connect native `EventSource`, apply monotonically increasing event IDs, and refetch authoritative queries after a gap or reconnect. Render no inventory until the API responds.

- [x] **Step 4: Verify accessibility and commit**

Run: `npm --prefix web test -- --run && npm --prefix web run lint && npm --prefix web run build`

Expected: PASS, including Axe checks for marketplace routes.

Commit: `git add web && git commit -m "feat: show live inference marketplace"`

### Task 15: Implement billing and provider operations ✅

**Files:**
- Create: `web/src/features/billing/Billing.tsx`
- Create: `web/src/features/billing/SpendingSession.tsx`
- Create: `web/src/features/provider/Machines.tsx`
- Create: `web/src/features/provider/Offers.tsx`
- Create: `web/src/features/provider/Earnings.tsx`
- Create: `web/src/lib/amount.ts`
- Create: `web/src/features/billing/billing.test.tsx`
- Create: `web/src/features/provider/provider.test.tsx`

- [ ] **Step 1: Write failing integer-amount and transaction tests**

Test decimal MON-to-wei parsing without floating point, deposit simulation, wrong network, rejected wallet, contract revert, pending/finalized deposits, bounded sessions, delayed close, provider bond, immutable offer version, claimable earnings, and explorer links.

- [ ] **Step 2: Verify RED**

Run: `npm --prefix web test -- --run src/features/billing src/features/provider`

Expected: FAIL because billing and provider features do not exist.

- [ ] **Step 3: Implement simulated contract writes and confirmed reads**

Use Viem to simulate every write, request the wallet transaction, display its hash as pending, and wait for configured finality plus indexer observation before updating available balances or routability. Parse and format integer wei only.

- [ ] **Step 4: Verify and commit**

Run: `npm --prefix web test -- --run && npm --prefix web run lint && npm --prefix web run build`

Expected: PASS.

Commit: `git add web && git commit -m "feat: manage Myference billing and providers"`

### Task 16: Prove the real Monad testnet vertical slice

**Files:**
- Create: `scripts/e2e-testnet.sh`
- Create: `docs/demo.md`
- Modify: `README.md`
- Modify: `Makefile`
- Modify: `web/`

- [ ] **Step 1: Write the executable acceptance script**

The script must reject missing RPC URL, deployed contract, funded customer/provider wallets, PostgreSQL URL, broker URL, Windows machine ID, and real Ollama model. It must never substitute local Anvil, a canned response, or fake transaction hashes.

- [ ] **Step 2: Deploy and verify on Monad testnet**

Run: `forge script contracts/script/Deploy.s.sol --root contracts --rpc-url "$MONAD_TESTNET_RPC_URL" --broadcast --verify`

Expected: a successful deployment transaction and explorer-visible contract address on chain ID 10143.

- [ ] **Step 3: Run the real workflow**

Use the web client to deposit provider collateral, publish the discovered Ollama offer, deposit customer MON, open a capped session, and create the real API key. Start the Windows daemon, call the hosted OpenAI-compatible streaming endpoint, observe the real routing rail, co-sign the observed receipt, settle it, and withdraw provider/platform claimable balances.

- [ ] **Step 4: Capture objective evidence**

Record contract address, transaction hashes, request ID, model name, input/output usage, total charge, provider amount, fee amount, Windows CLI version, broker commit SHA, and explorer URLs in `docs/demo.md`. Never include private keys, bearer tokens, prompts, or full outputs.

- [ ] **Step 5: Run the full release gate**

Run: `make verify && npm --prefix web test -- --run && npm --prefix web run build && ./scripts/e2e-testnet.sh`

Expected: all automated checks pass and the real testnet acceptance script exits 0 with explorer links.

- [ ] **Step 6: Build, tag, and commit**

Run: `GOOS=windows GOARCH=amd64 go build -trimpath -o dist/myference-windows-amd64.exe ./cli/cmd/myference && go build -trimpath -o dist/myference-server ./server/cmd/myference-server`

Expected: both builds exit 0.

Commit: `git add scripts docs README.md Makefile && git commit -m "release: prove Monad testnet inference settlement"`

## Execution rule

No adapter, balance, settlement, or availability status appears in the demo or public API until its real integration check passes. Unit tests may use controlled inputs to prove edge cases; release acceptance always uses real external components and real Monad testnet transactions.
