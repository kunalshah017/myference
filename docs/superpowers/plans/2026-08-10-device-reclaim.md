# Device Reclaim Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make approved device exchange reclaim an account-owned machine name after a clean reinstall.

**Architecture:** Resolve an existing machine by `(account_id, name)` inside the exchange transaction. Reuse and update it when present, otherwise create it, then replace its single token before consuming the authorization.

**Tech Stack:** Go, `database/sql`, PostgreSQL integration tests, Render deployment.

---

### Task 1: Reproduce machine-name collision

**Files:**
- Modify: `server/internal/auth/device_integration_test.go`

- [ ] Add `TestDeviceAuthorizationReclaimsAccountMachineName` that exchanges the same approved account/name twice and expects the second exchange to succeed with the first machine ID, a rotated signer/token, rejected old token, and one database row.
- [ ] Run the test against `MYFERENCE_TEST_DATABASE_URL`; expect the current duplicate-key failure.

### Task 2: Implement transactional reclaim

**Files:**
- Modify: `server/internal/auth/device.go`

- [ ] In `ExchangeDeviceAuthorization`, select the account/name machine `FOR UPDATE`; insert only when absent, otherwise update signer and clear revocation.
- [ ] Delete the prior `machine_tokens` row and insert the new token in the same transaction.
- [ ] Run the focused integration test and full auth tests; expect PASS.

### Task 3: Verify and deploy

**Files:**
- No additional source files unless verification exposes a defect.

- [ ] Run `make verify` and the Windows CLI cross-tests.
- [ ] Commit and push `main`, monitor GitHub checks and Render deployment, verify production health, then retry clean CLI device authorization.
