# Uint256 Provider Rate Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make existing Solidity `uint256` provider rates routable and publicly visible while preserving `uint64` request and receipt charges.

**Architecture:** Carry rates as exact canonical decimal strings from PostgreSQL into router candidates and use `math/big` for validation and arithmetic. Narrow only the final request-specific charge to `uint64`; existing contracts, rows, provider messages, reservations, and signed receipts remain compatible.

**Tech Stack:** Go, `math/big`, PostgreSQL integration tests, existing relay/API test harness, PowerShell CLI verification.

## Global Constraints

- Preserve the deployed contract, existing offer versions, database schema, provider protocol, and signed receipt format.
- Valid rates are canonical non-negative base-10 strings no greater than `2^256 - 1`.
- Request maximum spend, session balance, reservations, and signed receipt totals remain `uint64`.
- Never truncate or wrap monetary values.
- Existing offers reconcile on the next provider heartbeat; no republication is required.
- Keep the normal Windows provider service running; do not enable headless mode.

---

### Task 1: Exact Router Rate Arithmetic

**Files:**
- Modify: `server/internal/router/router.go`
- Test: `server/internal/router/router_test.go`

**Interfaces:**
- Produces: `Candidate.InputPerMillion string`, `Candidate.OutputPerMillion string`, and `Candidate.ComputePerSecond string`.
- Produces: `ValidRate(string) bool`, `HasPricing(Candidate) bool`, and `SumRates(string, string, string) (string, error)` for storage and API boundaries.
- Preserves: `WorstCaseCost(Candidate, uint64, uint64, uint64) (uint64, error)` and `ErrNoEligibleProvider`.

- [ ] **Step 1: Write failing wide-rate and validation tests**

Add focused tests that demonstrate the desired string API and exact behavior:

```go
func TestWorstCaseCostAcceptsUint256RatesWhenFinalChargeFits(t *testing.T) {
    candidate := Candidate{InputPerMillion: "292000000000000000000", OutputPerMillion: "0", ComputePerSecond: "0"}
    cost, err := WorstCaseCost(candidate, 1, 1, 1)
    if err != nil || cost != 292000000000000 {
        t.Fatalf("cost=%d err=%v", cost, err)
    }
}

func TestRateValidationRejectsNonCanonicalAndAboveUint256(t *testing.T) {
    for _, rate := range []string{"", "01", "-1", "+1", "not-a-number", "115792089237316195423570985008687907853269984665640564039457584007913129639936"} {
        if ValidRate(rate) { t.Fatalf("accepted invalid rate %q", rate) }
    }
}
```

Update existing router fixtures to use decimal strings and add a final-charge overflow assertion using a valid `uint256` rate.

- [ ] **Step 2: Run the router tests and verify RED**

Run: `go test ./internal/router -run 'TestWorstCaseCost|TestRateValidation' -count=1`

Expected: compile failure because candidate rates still require `uint64` and the new validation helpers do not exist.

- [ ] **Step 3: Implement validated exact rate helpers and arithmetic**

In `router.go`, parse a canonical decimal string with `new(big.Int).SetString(value, 10)`, reject leading zeroes except `"0"`, reject signs, and require `BitLen() <= 256`. Make `SumRates` add fresh `big.Int` values and return the decimal sum. Make `HasPricing` require three valid rates and report whether any rate is non-zero. Change `WorstCaseCost` to multiply parsed rates by the request limits, ceil-divide by `1_000_000`, `1_000_000`, and `1_000`, sum exactly, and return `ErrNoEligibleProvider` unless the total fits `uint64`.

- [ ] **Step 4: Run router tests and verify GREEN**

