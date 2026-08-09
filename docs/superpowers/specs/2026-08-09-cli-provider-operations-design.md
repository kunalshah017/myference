# CLI Provider Operations Design

## Goal

Make the Myference CLI the only hosting control plane while retaining the web client as a focused provider account console. A provider can discover and deploy models, create offers, set prices, manage collateral, make models public, and monitor hosting from the terminal. Wallet-owned transactions are prepared in the CLI and approved on a minimal browser page without importing the provider wallet key into the CLI.

## Product Boundary

The CLI owns:

- local and API-backed provider discovery;
- backend credentials and configuration;
- model deployment, enablement, disablement, start, and stop;
- creation of new offers from exact configured backends;
- initial and replacement price entry;
- collateral status and initiation of deposit and exit actions;
- activation of finalized offer versions;
- live hosting health and compatible-version synchronization.

The normal provider web page owns:

- collateral and earnings display;
- collateral deposit, exit request, and exit finalization;
- published offer history;
- repricing of an existing offer by publishing a new immutable version.

The web client does not discover models, display machine backends, deploy models, or create offers. An existing offer editor never permits changes to the offer identity, model, capabilities, or metering mode.

## Security Boundary

`depositBond`, `requestBondExit`, `finalizeBondExit`, and `publishOffer` are authorized by the provider wallet as the contract caller. The delegated machine signer remains limited to inference receipts. Myference will not store or import the provider wallet private key in the CLI.

Terminal-initiated wallet operations use a short-lived provider-action draft bound to the authenticated machine and account. The CLI supplies the exact action and values. The browser authenticates the same account, displays those immutable values, and requests approval from the connected wallet.

The server does not trust a version number or success flag posted by the browser. A provider action completes only after the indexed chain state proves that the signed-in account wallet performed the requested action and the finalized values match the draft. Cross-account access returns not found and cannot mutate or expire another account's draft.

Provider API keys, machine credentials, signer keys, and wallet secrets never enter action drafts, URLs, browser storage, logs, status files, or rendered terminal output.

## Account-Owned Offer Projection

The existing account operations response exposes machines and backend rows because the current web hosting form needs them. The new offer editor instead consumes an account-owned editable-offer projection.

An editable offer exists only when:

1. a plaintext `offer_id`, model, capabilities, and metering mode are present in provider routing state;
2. that routing row belongs to a machine whose `account_id` is the authenticated account;
3. the hashed identity matches an indexed on-chain offer owned by that account's wallet;
4. the returned version has the same model and capability hashes.

The projection returns the plaintext offer ID, locked identity fields, latest compatible finalized version, and integer MON rates. It does not return another account's machines or backends. Queries enforce account ownership at the database join rather than relying on client filtering.

The normal web provider response no longer needs machine or backend data for offer editing. Existing machine data may remain in the API temporarily for compatibility, but the provider page will not render or use it.

## Provider-Action Drafts

A bounded in-memory store extends the existing activation mechanism into typed provider actions:

- `publish_offer`: offer ID, model, sorted capabilities, metering mode, and three integer rates;
- `deposit_collateral`: positive integer wei amount;
- `request_collateral_exit`;
- `finalize_collateral_exit`.

Drafts expire after fifteen minutes, contain at most the selected offer batch, and are bound to machine and account IDs. New-offer publication may contain several offers, but the browser submits one contract transaction per offer because the contract has no batch publication function.

The browser records transaction hashes against the draft. The server validates their indexed effects through the repository and moves each action from `pending_wallet` to `pending_chain`, `confirmed`, or an actionable terminal failure. A server restart may expire an in-progress draft; the CLI can safely create a replacement because no action is considered complete without indexed evidence.

## CLI Application Services

Hosting presentation code remains separate from financial and configuration rules. Focused services provide:

- account-owned collateral and editable-offer reads using machine authentication;
- exact decimal-to-wei parsing and optional reference-price conversion;
- action-draft creation and polling;
- browser URL construction and opening;
- atomic price-version updates in the local configuration;
- compatible offer-version synchronization.

The TUI and command handlers call these same services. No command duplicates transaction construction or version-validation rules.

## Terminal UI

The home screen contains Providers, Offers & Pricing, Collateral, Live Status, and Quit.

### Providers

Provider discovery, model selection, API credential entry, deployment, enablement, and disablement remain CLI-only. The screen shows every configured backend and whether it has a public compatible offer.

### Offers & Pricing

The screen lists locally configured backends, never marketplace-wide or another account's models. Each row shows one of:

- Not published;
- Waiting for wallet approval;
- Waiting for chain confirmation;
- Public at version N;
- New compatible version synchronizing;
- Identity mismatch requiring a new offer.

