# OpenAI Completion Token Compatibility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent OpenAI-backed provider jobs from failing when a model requires `max_completion_tokens`.

**Architecture:** Change only the OpenAI-compatible request boundary. A local HTTP integration test observes the serialized payload, then the adapter replaces the legacy field without changing streaming or accounting behavior.

**Tech Stack:** Go standard library, `net/http/httptest`, OpenAI-compatible chat completions API.

## Global Constraints

- Keep `/v1/chat/completions`, streaming, usage accounting, and output-limit validation unchanged.
- Do not add model-name-specific branches.

---

### Task 1: Use the modern completion-limit field

**Files:**
- Modify: `cli/internal/backend/openai/openai_integration_test.go`
- Modify: `cli/internal/backend/openai/openai.go`

**Interfaces:**
- Consumes: `backend.Request.MaximumOutputTokens uint64`
- Produces: OpenAI-compatible request JSON containing `max_completion_tokens` and no `max_tokens`

- [ ] **Step 1: Write the failing integration test**

Add a request-body assertion to the real adapter integration test. Decode the POST body and require literal `max_completion_tokens: 17`; fail if `max_tokens` is present.

- [ ] **Step 2: Verify the regression test fails**

Run: `go test ./cli/internal/backend/openai -run TestClientUsesCompletionTokenLimitField -v`

Expected: FAIL because the request contains `max_tokens` instead of `max_completion_tokens`.

- [ ] **Step 3: Implement the minimal adapter change**

Replace the request payload key `max_tokens` with `max_completion_tokens` in `Client.Generate`.

- [ ] **Step 4: Verify focused and full tests**

Run: `gofmt -w cli/internal/backend/openai/openai.go cli/internal/backend/openai/openai_integration_test.go`, `go test ./cli/internal/backend/openai -v`, and `go test ./...`.

Expected: PASS.

- [ ] **Step 5: Rebuild, install, and verify live behavior**

Build `./cli/cmd/myference`, replace the installed executable while the Scheduled Task is stopped, restart it once, and verify Luna returns streamed output without becoming unhealthy.

- [ ] **Step 6: Commit and push**

Commit the spec, plan, test, and implementation, then push `main` to `origin`.