Run: `go test ./internal/router -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the router boundary**

```powershell
git add server/internal/router/router.go server/internal/router/router_test.go
git commit -m "fix: support uint256 provider rates in router"
```

### Task 2: Wide-Rate Reconciliation and Reservations

**Files:**
- Modify: `server/internal/store/inference.go`
- Modify: `server/internal/store/runtime_evidence_test.go`
- Test: `server/internal/store/inference_integration_test.go`

**Interfaces:**
- Consumes: router string rate fields, `ValidRate`, `HasPricing`, `SumRates`, and `WorstCaseCost` from Task 1.
- Preserves: legacy `RoutingState`/`UpsertRoutingState` `uint64` inputs for internal fixtures and `InferenceReservation` `uint64` limits.

- [ ] **Step 1: Write a failing reconciliation integration test for real-sized rates**

Extend `TestCapacityReconcilesOnlyIndexedBondedMonadOffer` with a separately hashed `gpt-5.6-sol` offer whose indexed rates are:

```go
const wideInput = "292000000000000000000"
const wideOutput = "1752000000000000000000"
const wideCompute = "4866666666666667"
```

Reconcile the matching capacity heartbeat, then assert `RoutingCandidates` returns those exact strings and `MarketplaceModel` exposes the same strings. Assert a reservation with one input token, one output token, and one compute millisecond succeeds when its maximum spend and session balance cover the exact bounded result.

- [ ] **Step 2: Run the store integration test and verify RED**

Run: `go test ./internal/store -run TestCapacityReconcilesOnlyIndexedBondedMonadOffer -count=1`

Expected with `MYFERENCE_TEST_DATABASE_URL` configured: FAIL because reconciliation skips rates that exceed `uint64`. Without the variable, start the repository's test PostgreSQL service according to the existing test setup and rerun rather than accepting a skip.

- [ ] **Step 3: Load and reconcile exact decimal rates**

In `RoutingCandidates`, assign the scanned rate strings directly to the candidate. Parse `maximum_cost` with `big.Int`; reject malformed/negative values and saturate only an oversized in-memory placeholder to `math.MaxUint64`.

In `ReconcileProviderCapacity`, replace all three `strconv.ParseUint` calls and overflow checks with `router.ValidRate`, string-aware metering validation, and `router.SumRates`. Persist that exact sum as `maximum_cost` and the original exact rate strings in the existing numeric columns.

Change `meteringRatesEligible` to accept strings and require exact `"0"` input/output rates for `compute_only`. Update its unit tests to pass strings.

In `ReserveInference`, construct a candidate from the scanned strings and use `router.WorstCaseCost`; reject zero, invalid, over-limit, or over-balance results with the existing errors.

- [ ] **Step 4: Run focused and complete store tests**

Run: `go test ./internal/store -run 'TestCapacityReconcilesOnlyIndexedBondedMonadOffer|TestInferenceReservationRoutingAndReceiptAreDurableAndAtomic|TestMetering' -count=1`

Run: `go test ./internal/store -count=1`

Expected: PASS with integration tests executed, not skipped.

- [ ] **Step 5: Commit the storage compatibility fix**

```powershell
git add server/internal/store/inference.go server/internal/store/runtime_evidence_test.go server/internal/store/inference_integration_test.go
git commit -m "fix: reconcile wide on-chain provider rates"
```

### Task 3: API Request-Specific Selection

**Files:**
- Modify: `server/internal/api/openai.go`
- Test: `server/internal/api/openai_integration_test.go`
- Modify: other Go tests containing `router.Candidate` rate literals, as reported by compilation

**Interfaces:**
- Consumes: `router.HasPricing` and `router.WorstCaseCost`.
- Preserves: `Reservation.MaximumSpend uint64` and `v1.JobOffer.MaximumSpend uint64`.

- [ ] **Step 1: Make the OpenAI relay test exercise a wide rate**

Change the real relay candidate to string rates that exceed `uint64` individually but produce a bounded request maximum, and update the expected `Reservation.MaximumSpend` to the exact ceiling-divided result. Keep the end-to-end assertions for relay delivery, reservation, output streaming, and proposal persistence.

- [ ] **Step 2: Run the API test and verify RED**

Run: `go test ./internal/api -run TestOpenAIStreamingUsesRealRelayAndPersistsProposal -count=1`

Expected: compile failure until the handler uses the new pricing predicate and all candidate literals use strings.

- [ ] **Step 3: Update request-specific pricing and remaining fixtures**

Replace numeric comparisons against candidate rate fields with `router.HasPricing`. On calculation failure set `MaximumCost` to `math.MaxUint64`; on valid priced candidates replace it with the exact request-specific result. Convert remaining test-only candidate rate literals to canonical decimal strings without changing their economics.

- [ ] **Step 4: Run API and whole-server tests**

Run: `go test ./internal/api -count=1`

Run: `go test ./... -count=1`

Expected: PASS.

- [ ] **Step 5: Commit API compatibility**

```powershell
git add server/internal/api/openai.go server/internal/api server/internal/router server/internal/store
git commit -m "fix: route requests with wide provider rates"
```

### Task 4: Verification, Integration, Push, and Live Proof

**Files:**
- Verify: `docs/superpowers/specs/2026-08-03-uint256-provider-rate-routing-design.md`
- Verify: `docs/superpowers/plans/2026-08-03-uint256-provider-rate-routing.md`
- Verify: repository working tree and public deployment

**Interfaces:**
- Consumes: completed Tasks 1-3.
- Produces: merged and pushed `main`, deployed broker behavior, and public marketplace evidence.

- [ ] **Step 1: Run formatting and repository verification**

Run `gofmt` on every modified Go file, then run the repository's documented verification commands plus at minimum:

```powershell
go test ./... -count=1
git diff --check
git status --short
```

Expected: all tests PASS, no whitespace errors, and only intended changes remain.

- [ ] **Step 2: Review monetary and compatibility invariants**

Inspect the final diff and confirm there is no numeric-to-`uint64` rate conversion, no schema/contract/protocol modification, no receipt field change, no silent truncation, and no unrelated user change included.

- [ ] **Step 3: Merge the implementation into current `main`**

Fetch the latest origin state, replay or merge the isolated feature branch onto it without discarding user work, rerun focused verification, and create the final integration commit only if the merge itself produces changes.

- [ ] **Step 4: Push and verify CI/deployment**

Push `main` to `origin`, inspect the resulting GitHub checks until complete, and verify the production health endpoint reports the new revision.

- [ ] **Step 5: Verify the live provider and marketplace**

Confirm the `Myference Provider` scheduled task remains running in normal mode, broker connectivity is healthy, and all four configured models are healthy. Poll the public marketplace until `gpt-5.6-sol`, `gpt-5.6-terra`, `gpt-5.6-luna`, and `gpt-5.3-codex` appear with their exact rates. If deployment is asynchronous, continue monitoring rather than switching to headless mode or republishing offers.

- [ ] **Step 6: Report evidence**

Report pushed commit IDs, test/CI results, deployed revision, normal provider task status, and the four public marketplace entries.