Creating an offer starts from a configured backend so the offer ID, model, capabilities, evidence, and metering mode cannot drift from the runtime. Token-capable backends accept input-per-million, output-per-million, and compute-per-minute rates. Compute-only backends disable token fields. If the reference-price endpoint is fresh, the user may enter USD targets; review always displays the exact MON integers to be published. MON input remains available when the quote is absent.

Reviewing starts the provider-action draft, opens the minimal wallet page, polls chain confirmation, saves the confirmed version atomically, and only then advertises the offer.

### Collateral

The screen shows the wallet address, bonded amount, required minimum, exit state, and claimable amount. It offers Deposit, Request Exit, and Finalize Exit only when each action is currently valid. The amount and action are fixed before browser approval.

### Live Status

Live status continues to show relay health, requests, and backend failures. It also shows pending wallet actions, pending finality, and version-synchronization failures without disconnecting unrelated healthy backends.

## CLI Commands

Interactive functionality also remains available to scripts and recovery workflows:

```text
myference offer list
myference offer publish --backend <name> [pricing flags]
myference offer price --name <offer> [pricing flags]
myference collateral status
myference collateral deposit --amount <MON>
myference collateral exit
myference collateral finalize
```

Commands that mutate chain state open the same approval page and poll the same draft endpoint. Non-interactive environments may use a no-browser flag to print the approval URL, but they still require an external wallet approval. Existing backend and hosting commands remain compatible.

## Minimal Wallet Page

The activation route becomes a focused approval page rather than embedding the full provider console. It displays:

- the connected account and expected wallet;
- action type and exact values;
- offer identity and pricing for publication;
- per-transaction pending, confirmed, rejected, or reverted state;
- a final message telling the user that the terminal will continue automatically.

It contains no model discovery, backend dropdown, machine management, deployment controls, or editable transaction fields. A wallet address different from the account wallet is rejected before simulation.

## Web Provider Console

The normal `/host` page retains collateral, earnings, and offer history. Its offer editor lists only the account-owned editable-offer projection. Selecting an existing offer locks its identity fields and permits rate changes only. Publishing creates a new immutable version; it never edits historical chain data.

When no editable offer exists, the page says to run `myference` to configure a backend and publish the first offer. There is no manual offer ID or model input and no route from the web client to new-offer creation.

The Machines component and discovered-backend dropdown are removed from this page. Account-scoped APIs remain the enforcement boundary even if client state or URLs are manipulated.

## Compatible-Version Synchronization

The server exposes each machine's newest finalized offer version only when its offer, model, and capability hashes match the machine's reported runtime identity. The foreground TUI and headless provider poll this account-bound endpoint at a modest interval.

When a newer compatible version exists, the CLI atomically updates the backend's price version. The existing config watcher reloads the daemon without restarting other backends. A version with changed model or capability hashes is ignored and surfaced as an identity mismatch. Temporary API or indexer failures leave the currently confirmed version active.

## Error Handling

- Browser launch failure leaves a copyable approval URL.
- Wrong-account and wrong-wallet access cannot read or complete a draft.
- Wallet rejection keeps the action retryable until draft expiry.
- Reverted transactions show the transaction hash and revert state.
- Indexer delay remains `pending_chain` and never reports success early.
- Insufficient collateral blocks offer publication with the exact required amount.
- Collateral exit actions are unavailable when the contract state disallows them.
- Invalid decimal values, overflow, negative rates, and unsupported metering dimensions are rejected before a draft is created.
- Config-saving failure does not advertise the new version and remains retryable.
- Version-sync failure never disables a healthy currently confirmed offer.

## Testing

Store integration tests create two accounts with colliding model names and offer labels and prove that editable-offer and machine-sync projections return only the authenticated account's records.

Server API tests cover machine and browser authentication, draft validation, cross-account isolation, expiry, wrong-wallet transactions, unindexed transactions, exact on-chain effect verification, and multi-offer partial completion.

CLI unit tests cover decimal conversion, metering-aware pricing, action creation and polling, atomic version application, compatible-version synchronization, browser fallback URLs, commands, and TUI navigation. Provider tests prove that synchronization reloads one offer without interrupting another.

Web tests prove that the normal provider page contains no machine list, backend dropdown, model discovery, or new-offer controls; that only existing account-owned offers can be repriced; and that the minimal approval page renders immutable draft values and rejects a mismatched wallet.

Full verification runs Go formatting, `go test ./...`, focused race tests, `go vet ./...`, `go build ./...`, the complete web test suite, the production web build, and `git diff --check`.

## Non-Goals

- Importing provider wallet private keys into the CLI.
- Contract changes or batch-publication methods.
- Manual offer creation in the web client.
- Arbitrary model or capability changes through offer repricing.
- A general wallet-connectivity framework beyond the existing injected browser wallet.
- Removing account consumer functions from the web client.
