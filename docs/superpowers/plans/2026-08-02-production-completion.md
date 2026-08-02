# Myference Production Completion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the real headless provider-to-Monad settlement path, both public API dialects, secure disposable coding workspaces, Windows/macOS lifecycle packaging, and an evidence-producing Monad testnet release gate.

**Architecture:** Provider wallets authorize a per-machine EVM signer on `MyferenceMarket`; the CLI stores that machine key in the OS credential vault and signs broker-proposed EIP-712 usage receipts without exposing the provider wallet. PostgreSQL projects finalized Monad sessions into operational spend limits, the broker validates and co-signs receipts, a durable worker broadcasts batches, and the indexer alone confirms balances. OpenAI and Anthropic handlers share one inference service; Ollama, OpenAI-compatible cloud endpoints, and isolated command agents implement one backend interface.

**Tech Stack:** Go 1.26, Solidity 0.8.30/Foundry, PostgreSQL 17, WebSocket/SSE, React/TypeScript/Viem, Windows Credential Manager, macOS Keychain/launchd, native MON on Monad chain 10143.

---

### Task 1: Project finalized Monad sessions into API spending sessions

**Files:**
- Modify: `server/internal/chain/indexer.go`
- Test: `server/internal/chain/client_integration_test.go`

- [x] **Step 1: Write the failing real-chain projection test**

Create an account whose wallet matches the Anvil customer, sync `SessionOpened`, `SessionCloseRequested`, and `ReceiptSettled`, then query `sessions` and require the exact account, `closing` state, and remaining allowance.

- [x] **Step 2: Run the test and verify RED**

Run: `MYFERENCE_TEST_RPC_URL=http://127.0.0.1:8546 MYFERENCE_TEST_DATABASE_URL=postgres://myference:myference@127.0.0.1:5432/myference_test?sslmode=disable go test ./server/internal/chain -tags=integration -run TestClientDeploysAndSettlesActualMyferenceContract -count=1 -v`

Expected: FAIL with `sql: no rows in result set` for the operational session.

- [x] **Step 3: Implement the projection**

On finalized `SessionOpened`, insert the operational session only when the indexed customer matches an authenticated account. Project close requests to `closing`, finalization to `closed`, and settled charges to the remaining confirmed allowance.

- [x] **Step 4: Verify GREEN**

Run the Step 2 command. Expected: PASS against actual Anvil and PostgreSQL.

### Task 2: Authorize secure headless machine receipt signers

**Files:**
- Modify: `contracts/src/MyferenceMarket.sol`
- Modify: `contracts/test/MyferenceMarket.t.sol`
- Create: `migrations/000008_machine_signers.sql`
- Modify: `server/internal/auth/device.go`
- Modify: `server/internal/auth/browser.go`
- Modify: `server/internal/auth/http.go`
- Modify: `cli/internal/account/client.go`
- Modify: `cli/cmd/myference/main.go`
- Modify: `web/src/features/auth/DeviceApproval.tsx`
- Modify: `web/src/lib/marketContract.ts`

- [x] **Step 1: Write and run the delegated-signer contract test**

Require an unauthorized machine signature to revert, an explicitly authorized signer to settle, and the same signer to fail after revocation.

Run: `forge test --root contracts --match-test testProviderCanAuthorizeAndRevokeHeadlessMachineSigner -vv`

Expected RED: `setProviderSigner` does not exist.

- [x] **Step 2: Implement delegated signer verification**

Add `providerSigners[provider][signer]`, `ProviderSignerSet`, and `setProviderSigner`. Accept either the provider wallet or an authorized signer in settlement and double-sign evidence verification.

- [x] **Step 3: Bind signer identity to browser device authorization**

Generate the machine secp256k1 key before login, store only its public address in `device_authorizations` and `machines`, show that address during exact-device approval, and store the private key only under `myference.signer/<machine-id>` in Credential Manager or Keychain. Reject malformed, zero, duplicate, or changed signer addresses.

- [x] **Step 4: Require the wallet transaction before routing**

Add `setProviderSigner(address,bool)` to the web writer, simulate it, submit it from the connected provider wallet, and wait for the configured confirmations/indexer observation. The broker must reject capacity from a machine whose signer is not authorized by the offer owner.

- [x] **Step 5: Verify and commit**

