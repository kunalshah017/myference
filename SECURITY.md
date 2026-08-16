# Security Policy

## Reporting a vulnerability

Do not open a public issue for security bugs. Report them privately so we can
fix them before they are disclosed.

- **Preferred:** use GitHub's private vulnerability reporting at
  <https://github.com/kunalshah017/myference/security/advisories/new>.
- **Fallback:** email `security@myference.xyz`.

Include, when possible:

1. The affected component (broker server, provider CLI, smart contract, web
   client, or agent image) and version/commit.
2. A description of the vulnerability and its impact.
3. Steps to reproduce, a proof of concept, or a patch suggestion.
4. Whether the issue is on testnet or affects production.

We aim to acknowledge reports within 3 business days and provide a fix or
mitigation for confirmed issues within 30 days. We coordinate disclosure dates
with the reporter. We do not pay bounties at this time.

## Supported versions

Only the latest release is supported. The `main` branch is in active
development and may contain unreviewed changes.

| Component | Status |
|-----------|--------|
| Broker server (`server/`) | Supported |
| Provider CLI (`cli/`) | Supported |
| Smart contract (`contracts/src/MyferenceMarket.sol`) | Supported |
| Web client (`web/`) | Supported |
| Agent images (`packaging/agents/`) | Supported |

## Scope

In scope for the Myference platform:

- The broker API (`/relay`, `/v1/chat/completions`, `/v1/messages`, auth,
  marketplace, and realtime endpoints).
- Receipt construction, EIP-712 signing, settlement batching, and the chain
  indexer.
- The `MyferenceMarket` contract and its economic invariants (escrow, bonds,
  sessions, fees, slashing, claims).
- The provider CLI and its sandboxing of command-agent images, workspace
  isolation, credential handling, and installer/lifecycle scripts.
- Anything that would let an attacker spend another customer's funds, settle a
  forged receipt, exfiltrate a provider's wallet key, or escape the agent
  container.

## Out of scope

- Denial-of-service of the Monad testnet or public RPC endpoints.
- Social-engineering of individual users.
- Vulnerabilities in third-party dependencies unless they are exploited in a
  way specific to Myference (report those upstream instead).
- Brute-force of credentials with no rate-limit-independent impact.

## Security model

- Provider wallets never touch the broker; each machine authorizes a
  headless EIP-712 signer on-chain and stores the key in the OS credential
  vault.
- The settlement signer is a single high-value key held by the operator. Treat
  any exposure of `MYFERENCE_SETTLEMENT_PRIVATE_KEY` or the wallet key that
  deploys/owns the contract as a critical incident.
- API keys are stored only as SHA-256 digests; tokens are never logged.
- Access logs contain request IDs, account/key identifiers, model, provider,
  and measured usage only — never prompts, model output, or secret material.

## Disclosure of a settlement-key compromise

If the settlement signer or contract owner key is compromised, contact us
immediately through the channel above. The contract owner can `pause()` the
market; do not attempt to broadcast recovery transactions yourself.
