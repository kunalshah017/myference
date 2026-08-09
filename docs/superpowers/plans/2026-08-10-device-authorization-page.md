# Device authorization page implementation plan

1. Update route and dashboard tests to specify a dedicated Devices destination and separated API-key content.
2. Add `devices` to the dashboard view model, provider navigation, and icon map.
3. Route `/devices` to the new view and render a focused authorization page.
4. Refine device-approval copy and layout styles while preserving the existing approval API and wallet transaction flow.
5. Run focused web tests, lint, build, repository-wide verification, and browser checks at desktop and mobile widths.
