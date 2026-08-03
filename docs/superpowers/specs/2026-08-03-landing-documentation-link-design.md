# Landing Documentation Link Design

## Goal

Give visitors a direct path from the landing page’s “How it works” summary to the complete documentation.

## Design

Add one secondary action labeled “See full documentation” immediately below the four process cards. It links to `/docs`, reuses the existing `landing-secondary` button treatment, includes the existing arrow icon, and remains left-aligned with the section content. No new component, dependency, copy block, or route is needed.

## Verification

The landing-page test asserts that the uniquely named link targets `/docs`. The full web test, lint, and production-build commands must pass before deployment.
