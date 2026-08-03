# One-command CLI Installers Design

## Goal

Provide one copy-paste command for Windows and one for macOS. Each installer detects the supported architecture, downloads the latest published CLI release, verifies its SHA-256 checksum, installs the CLI and agent proxy together, and leaves `myference` available on PATH.

## Public commands

```powershell
irm https://myference.xyz/install.ps1 | iex
```

```bash
curl -fsSL https://myference.xyz/install.sh | sh
```

Windows supports AMD64. macOS maps `x86_64` to AMD64 and `arm64` to Apple Silicon. Unsupported systems fail before downloading.

## Security and update behavior

The installers resolve GitHub's latest release, download the matching archive and `SHA256SUMS` over HTTPS, require an exact checksum match, then install. Temporary files are deleted on success and failure. Windows installs under `%LOCALAPPDATA%\Programs\Myference` and updates the user PATH. macOS installs into `/usr/local/bin`, requesting `sudo` only when the directory is not writable. Existing files are replaced only after a verified archive is extracted.

Environment overrides for release tag, download base, architecture, and install directory exist solely to make deterministic integration testing possible; they never disable checksum verification.
