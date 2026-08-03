# Task 2: Atomic recovery journal report

## Implementation

- Added `RecoveryJournal`, containing the active power scheme; AC/DC lid actions; shell policy; stopped process paths and services; Ollama executable, priority, and environment; installed task names; and completed mutation stages.
- Added `JournalStore` under `%LOCALAPPDATA%\Myference\state` via `DefaultJournalStore`, with temp-file write, `0600` request, `Sync`, and atomic create/update semantics.
- `Save` uses an exclusive hard-link commit so an active journal cannot be overwritten.
- Stage updates replace the prior journal atomically. Windows uses `MoveFileExW` with `MOVEFILE_REPLACE_EXISTING | MOVEFILE_WRITE_THROUGH`, because `os.Rename` does not replace an existing file on Windows.
- `CompleteRecovery` retains state if a mandatory restore callback fails and removes it only after that callback succeeds; repeated recovery is a no-op. `RollbackStages` exposes completed setup stages in reverse order.
- Task-name validation accepts only Myference-owned scheduled-task names, preventing recovery flows from removing unrelated tasks.

## Files

- `cli/internal/platform/windows/journal.go`
- `cli/internal/platform/windows/journal_rename_other.go`
- `cli/internal/platform/windows/journal_rename_windows.go`
- `cli/internal/platform/windows/journal_test.go`

## TDD evidence

Initial RED command:

```text
go test ./cli/internal/platform/windows -run Journal
FAIL ... journal_test.go: undefined: NewJournalStore
```

A subsequent RED test for the partial-setup order produced:

```text
go test ./cli/internal/platform/windows -run Journal
FAIL ... journal.RollbackStages undefined
```

GREEN command:

```text
go test ./cli/internal/platform/windows -run Journal
ok github.com/kunalshah017/myference/cli/internal/platform/windows
```

The tests use real temporary directories for save/load, active-journal refusal, stage persistence, recovery retention/removal, and repeatability. The failed-update test injects only an atomic-replace failure: a deterministic filesystem fault otherwise not portable across Windows/Unix rename semantics. Unix permission assertions are intentionally skipped on Windows because Windows `os.FileMode` bits are not inspectable in the Unix model, while production always requests `0600`.

## Final verification

```text
go test ./cli/internal/platform/windows
ok github.com/kunalshah017/myference/cli/internal/platform/windows

go test ./...
PASS: all repository Go packages; packages without tests reported `[no test files]`.
```

## Self-review

- Confirmed every specified recovery field round-trips through the durable journal.
- Confirmed a second start cannot replace active recovery state.
- Confirmed a failed atomic replacement leaves the previous valid journal readable.
- Confirmed repeated stage completion and repeated completed recovery are idempotent.
- Confirmed failed mandatory restoration retains the journal, while successful restoration removes it.
- Confirmed reverse-order stage access supports partial setup rollback.
- `git diff --check` was clean before commit.

## Concerns

- This task supplies the journal contract and reverse-order stage data. A later host-action task must invoke `CompleteStage` only after each mutation succeeds and implement its restore callback using `RollbackStages`.
