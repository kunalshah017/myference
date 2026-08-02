# Account Analytics Design

## Goal

Expose historical customer usage and provider economics using only confirmed request receipts and Monad contract events.

## Persistence

Migration `000013_account_analytics.sql` creates `chain_slashes`, keyed by chain, contract, request, provider, and transaction. It stores the confirmed slash amount, block number, transaction hash, and indexed timestamp.

Settlements remain canonical in `chain_settlements`. Usage dimensions are joined from `receipt_proposals`, which contains signed input tokens, output tokens, and compute milliseconds. Customer ownership comes from `sessions` and `accounts`; provider ownership comes from `machines` and `accounts`.

The chain indexer inserts a slash ledger row when handling `ProviderSlashed`, then updates the provider bond. Rewind deletes slashes alongside settlements and other derived chain state, allowing canonical replay.

## API

Authenticated `GET /api/account/analytics` returns:

- customer totals: settled requests, input tokens, output tokens, compute milliseconds, provider charges, protocol fees, and total spent;
- provider totals: settled requests, input tokens, output tokens, compute milliseconds, gross revenue, and total slashed;
- daily customer/provider aggregates for the last 30 UTC days;
- recent settled usage records;
- recent provider settlements;
- recent slashing events.

All integer token/time values are JSON numbers. All wei values are decimal strings to avoid JavaScript precision loss. Empty accounts return zero totals and empty lists.

## UI

The Usage view shows confirmed totals, a CSS daily activity chart, and a recent settlement ledger. The Earnings view shows confirmed revenue, served tokens/compute, collateral, daily revenue, settlement history, and slash history. Realtime account events invalidate analytics queries so dashboards update without fabricated interim values.

## Privacy and limits

Analytics never store or return prompt content, response content, input/output hashes, API keys, or workspace files. Recent lists are capped at 100 records. Daily series is limited to 30 UTC days.

## Verification

Integration tests cover migration shape, indexer slash insertion and rewind, account-scoped aggregation, API authentication and response encoding, and React rendering of totals/history. Existing chain, store, API, and web suites remain green.
