# OpenAI Completion Token Compatibility Design

## Problem

The OpenAI-compatible backend sends `max_tokens` to `/v1/chat/completions`. OpenAI rejects that field for `gpt-5.6-luna` and requires `max_completion_tokens`, causing provider jobs to terminate with `backend_failed`.

## Design

Send `max_completion_tokens` for every OpenAI-compatible chat-completions generation request. Keep the endpoint, streaming protocol, usage accounting, default limit, and post-response output-limit validation unchanged. This uses the modern field already accepted by the configured OpenAI models and avoids model-name-specific branching.

## Verification

An integration test will decode the actual request received by a local HTTP server and require the configured limit under `max_completion_tokens` while rejecting the legacy `max_tokens` field. Focused package tests, the full Go suite, a rebuilt installed CLI, and a live provider request will verify the change end to end.
