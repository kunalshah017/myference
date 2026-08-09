# Interactive CLI Hosting Design

## Goal

Make hosting approachable without requiring users to memorize commands. Running `myference` in an interactive terminal opens a full-screen interface that discovers usable local and API-backed AI providers, configures several models at once, completes browser-based account and wallet approval when required, and starts or stops hosting from the terminal. Existing subcommands remain available for automation and advanced workflows.

## Scope

The first release covers the complete hosting lifecycle:

- browser-assisted sign-in when the machine is not connected;
- automatic discovery of Ollama models and installed Codex and Claude CLIs;
- guided OpenAI and OpenAI-compatible API setup;
- multi-provider and multi-model selection;
- review, activation, foreground start and stop;
- live provider health and request activity;
- independent enable, disable, and retry actions.

Advanced Windows tuning, service installation, diagnostics, and account administration remain explicit commands. This release does not add a general provider plugin system or arbitrary CLI command templates.

## Entry Points and Compatibility

`myference` with no arguments opens the TUI only when standard input and output are interactive terminals. In a non-interactive environment it prints concise command usage and exits instead of waiting for input. Existing commands and flags retain their current behavior so scripts do not depend on terminal rendering.

The TUI and subcommands call the same hosting application service. The TUI contains presentation state only; it does not duplicate discovery, validation, credential, configuration, or runtime rules.

## Architecture

Add a focused hosting service under `cli/internal/host`. It coordinates provider discovery, validates selections, stages config changes, stores credentials through the existing credential abstraction, and controls the provider runtime. Existing command handlers migrate to these operations where doing so removes duplicated behavior without changing their public interface.

Add a TUI package under `cli/internal/tui`. It uses Bubble Tea and standard Bubble components for lists, text input, spinners, and viewports. Its model receives typed events and invokes the hosting service through a small dependency structure, allowing navigation and rendering tests without real subprocesses, network calls, a credential vault, or a raw terminal.

The main command selects between three paths:

- no arguments and an interactive terminal: launch the TUI;
- no arguments and a non-interactive terminal: print usage and return an error;
- arguments present: run the existing command path.

Hosting remains a foreground process. The live-status screen owns the running provider session. Quitting while a session is active asks whether to stop it; normal cancellation shuts it down cleanly. Background service management remains available through existing commands.

## TUI Navigation

The Home screen shows account state, provider discovery progress, configured model count, and host state. Its primary actions are Manage Providers, Review & Start, Live Status, and Quit.

Manage Providers presents a searchable checklist. Discovered entries and existing configuration appear in one list, with configured entries preselected. Users may select multiple Ollama models and installed CLI providers together. Add OpenAI and Add OpenAI-compatible open focused setup forms. The interface never silently selects a new provider or begins serving without review.

Review & Start shows each selected provider, model, evidence type, metering capability, and activation state. The user confirms the final set before configuration is committed or hosting starts. Pricing is entered in the short browser activation because those exact values become part of the wallet-signed publication transaction.

Live Status shows connection state and one row per offer with provider, model, health, active and completed request counts, and latest actionable error. The user may enable, disable, or retry an entry independently. A failed backend does not stop healthy backends.

Keyboard help is visible on every screen. Back and cancel operations never commit partial form state. Secret fields are masked and are cleared when their form is submitted or abandoned.

## Provider Discovery

Discovery runs providers concurrently with bounded per-provider timeouts and streams results into the TUI as they arrive. Failure in one detector does not delay or hide other results.

### Ollama

Probe the configured loopback Ollama endpoint and call its existing model-list operation. Display every installed model with its name, size, and digest. Each model is independently selectable. If the executable exists but the service cannot be reached, show an actionable “installed but not running” state. An empty catalog instructs the user to pull a model.

### Codex and Claude

Use executable lookup to detect `codex` and `claude` on `PATH`. Selecting one performs a bounded readiness check using its existing authenticated session. No login token is copied into Myference.

Neither CLI is assumed to expose a stable, complete model-list protocol. Myference therefore displays a small application-owned list of supported model presets plus Enter model manually. Native CLI providers remain model-only: marketplace prompts cannot access shell, filesystem, web, MCP, or other agent tools. Codex continues to use its existing hardened runner. Native Claude support must provide the same model-only isolation and fail closed on attempted tool use before it can be advertised.

