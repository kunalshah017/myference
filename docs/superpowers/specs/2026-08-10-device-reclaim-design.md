# Device Reclaim Design

## Goal

Allow a clean CLI reinstall to reconnect a previously registered machine name without deleting account, offer, or routing history.

## Authorization and Identity

Device exchange remains gated by an approved, unexpired, one-time browser authorization. When the approved account already owns a machine with the requested name, exchange reuses that machine ID instead of inserting a duplicate. A machine with the same name under another account is unrelated and cannot be reclaimed.

Reclaiming updates the machine signer, clears machine revocation, replaces the machine token, and marks the device authorization consumed in one database transaction. Replacing the token immediately invalidates credentials from the previous local installation. Existing machine-linked records remain attached to the stable machine ID.

When no matching machine exists, exchange retains the existing new-machine behavior.

## Failure Handling

The transaction rolls back completely if machine update, token rotation, authorization consumption, or commit fails. Expired, pending, consumed, and invalid device codes retain their existing errors.

## Testing

A PostgreSQL integration test performs two approved exchanges for the same account and machine name. It requires a stable machine ID, the new signer and token to authenticate, the old token to fail, and exactly one machine row. Existing new-machine, expiry, one-time, and revocation tests remain green.
