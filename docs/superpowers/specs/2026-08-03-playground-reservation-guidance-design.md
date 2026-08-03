# Playground Reservation Guidance

## Problem

The live `gpt-5.6-luna` offer is healthy and has capacity, but the playground omits an output-token limit. The API therefore reserves against its safe 4,096-token default. A small Luna prompt needs about 1.46 MON under that bound, while the customer's selected spending session has 1 MON remaining. The router reports this economic rejection as `no eligible provider`, which incorrectly suggests a hosting failure.

## Design

- The playground sends an explicit output-token ceiling, defaulting to 256 tokens, and defaults the per-request maximum to 1 MON.
- It calculates the same conservative reservation used by the server from UTF-8 prompt bytes, the selected live offer's published input/output/compute rates, a 256-token default output ceiling, and the existing 120-second compute ceiling.
- The estimate is shown in MON. Submission is stopped locally with a budget-specific explanation when the entered request maximum is below it.
- The router returns a distinct insufficient-budget error only when at least one candidate passes model, capability, bond, health, capacity, and pricing checks but exceeds the request maximum or selected session balance.
- The API maps that condition to HTTP 402 with an actionable message. Truly unavailable providers continue returning HTTP 503.
- API callers that omit an output ceiling retain the server's existing safe 4,096-token default.

## Verification

- Router tests distinguish budget rejection from runtime/provider rejection.
- API integration coverage verifies the HTTP 402 message.
- Playground tests verify the 256-token request field, default 1 MON ceiling, calculated estimate, and request payload.
- The full repository verification gate and deployed browser flow are checked before completion.
