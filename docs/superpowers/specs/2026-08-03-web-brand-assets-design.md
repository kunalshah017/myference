# Web Brand Assets Design

## Goal

Create a complete, clean web identity set for Myference that matches the existing product UI and produces correct link previews across social platforms.

## Visual system

The asset family uses the existing interface tokens: circuit ink `#171522`, relay violet `#6e5ae6`, paper `#fbfaff`, node mist `#f3f2f8`, and line `#d7d3e2`. The compact mark is a geometric `M/`: the `M` represents Myference and the violet slash echoes the relay/routing language already used in the wordmark. It avoids fine detail so it remains identifiable at 16px.

The Open Graph card uses the landing page’s restrained grid, large display typography, and one headline: “Unused compute, useful inference.” A compact visual uses locally stored OpenAI, Anthropic, Ollama, and Kimi marks around the Myference router, with Monad as the settlement layer. No remote assets are loaded by the published card.

## Assets

- `favicon.svg`: scalable primary favicon.
- `app-icon.svg`: opaque full-bleed source for platform-controlled icon masks.
- `favicon-16x16.png` and `favicon-32x32.png`: browser fallbacks.
- `apple-touch-icon.png`: 180×180 iOS icon.
- `icon-192.png` and `icon-512.png`: installable-site icons.
- `mstile-150x150.png`: Windows pinned-site tile.
- `favicon.ico`: legacy browser fallback containing a PNG favicon.
- `safari-pinned-tab.svg`: monochrome Safari pinned-tab mark.
- `og-image.svg`: editable social-card source.
- `og-image.png`: 1200×630 Open Graph/Twitter image.
- `brand/providers/*.png`: local provider and Monad marks used by the social card.
- `site.webmanifest`: app name, colors, and icon declarations.

## Metadata

`web/index.html` declares all favicon variants, Apple icon, manifest, canonical URL, theme color, Open Graph fields, and large-card Twitter fields. Social image URLs are absolute under `https://myference.xyz` so crawlers can resolve them.

## Verification

A dependency-free Node script verifies required files, raster dimensions, manifest fields, absolute social URLs, titles/descriptions, and that no placeholder domains remain. The production build must copy every public asset unchanged. Raster assets are visually inspected at full size and favicon scale before deployment.
