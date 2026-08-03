# Optional API Key Model Scopes

## Goal

New API keys should work with every currently available and future model by default. Users may optionally restrict a key to models selected from the live marketplace catalog.

## Design

- `models: []` is the canonical all-models scope. The server accepts it during key creation and authorizes any requested model for that key.
- A non-empty `models` array remains an exact allowlist. Existing restricted keys retain their current behavior.
- Endpoint and maximum-spend scopes remain required because they limit credential capabilities and financial exposure independently of model selection.
- The dashboard defaults to **All models**. Enabling **Restrict to selected models** reveals a multi-select populated from `GET /api/models`; users never type model IDs.
- If the marketplace catalog cannot load, unrestricted key creation remains available while restricted creation is disabled with a retryable error state.
- The first-time consumer onboarding creates an unrestricted key and treats unrestricted keys as valid for the selected playground model.
- Key lists label empty model scopes as **All models**.

## Verification

- Server integration tests cover unrestricted creation, arbitrary-model authorization, and preservation of non-empty allowlists.
- Web component tests cover the default empty model array, catalog-backed restriction, and all-model labeling.
- Onboarding tests cover unrestricted-key progress and unrestricted creation.
- Documentation describes unrestricted defaults and optional least-privilege model restrictions.
