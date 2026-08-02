# Reservation and settlement finality design

## Goal

Prevent a customer from spending the same session allowance on concurrent or not-yet-finalized inference requests while reserving no more than the selected provider can charge.

## Invariants

1. The customer-provided maximum spend is an authorization ceiling, not the amount held.
2. Admission holds the selected offer's exact worst-case charge, recomputed transactionally from the immutable request limits and current selected rate version.
3. A zero-cost or overflowing route is ineligible. Every billable request has a positive hold representable by the protocol's `uint64` receipt fields.
4. While inference is running, the hold equals the worst-case charge.
5. After usage is validated and priced, the hold shrinks atomically to the exact charge; it is never released before confirmed settlement.
6. Abort before a billable receipt releases the full hold. A usage-limit violation is failed and unbilled.
7. Confirmed `ReceiptSettled` releases the exact hold in the same database transaction that reduces the indexed session balance.
8. An indexer rewind restores holds for reverted settlements before rebuilding chain projections.
9. Financial holds and provider execution slots are independent: completed/signed/submitted requests retain money holds but do not consume host capacity.
10. Contract checks remain authoritative: actual charge must equal on-chain rates, remain within the signed maximum, and fit the session allowance.

## Lifecycle

`reserve` computes and holds worst case. `complete` validates usage and stores the proposal while retaining the hold. `prepare receipt` calculates actual charge using the indexed immutable on-chain offer and atomically shrinks the hold. Provider and settlement signatures then move the request through `signed` and `submitted`. The finalized chain event moves it to `settled`, decreases confirmed balance, and releases the hold.

Failures before a co-signed receipt release everything, including provider signature timeout or rejection. Once a co-signed receipt is queued, its exact hold remains because it can still be settled and must be retried operationally rather than silently abandoned. Session close requests stop new admission; the on-chain close delay is the settlement window for pending receipts.

## Customer balance policy

There is no global minimum inference-user deposit. A request requires an open session with enough confirmed, unreserved allowance for that request's computed worst-case charge. The API key ceiling and `X-Myference-Max-Spend` must also cover that amount. The customer's wallet separately needs enough native MON for its own deposit/session/withdrawal gas transactions.

## Edge cases covered

- Concurrent reservations serialize per session.
- Prices changing after routing cannot change an existing request's snapshotted version and cap.
- Provider-reported usage above any hard limit fails without a receipt.
- Exact arithmetic uses ceiling division and rejects `uint64` overflow.
- Monad session balances remain arbitrary precision; only the router's per-request view saturates at the receipt protocol's `uint64` charge ceiling (about 18.44 MON).
- Zero-priced routes are rejected instead of conflicting with positive-hold schema and nonce semantics.
- Repeated completion, settlement events, aborts, and indexer replay are idempotent.
- Client disconnects and provider timeouts release holds only while no billable receipt exists.
- Relay decode errors, missing stream support, and acceptance timeouts cancel provider work and release their holds.
- Settlement reorgs restore both request state and the financial hold.
- Settled event value must match the locally held exact charge when a local reservation exists.
- Host heartbeats cannot reopen an executing slot, and financial finality holds cannot keep a completed host slot closed.
- Settlement transactions contain one receipt so a stale or invalid receipt cannot revert unrelated providers' payouts.