The detector registry is an explicit Go slice rather than a plugin framework. A future supported CLI adds one detector and one backend implementation.

### OpenAI

The form requests an API key only and uses the fixed official OpenAI API base URL. After submission, Myference calls `/v1/models` with the key and shows a searchable model list. The user explicitly selects models to host. Failed authentication returns to the key field without printing or saving the secret.

### OpenAI-Compatible Providers

The form requests a provider name, HTTPS base URL, API key, and then model selection. Loopback HTTP remains allowed for local development. Myference calls the provider's `/v1/models` endpoint. If the endpoint is unsupported or returns no usable catalog, the form offers manual model entry rather than treating the whole provider as unavailable.

Model-list responses are bounded in size and time. Provider errors are sanitized so response bodies cannot echo submitted credentials into the terminal.

## Configuration and Credentials

Selections use stable identities derived from provider kind, endpoint identity where applicable, and model. Re-running setup matches existing entries, preserves their backend names and price versions, and updates them instead of creating duplicates. Name collisions that refer to a different identity require an explicit rename.

Configuration updates are staged and validated as a complete selection before replacing the config atomically. API keys are stored only in the existing OS credential vault. They never appear in the JSON config, process arguments, status snapshots, logs, or rendered TUI. If credential storage or config saving fails, newly created credential entries are rolled back where possible and the old config remains active.

Removing a selection from the review does not delete an existing backend; it disables it. Permanent removal remains an explicit backend command in this release.

## Authentication, Activation, and Pricing

If no machine account exists, the TUI starts the existing device authorization flow, opens the verification page, displays the fallback URL and code, and waits for completion. After browser sign-in succeeds, control returns to the same TUI step.

The TUI sends a short-lived activation draft containing offer identity, model, capabilities, and metering dimensions. Private wallet keys never enter the CLI. The browser is opened to enter integer-safe pricing and review and sign account, bond, signer-authorization, and immutable offer-publication actions. The TUI polls an authenticated activation-status endpoint and resumes automatically when the required actions are confirmed. Browser-open failure leaves a copyable URL and does not discard the draft.

Ollama and OpenAI-compatible backends use token and compute rates when their usage response supports them. Command CLI backends default to compute-only pricing unless reliable token usage is available. The review screen disables unsupported price dimensions rather than accepting rates that cannot be measured.

## Runtime and Errors

Starting from the TUI validates all enabled selections against their live provider before launching the shared provider daemon. Activation-pending entries remain configured but are not advertised. Healthy activated entries start even if another selected entry requires attention; the review screen clearly lists skipped entries before confirmation.

Runtime failures update only the affected offer. Each error provides one relevant action: Retry, Authenticate, Edit Settings, Start Dependency, or Disable. Secrets and raw upstream response bodies are excluded from error text. Terminal resize, end-of-input, cancellation, browser failure, network timeout, invalid URL, unavailable credential vault, empty catalog, and lost relay connection all have explicit states.

The host reconnects using the existing daemon behavior. Stopping cancels in-flight setup work, closes the provider session cleanly, and leaves saved selections available for the next run.

## Testing

Unit tests cover:

- interactive versus non-interactive entry selection;
- reducer navigation, selection, back, cancel, resize, and quit confirmation;
- concurrent discovery result ordering and independent failures;
- Ollama model enumeration and stable identity generation;
- Codex and Claude executable and readiness states;
- OpenAI fixed-URL model discovery;
- OpenAI-compatible URL validation, model discovery, and manual fallback;
- masked and cleared secret state;
- idempotent config updates and preserved price versions;
- credential rollback after save failures;
- multi-provider review and partial activation;
- runtime start, stop, retry, and isolated backend failure.

Integration tests use bounded fake Ollama and OpenAI-compatible HTTP servers, temporary config paths, fake credential stores, controlled executable lookup, and a fake hosting runtime. They verify that secrets never reach config, arguments, output, errors, or snapshots. Existing command tests remain green and demonstrate backward compatibility.

Repository verification runs Go formatting, focused package tests, `go test ./...`, race tests for the hosting and provider packages, `go vet ./...`, `go build ./...`, and `git diff --check`. Manual smoke checks cover keyboard navigation and browser return on macOS and Windows, with Linux terminal behavior checked where supported.