Run: `forge test --root contracts && MYFERENCE_TEST_DATABASE_URL=postgres://myference:myference@127.0.0.1:5432/myference_test?sslmode=disable go test ./server/internal/auth ./cli/cmd/myference -count=1 && npm --prefix web test -- --run`

Expected: all tests PASS.

### Task 3: Build, sign, co-sign, queue, and settle every completed receipt

**Files:**
- Create: `protocol/v1/signing.go`
- Create: `protocol/v1/signing_test.go`
- Create: `server/internal/settlement/coordinator.go`
- Create: `server/internal/settlement/coordinator_integration_test.go`
- Modify: `server/internal/store/inference.go`
- Modify: `server/internal/relay/hub.go`
- Modify: `cli/internal/provider/daemon.go`
- Modify: `server/cmd/myference-server/main.go`

- [x] **Step 1: Write failing EIP-712 parity and full-pipeline tests**

Assert the offline Go digest exactly equals `MyferenceMarket.hashReceipt`; then execute real Ollama output through relay, receive the broker proposal in the CLI, sign with the machine key, co-sign with the configured settlement key, enqueue once, broadcast, index, and observe `settled` plus exact provider/platform amounts.

- [x] **Step 2: Verify RED**

Run: `go test ./protocol/v1 -run TestReceiptDigestMatchesContract -count=1 && go test ./server/internal/settlement -tags=integration -count=1 -v`

Expected: FAIL because the digest and coordinator do not exist.

- [x] **Step 3: Implement exact receipt construction**

Derive the on-chain request ID as `keccak256(transport request ID)`, use the indexed bytes32 session, exact provider/offer/model/capability fields, ceiling rate arithmetic, current indexed fee version, a transactionally allocated per-provider nonce, and persisted input/output hashes. Reject any field that cannot be proven from indexed or measured data.

- [x] **Step 4: Implement the signing handshake and durable workers**

After output persistence, send `receipt_proposal`; the CLI verifies request/offer/usage/maximum fields and signs the EIP-712 digest; the coordinator recovers the authorized machine signer, broker-signs, and calls `SettlementQueue.Enqueue`. Start bounded indexer and batch-settlement loops from server configuration with graceful shutdown and retry backoff.

- [x] **Step 5: Verify GREEN and crash recovery**

Run the Step 2 commands plus `go test ./server/internal/chain -tags=integration -count=1 -v`. Expected: PASS, including replay, reconnect, duplicate signature, crash-before-broadcast, already-known transaction, and indexed confirmation.

### Task 4: Add Anthropic Messages and reusable inference service

**Files:**
- Create: `server/internal/inference/service.go`
- Create: `server/internal/api/anthropic.go`
- Create: `server/internal/api/anthropic_integration_test.go`
- Modify: `server/internal/api/openai.go`
- Modify: `server/cmd/myference-server/main.go`

- [ ] **Step 1: Write a failing real-relay Anthropic streaming test**

POST `/v1/messages` with `x-api-key`, `anthropic-version`, model, messages, `stream:true`, and max-spend. Require Anthropic `message_start`, `content_block_delta`, `message_delta`, and `message_stop` events sourced from the real relay output and one shared persisted receipt.

- [ ] **Step 2: Verify RED**

Run: `go test ./server/internal/api -run TestAnthropicStreamingUsesRealRelay -count=1 -v`. Expected: HTTP 404.

- [ ] **Step 3: Extract and implement the shared service**

Move authorization, selection, reservation, leasing, metering, cancellation, and persistence behind one typed streaming service. Keep protocol-specific validation and SSE rendering in the two thin HTTP adapters; enforce body limits, endpoint scopes, request cancellation, and no non-streaming fallback.

- [ ] **Step 4: Verify compatibility**

Run: `go test ./server/internal/api -count=1 -v`. Expected: OpenAI and Anthropic real-relay suites PASS.

### Task 5: Add cloud endpoints and disposable CLI-agent workspaces

**Files:**
- Create: `cli/internal/backend/openai/openai.go`
- Create: `cli/internal/backend/openai/openai_integration_test.go`
- Create: `cli/internal/backend/command/command.go`
- Create: `cli/internal/backend/command/workspace.go`
- Create: `cli/internal/backend/command/command_integration_test.go`
- Modify: `cli/internal/config/config.go`
- Modify: `cli/cmd/myference/main.go`
- Modify: `protocol/v1/messages.go`

- [ ] **Step 1: Write failing adapter and sandbox tests**

