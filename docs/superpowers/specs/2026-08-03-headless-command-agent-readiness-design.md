# Headless Command-Agent Readiness Design

## Goal

Make enabled Codex, Claude, and Kimi backends start reliably from the Windows provider and headless scheduled task while preserving Myference's model-only public API and disposable-container security boundary.

## User contract

- The public provider surface returns model output only. It does not expose Codex shell, MCP, filesystem, Docker, or other agent tools to callers.
- The public Windows installer remains `irm https://myference.xyz/install.ps1 | iex` and installs every host-side runtime asset required by Linux Docker containers.
- `serve` and `windows headless run` start Docker Desktop when an enabled command-agent backend needs it, wait for the Linux engine with a bounded deadline, pull only missing digest-pinned images, and fail with an actionable backend/image-specific error.
- `windows doctor` and `windows headless status` are read-only. They report Docker CLI discovery, engine readiness, Linux-container mode, and missing configured images without starting Docker or pulling images.
- Multiple enabled providers remain owned by the shared provider configuration and can be enabled or disabled independently.

## Architecture

Add a focused Windows Docker runtime adapter under `cli/internal/platform/windows`. It accepts narrow command/start/wait dependencies so readiness behavior is unit-testable without Docker Desktop. Its mutating `Prepare` path starts Docker Desktop hidden when necessary, polls `docker info`, enforces Linux-container mode, inspects each unique immutable image, and pulls only an absent digest. Its read-only `Status` path returns structured findings for doctor and headless status.

The normal Windows backend preparation hook invokes the same readiness adapter before backend discovery, so foreground service and headless runs cannot diverge. Ollama preparation remains unchanged. Command-agent construction continues to reject mutable image references.

The container boundary remains defense in depth: a fresh internal network, read-only root, dropped capabilities, no new privileges, bounded resources, no Docker socket, no host home, and only the disposable request workspace mounted. Codex uses documented non-interactive `exec`, explicit read-only sandbox, never-approval, and ephemeral-session flags. Its private credential proxy permits only inference endpoints, rewrites the configured model and token limit, and never returns the long-lived credential. Codex may use its own internal agent protocol inside the disposable container, but callers receive only streamed final output through Myference's existing request schema.

## Packaging

Windows Docker Desktop normally runs Linux containers. Therefore the Windows release must place a Linux `myference-agent-proxy` binary beside `myference.exe`; mounting the existing Windows `.exe` into a Linux container cannot work. Release builds and installer tests will require and transactionally install this Linux sidecar.

A version-pinned Codex image definition is checked in under `packaging/agents/codex`. A GitHub workflow builds and publishes it to GHCR on release tags and reports its immutable digest. Provider configuration continues to store the full `@sha256:` reference; startup never trusts a mutable tag.

## Failure handling

- Missing Docker CLI: point to Docker Desktop installation.
- Missing Docker Desktop executable while the engine is stopped: explain that the engine could not be started.
- Startup timeout: identify the deadline and surface Docker's last health error without secrets.
- Windows-container mode: require switching Docker Desktop to Linux containers.
- Missing image: pull the exact digest; on failure include the backend/image and Docker error.
- Missing Linux sidecar: point to rerunning the one-command installer.
- Docker Desktop first-run terms, login, or interactive setup remain explicit external prerequisites and are reported rather than bypassed.

## Verification

Unit tests drive engine-ready, start-and-wait, timeout, non-Linux engine, image-present, image-pull, and pull-failure branches with literal expected commands. Existing command-runner tests additionally assert Codex's explicit sandbox/ephemeral flags and absence of host/Docker-socket exposure. Windows installer tests verify the Linux sidecar is installed and transactionally rolled back. CI builds all release artifacts and the Codex image. A real laptop smoke test starts Docker, inspects/pulls the published digest, adds the backend without printing its credential, and runs read-only doctor/headless status before the user starts headless mode.
