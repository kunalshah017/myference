# Connected provider routing design

## Problem

Routing state can contain multiple candidates for one model. A historical machine may remain healthy with non-zero capacity in the database after its heartbeat becomes stale. The router currently ranks candidates before checking whether their relay is connected. If a disconnected historical machine wins that ranking, the API returns `no eligible provider` without considering a connected candidate for the same model.

This occurred for `qwen2.5:0.5b`: the stale `mach_I239…` candidate won the deterministic tie-break ahead of the live `mach_izP5…` laptop.

## Design

The OpenAI request handler will mark candidates whose machine is not connected to its relay hub as unavailable before calculating costs and calling `router.Select`. The existing router will then rank only candidates that pass all current requirements:

- requested model and capabilities;
- confirmed provider bond;
- healthy, non-zero capacity;
- valid non-zero pricing;
- request and session budgets;
- optional machine pin.

The post-selection connection guard remains as a race-safety check because a provider can disconnect between candidate filtering and job dispatch.

## Alternatives rejected

- Filtering only by database heartbeat age still leaves a window where a disconnected machine appears healthy until the timeout expires.
- Selecting first and retrying after failure complicates reservation and dispatch behavior. Pre-filtering expresses relay connectivity as an eligibility condition before ranking.

## Testing

Add an OpenAI handler regression with two otherwise eligible candidates: the first-ranked machine is disconnected and the second is connected. A valid request must route to the connected machine instead of returning `no eligible provider`.

Existing tests continue to cover bond, capacity, pricing, capability, budget, and relay-disconnect behavior.
