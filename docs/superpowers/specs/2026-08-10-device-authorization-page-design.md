# Device authorization page design

## Goal

Make browser-based provider-device authorization a focused dashboard task instead of hiding it below unrelated API-key content.

## Information architecture

- Add **Devices** to the **Provide inference** sidebar group.
- Rename **API access** to **API keys**. That destination contains endpoint guidance and API-key management only.
- Route `/devices` directly to the Devices destination and mark Devices active in the sidebar.
- Keep the existing provider-account and earnings destinations unchanged.

## Devices page

The page has one primary task and one compact supporting explanation:

1. Page heading: **Authorize a provider device** with a short explanation that the code comes from the Myference CLI.
2. The existing device-code inspection and wallet approval flow.
3. A short **How it works** panel explaining code entry, machine verification, and wallet authorization.

When disconnected, the page explains only the wallet requirement for approving a provider device. It does not mention API keys.

## Visual and interaction direction

- Preserve the existing Myference light circuit-board visual language, typography, colors, and Lucide icon family.
- Use a two-column task/support layout on wide screens and one column on narrow screens.
- Keep the authorization form readable without nested scrolling.
- Preserve visible labels, keyboard navigation, focus treatment, status/error semantics, and touch-friendly controls.

## Acceptance criteria

- `/devices` opens a dedicated Devices page and highlights Devices in the sidebar.
- API keys and device authorization are never rendered in the same destination.
- The Devices page has a single clear approval task and concise supporting instructions.
- The disconnected state and navigation labels accurately describe their destinations.
- Existing device inspection and approval behavior remains covered by its component tests.
- The dashboard remains usable at desktop and small-phone widths.
