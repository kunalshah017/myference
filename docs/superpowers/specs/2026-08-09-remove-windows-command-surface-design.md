# Remove Windows Command Surface Design

## Goal

Remove the Windows-only and legacy CLI commands that were retained during the legacy Windows CLI migration. Windows remains a supported hosting platform through the shared terminal UI and shared hosting and service commands.

## Public CLI Surface

The following commands are removed and must return the normal unknown-command error:

```text
myference windows ...
myference legacy-start
myference legacy-status
myference legacy-stop
myference stop
```

The shared provider interface remains:

```text
myference
myference login
myference host
myference backend ...
myference offer ...
myference collateral ...
myference capacity
myference status
myference serve
myference service ...
```

The usage message lists only this supported shared interface. The existing `publish` alias for `capacity` remains for compatibility because it is not part of the Windows CLI migration.

## Windows Hosting Boundary

Removing the commands does not remove Windows hosting. The Windows build retains the platform hooks called by shared commands:

- `serve` applies and restores reversible Windows provider tuning;
- enabled Ollama models are validated and preloaded before serving;
- required digest-pinned command-agent images are prepared through Docker;
- wallet approval URLs open through the Windows shell;
- `service install|start|stop|status|uninstall` manages the Windows provider service.

The automatic recovery journal remains an implementation detail of a normal provider session. No standalone Windows recovery, dashboard, focus, headless, doctor, model-listing, or model-test command remains public.

## Code Removal

The top-level router no longer dispatches `windows`, `stop`, or any `legacy-*` command. The Windows command parser and command-specific handlers are deleted. Command-only tests, acceptance-script calls, and documentation are updated or removed. Windows platform packages that implement automatic `serve` preparation, tuning, restoration, Docker readiness, and service lifecycle remain.

## Error Handling

Removed commands follow the same behavior as any unsupported top-level command. For example, `myference windows doctor` returns `unknown command "windows"`. No deprecation fallback or hidden alias is retained.

If automatic Windows provider tuning fails during `serve`, the existing rollback and recovery-journal behavior remains unchanged.

## Testing

- Router tests prove every removed top-level command is rejected.
- Existing command tests prove `offer`, `collateral`, `serve`, and `service` remain routed.
- Windows-targeted compilation proves the retained Windows hosting hooks and service implementation still build.
- Windows platform unit tests continue to cover automatic tuning, recovery, Docker preparation, and service lifecycle.
- Documentation and scripts contain no calls to the removed commands except historical design and plan records, which remain immutable project history.

## Non-Goals

- Removing Windows as a supported host platform.
- Removing Windows service support.
- Changing provider discovery, offers, collateral, pricing, or terminal UI behavior.
- Changing macOS hosting behavior.
- Adding a replacement namespace for the removed commands.
