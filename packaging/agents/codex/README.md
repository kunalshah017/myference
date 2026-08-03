# Myference Codex agent image

This image contains the version-pinned OpenAI Codex CLI used inside Myference's disposable command-agent sandbox. It contains no provider credentials, host files, Myference account secrets, MCP configuration, or Docker socket.

Release tags publish a multi-architecture image to `ghcr.io/kunalshah017/myference-codex`. Provider configuration must use the immutable digest printed by the `agent-images` workflow, for example:

```text
ghcr.io/kunalshah017/myference-codex@sha256:<manifest-digest>
```

The CLI supplies `codex exec --ephemeral --sandbox read-only --ask-for-approval never`, a per-job proxy token, and the configured model at runtime. Myference API callers receive streamed model text only; they cannot invoke Codex tools directly.
