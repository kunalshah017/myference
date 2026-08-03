# Uint256 Provider Rate Routing Design

## Problem

Provider offers store their three rates as Solidity `uint256` values and PostgreSQL `numeric(78,0)` values. The broker currently converts those exact decimal values to `uint64` while reconciling a provider heartbeat. Normal OpenAI-priced offers exceed `uint64` in wei, so otherwise valid offers are silently skipped and never enter `provider_routing_state` or the public marketplace.

## Goals

- Reconcile and publicly list existing offers whose rates fit Solidity `uint256`, including the four OpenAI-priced models already running on the Windows provider.
- Calculate each request's maximum charge exactly from wide rates and only route it when the final charge fits the existing `uint64` request and receipt fields.
- Preserve the deployed contract, existing offer versions, database schema, provider protocol, signed receipt format, and one-command CLI installation flow.
- Reject malformed or economically unsafe data explicitly; do not truncate, wrap, or silently accept it.

## Constraints

- Rate values remain non-negative base-10 wei strings at database and router boundaries.
- A valid rate must be canonical decimal and no greater than `2^256 - 1`.
- Request maximum spend, session balance, reservations, and signed receipt totals remain `uint64`.
- Existing on-chain offers must become routable after a server deployment and the provider's next heartbeat; users must not republish them.
- No contract deployment or database migration is required.

## Design

### Router monetary boundary

`router.Candidate` carries `InputPerMillion`, `OutputPerMillion`, and `ComputePerSecond` as exact decimal strings. A router-local parser validates canonical unsigned decimal syntax and the Solidity `uint256` upper bound. A small pricing predicate reports whether at least one validated rate is non-zero.

`WorstCaseCost` parses each rate into a fresh `big.Int`, computes each component with ceiling division, and adds the components without intermediate narrowing:

```text
ceil(maximumInputTokens * inputPerMillion / 1,000,000)
+ ceil(maximumOutputTokens * outputPerMillion / 1,000,000)
+ ceil(maximumComputeMilliseconds * computePerSecond / 1,000)
```

The result is returned only if it fits `uint64`. Invalid rates and an oversized final charge return `router.ErrNoEligibleProvider`. This preserves the existing route-selection, reservation, and receipt interfaces while allowing economically realistic wei rates.

### Storage reconciliation and candidate loading

`ReconcileProviderCapacity` validates the exact chain-indexed rate strings with the shared router rules. It uses `big.Int` to calculate the persisted aggregate `maximum_cost` placeholder, rather than summing three `uint64` values. Metering-mode eligibility checks validated decimal strings for non-zero input/output pricing.

`RoutingCandidates` scans the three rates directly into their exact string fields. If the persisted aggregate `maximum_cost` exceeds `uint64`, it safely saturates the in-memory placeholder to `math.MaxUint64`; request handlers replace that placeholder with a request-specific bounded calculation before selection. Malformed persisted values remain an error rather than being silently truncated.

`ReserveInference` loads exact rate strings and recalculates the requested maximum charge using the same router function. This is the transactional enforcement boundary: a stale, malformed, or newly unaffordable route is rejected before funds are reserved.

### API routing

The OpenAI-compatible handler calculates a request-specific maximum cost for every priced candidate. A valid calculation replaces the persisted placeholder. Invalid rates or a result above `uint64` make that candidate ineligible. Zero-priced candidates retain existing behavior and remain rejected by route selection.

### Settlement compatibility

Receipt calculation already reads rate strings and performs exact `big.Int` arithmetic before requiring the final total to fit `uint64`. It remains unchanged. The signed `maximumCharge` and `totalCharge` fields, request limits, and deployed Solidity interfaces therefore remain byte-for-byte compatible.

## Error Handling

- Empty, signed, non-decimal, non-canonical, negative, or greater-than-`uint256` rate values are ineligible.
- Arithmetic never narrows an intermediate value.
- A request whose computed maximum charge exceeds `uint64`, maximum spend, or confirmed session balance receives the existing no-eligible-provider outcome.
- A reservation whose recalculated cost is unsafe receives `ErrIneligibleRoute`.
- Valid `uint256` rates are not skipped merely because an individual rate exceeds `uint64`.

## Testing

- Router unit tests prove that a rate such as `292000000000000000000` is accepted when a small request produces a bounded result, and that invalid/greater-than-`uint256` rates and final-charge overflow are rejected.
- Store integration tests insert wide chain rates, reconcile a provider heartbeat, assert the candidate preserves exact values, and assert the public marketplace exposes the offer unchanged.
- Reservation integration tests prove bounded requests reserve successfully and oversized requests are rejected.
- API tests prove a wide-rate candidate receives a request-specific bounded maximum and can be selected.
- The complete Go suite, CLI tests, repository verification, and deployment health checks run before push/completion.

## Deployment and Success Criteria

This is a server-only compatibility fix. After it reaches `main`, the normal Windows provider service remains running and its next authenticated heartbeat reconciles the existing offers. Success means all four healthy requested models appear in the public marketplace with their exact configured rates, without headless mode, contract changes, schema changes, or offer republication.