Use a real loopback OpenAI-compatible HTTP server to prove streaming translation and local secret use. Execute a real helper process in a fresh `0700` temporary directory and prove it cannot inherit Myference secrets, `HOME`, Git credentials, or files outside the workspace; reject archive traversal, symlinks, device files, excessive file count/bytes, and command timeout.

- [ ] **Step 2: Verify RED**

Run: `go test ./cli/internal/backend/openai ./cli/internal/backend/command -count=1 -v`. Expected: packages absent.

- [ ] **Step 3: Implement minimal adapters**

Store cloud keys in the OS credential vault, never config. Support explicit executable allow-list entries for Codex, Claude Code, and Kimi; run without a shell, with a minimal environment and disposable working directory, stream stdout, cap stderr, kill the process tree on cancel, and delete the workspace on every exit.

- [ ] **Step 4: Expose independent lifecycle**

Allow `backend add/list/start/stop` for `ollama`, `openai`, `codex`, `claude`, and `kimi`. Starting or stopping one backend must update capacity without stopping other providers or the relay daemon.

- [ ] **Step 5: Verify GREEN**

Run: `go test ./cli/... -count=1 -race`. Expected: PASS with no leaked workspace or credential in output/config.

### Task 6: Complete macOS lifecycle and signed release artifacts

**Files:**
- Create: `cli/internal/platform/darwin/lifecycle.go`
- Create: `cli/internal/platform/darwin/lifecycle_test.go`
- Create: `packaging/macos/com.myference.provider.plist`
- Create: `packaging/windows/install.ps1`
- Create: `scripts/build-release.sh`
- Modify: `cli/cmd/myference/main.go`
- Modify: `Makefile`

- [ ] **Step 1: Write failing lifecycle tests**

Require launchd install/start/drain/stop/status parity with Windows, fixed absolute executable/config/log paths, `KeepAlive` restart policy, `ProcessType=Background`, no embedded secrets, and idempotent install/uninstall.

- [ ] **Step 2: Implement macOS lifecycle**

Generate the per-user LaunchAgent atomically, validate it with `plutil`, use `launchctl bootstrap/bootout/kickstart`, and keep credentials in Keychain. A graceful stop drains leases before launchd shutdown.

- [ ] **Step 3: Build reproducible archives**

Cross-build `windows/amd64` and native `darwin/arm64` plus `darwin/amd64`, generate SHA-256 checksums and version metadata from a clean commit, and fail if binaries contain known test tokens/private keys.

- [ ] **Step 4: Verify**

Run: `go test ./cli/internal/platform/... -count=1 && ./scripts/build-release.sh`. Expected: PASS and real archives/checksums under `dist/`.

### Task 7: Run the no-mock Monad testnet release gate

**Files:**
- Create: `scripts/e2e-testnet.sh`
- Create: `docs/demo.md`
- Modify: `README.md`
- Modify: `Makefile`

- [ ] **Step 1: Implement strict preflight and evidence capture**

Require `MONAD_TESTNET_RPC_URL`, funded customer/provider/settlement wallets, deployed contract, PostgreSQL, hosted HTTPS/WSS broker, physical Windows machine, real installed Ollama model, and explorer API. Validate chain ID 10143 and reject Anvil/localhost/canned output/zero or duplicate transaction hashes.

- [ ] **Step 2: Exercise the real workflow**

Deploy, bond, authorize the machine signer, publish the exact discovered offer, deposit customer MON, open a session, create an API key, stream an OpenAI and Anthropic request, settle, index, claim provider/platform earnings, and verify deltas from RPC receipts and contract reads.

- [ ] **Step 3: Write sanitized objective evidence**

Generate `docs/demo.md` containing commit/version, contract and explorer link, transaction hashes, request IDs, model and measured usage, all-inclusive charge, provider/fee amounts, and CLI platform/version. Refuse to write private keys, bearer tokens, prompts, or full model output.

- [ ] **Step 4: Run the release gate**

Run: `make verify && ./scripts/build-release.sh && ./scripts/e2e-testnet.sh`

Expected: all local checks pass; testnet acceptance exits 0 only after explorer-verifiable transactions and final balances.

- [ ] **Step 5: Commit and push**

Run: `git add . && git commit -m "release: complete real Monad inference marketplace" && git push`

Expected: the feature branch contains no uncommitted changes and GitHub shows the verified release commit.
