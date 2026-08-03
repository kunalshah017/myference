# Myference Website Documentation Design

## Goal

Add a public, product-native documentation experience that takes a new customer or provider from zero to a real request or a routable machine. Every command, endpoint, limit, and behavior must correspond to the repository's implemented behavior.

## Chosen approach

Build one curated `/docs` page inside the existing React application. A sticky section rail and hash links make the page navigable without adding a documentation framework, search service, or second deployment. The content can later be split into individual routes without changing its information architecture.

Alternatives considered:

- A multi-page docs router scales further but adds routing and content-management complexity before the current guide needs it.
- An external documentation platform ships quickly but would fragment the product theme and deployment.

## Audience journeys

### Use inference

1. Connect a Monad Testnet wallet.
2. Deposit native MON into customer escrow.
3. Open a time- and amount-bounded spending session.
4. Select a live model and inspect its immutable MON pricing.
5. Create a model-, endpoint-, and spend-scoped API key.
6. Test through the playground or a streaming OpenAI/Anthropic-compatible request.
7. Inspect usage, settlement state, and remaining funds; close the session or withdraw when finished.

### Host inference

1. Download and checksum-verify the published release artifact for Windows or the correct macOS architecture.
2. Install Ollama and pull a model for the shortest local-model path.
3. Run `myference host`; browser login creates a machine signer and stores secrets in the OS credential vault.
4. Approve the machine wallet transaction in the browser.
5. Deposit provider collateral and publish an immutable offer from `/host`.
6. Select the confirmed offer version in the CLI, if it differs from the configured version.
7. Serve in the foreground or install the Windows Scheduled Task/macOS LaunchAgent.
8. Monitor requests, earnings, collateral, and claimable MON in the provider dashboard.

Advanced backend sections cover OpenAI-compatible cloud providers and isolated Codex, Claude, and Kimi command agents without presenting unimplemented automation as available.

## Page structure

- Product header with links to the app, source repository, and current docs page.
- Compact docs hero with the live API base URL and network status context.
- Sticky left navigation grouped into Start, Use, Host, Understand, and Reference.
- Main reading column with numbered steps, callouts, compact tables, and copyable code examples.
- Sticky right "On this page" navigation on wide screens; both rails collapse cleanly on smaller screens.
- Footer actions to open the app or provider workspace.

## Content rules

- Use the deployed API `https://api.myference.xyz` and web client `https://myference.xyz`.
- Link release downloads to the published GitHub release assets, not imaginary package-manager commands.
- Explain that inference endpoints are streaming-only and require an integer-wei `X-Myference-Max-Spend` reservation header.
- Show JavaScript and Python clients through standard streaming HTTP examples; do not claim drop-in SDK support where a required custom header cannot be configured.
- Cover both `/v1/chat/completions` and `/v1/messages`, including required authorization headers.
- Explain native MON escrow, bounded sessions, immutable rates, informational USD conversion, actual usage receipts, provider/platform splits, delayed exits, and claimable balances.
- Be explicit that model evidence verifies an Ollama digest, upstream model identifier, or pinned runtime image—not the semantic quality of a model response.
- Explain that command agents are compute-only unless trustworthy token usage exists, and that workspaces are disposable and bounded.
- Include actionable troubleshooting for 401 responses, no routable model, stale providers, failed machine approval, offer-version drift, Ollama connectivity, Docker requirements, and reference-price unavailability.
- Never include real private keys, API keys, wallet secrets, or user prompt contents.

## Interaction and accessibility

- Copy buttons use the browser Clipboard API and provide visible status feedback.
- Semantic `header`, `nav`, `main`, `section`, `ol`, `table`, and `footer` landmarks are used.
- Hash targets include scroll margin for the sticky header.
- Keyboard focus follows the existing focus tokens, code blocks scroll horizontally, and all controls have explicit labels.
- The layout respects the existing typography, violet/blue/mint palette, square borders, and grid-backed paper surface.

## Verification

- A route-level test must prove `/docs` renders instead of the landing page.
- Content assertions must cover both user journeys, both operating systems, both API dialects, the real deployed base URL, settlement/security guidance, and troubleshooting.
- Navigation tests must prove landing and dashboard documentation links point to `/docs`.
- Run the full web test suite, lint, and production build before commit and push.
