# Cross-device Offer Attachment Design

## Goal

Let a provider machine reuse a compatible offer already owned by the same wallet. A local backend keeps its machine-local name and credential identity while separately recording the public offer ID and finalized price version it advertises.

## Configuration

`config.Backend` gains an optional `offer_id`. `name` remains the local backend key used by commands, runtime maps, and credential-vault accounts. `offer_id` is the plaintext public identity hashed for routing and settlement. Existing configurations remain valid: an absent `offer_id` falls back to `name`, preserving every currently published backend.

New offers default to the backend name unless the provider explicitly attaches an existing offer. Publishing and version synchronization use the effective offer ID, while backend enable, disable, remove, and credential operations continue to use the local name.

## Account Offer Discovery and Attachment

Opening **Offers & Pricing** fetches `/api/provider/account` and renders two sections: **This machine** lists locally configured backends, while **Wallet offers** lists every account-owned offer, including offers that cannot run on this machine. Wallet rows state whether they are attached here, attachable to a configured backend, or unavailable until a matching provider is configured.

An existing offer is compatible only when all immutable runtime identity fields agree:

- exact model name;
- backend kind;
- sorted capabilities;
- metering mode.

If one compatible offer exists, the TUI recommends it. If several exist, the user chooses one. Attaching saves its `offer_id` and current finalized version locally and does not create a wallet transaction. An incompatible or missing offer continues through the existing new-offer pricing flow.

Selecting an attachable wallet offer attaches it directly when exactly one compatible local backend exists. If several local backends match, the user selects the desired machine row and then chooses the offer through the existing attachment chooser. Selecting an already attached wallet offer reports its local backend and never opens pricing. Selecting an unavailable wallet offer explains that a matching provider must be configured first.

On a machine row, Enter attaches a compatible existing offer, opens pricing only when no reusable offer exists and the backend is unpublished, and otherwise reports the existing attachment. Repricing an existing public offer is an explicit `e` action so inspecting or selecting an existing offer never asks for input and output prices.

## Command Surface

The equivalent recovery and automation command is:

```text
myference offer attach --backend <local-name> --offer <wallet-offer-id>
```

The command fetches the authenticated account projection, validates compatibility, persists the attachment atomically, and reports the attached version. It never accepts a caller-supplied version or bypasses compatibility checks.

`myference offer list` remains the account-wide wallet offer list. `myference backend list` remains the local machine backend list.

## Runtime and Synchronization

Capacity messages, offer hashes, backend maps, and receipts use the effective offer ID. Backend maps remain keyed by local backend name internally, with the capacity-to-backend association constructed explicitly so credentials and local lifecycle commands do not change.

The initial attachment supplies the existing finalized version. Once the machine advertises that offer, its routing state exists and the current machine-scoped version synchronization can follow later compatible price versions normally.

## Error Handling

- Account fetch failures remain visible and retryable; they do not erase local configuration.
- Unknown local backends and unknown wallet offers fail without mutation.
- Model, kind, capability, or metering mismatches explain why attachment is refused.
- Ambiguous matches require an explicit selection.
- Atomic config saving preserves the previous attachment on failure.
- Starting hosting remains blocked for enabled backends without an attached or newly published version.

## Testing

Tests cover configuration fallback, effective offer identity, compatibility checks, successful and rejected attachment, command behavior, complete wallet-offer rendering, selection behavior for attached, attachable, and unavailable offers, explicit repricing, capacity advertisement under a distinct offer ID, credential lookup under the unchanged backend name, and existing-config compatibility. Full Go, contract, web, installer, cross-platform build, and release verification remain required before publishing.

## Non-goals

- Deleting or deactivating immutable on-chain offer history.
- Renaming local backends or migrating credential-vault keys.
- Automatically attaching an ambiguous offer.
- Changing current per-machine stop/remove behavior.
- Adding a server endpoint; the account offer projection already contains the required data.
