# Reservation finality implementation plan

**Goal:** Hold the computed worst-case request charge during execution and the exact charge until confirmed Monad settlement, with reorg-safe accounting.

**Architecture:** Keep `inference_reservations` as the financial hold ledger. Admission writes the transactionally recomputed worst-case amount. Receipt preparation shrinks it to actual charge. The finalized event releases it; rewind reactivates it. Provider capacity counts only executing request states.

**Tech stack:** Go, PostgreSQL migrations/integration tests, Foundry contract tests.

### Task 1: Specify failing accounting behavior

- Update store integration coverage to assert worst-case rather than user-cap reservation.
- Assert completion retains worst case and receipt preparation shrinks to exact charge.
- Assert another request sees the exact pending charge until settlement.
- Add zero-cost and overflow admission tests.

### Task 2: Implement admission and exact-charge holds

- Change `ReserveInference` to persist computed worst case.
- Reject zero worst-case routes.
- Keep the financial reservation active at completion.
- Shrink the active reservation during `PrepareReceipt` under the session lock.
- Count only executing states when reconciling provider capacity.

### Task 3: Couple holds to finalized chain events

- Validate local hold against decoded settlement total.
- Release the hold in the `ReceiptSettled` projection transaction.
- Reactivate reverted settlement holds during indexer rewind.
- Extend indexer integration tests for settlement and rewind behavior.

### Task 4: Audit and verify

- Run focused Go tests, PostgreSQL integration tests, race tests where practical, full Go tests, web checks, and Foundry tests.
- Review cancellation, retry, duplicate delivery, session closing, fee/rate changes, zero usage, overflow, settlement failure, and reorg paths.
- Commit and push only after fresh verification succeeds.

