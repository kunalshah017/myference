package windows

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestJournalSaveLoadRecordsRequiredRecoveryState(t *testing.T) {
	store := NewJournalStore(t.TempDir())
	want := recoveryJournalFixture()

	if err := store.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Load() = %+v, want %+v", got, want)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(store.StateDir, recoveryJournalFileName))
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != 0o600 {
			t.Fatalf("journal permissions = %04o, want 0600", got)
		}
	}
}

func TestJournalSaveRefusesReplacementOfActiveRecovery(t *testing.T) {
	store := NewJournalStore(t.TempDir())
	first := recoveryJournalFixture()
	if err := store.Save(first); err != nil {
		t.Fatal(err)
	}

	second := recoveryJournalFixture()
	second.ActivePowerScheme = "22222222-2222-2222-2222-222222222222"
	if err := store.Save(second); !errors.Is(err, ErrActiveRecoveryJournal) {
		t.Fatalf("Save() error = %v, want ErrActiveRecoveryJournal", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, first) {
		t.Fatalf("active journal = %+v, want %+v", got, first)
	}
}

func TestJournalFailedUpdatePreservesLastGoodState(t *testing.T) {
	store := NewJournalStore(t.TempDir())
	want := recoveryJournalFixture()
	if err := store.Save(want); err != nil {
		t.Fatal(err)
	}
	store.replace = func(string, string) error { return errors.New("simulated rename failure") }

	if err := store.CompleteStage("power"); err == nil {
		t.Fatal("CompleteStage() succeeded despite a failed atomic replacement")
	}

	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("journal after failed update = %+v, want preserved %+v", got, want)
	}
}

func TestJournalCompleteStagePersistsAndIsIdempotent(t *testing.T) {
	store := NewJournalStore(t.TempDir())
	if err := store.Save(recoveryJournalFixture()); err != nil {
		t.Fatal(err)
	}

	if err := store.CompleteStage("power"); err != nil {
		t.Fatalf("CompleteStage() error = %v", err)
	}
	if err := store.CompleteStage("power"); err != nil {
		t.Fatalf("second CompleteStage() error = %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.AppliedStages, []string{"power"}) {
		t.Fatalf("AppliedStages = %v, want [power]", got.AppliedStages)
	}
}

func TestJournalRejectsUnknownSessionKindAndOwner(t *testing.T) {
	for name, mutate := range map[string]func(*RecoveryJournal){
		"kind":  func(journal *RecoveryJournal) { journal.SessionKind = "legacy-gateway" },
		"owner": func(journal *RecoveryJournal) { journal.OwnerPID = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			journal := recoveryJournalFixture()
			mutate(&journal)
			if err := NewJournalStore(t.TempDir()).Save(journal); err == nil {
				t.Fatal("invalid journal accepted")
			}
		})
	}
}

func TestJournalRollbackStagesRunsCompletedSetupInReverseOrder(t *testing.T) {
	store := NewJournalStore(t.TempDir())
	if err := store.Save(recoveryJournalFixture()); err != nil {
		t.Fatal(err)
	}
	for _, stage := range []string{"power", "shell", "tasks"} {
		if err := store.CompleteStage(stage); err != nil {
			t.Fatalf("CompleteStage(%q) error = %v", stage, err)
		}
	}

	journal, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := journal.RollbackStages(); !reflect.DeepEqual(got, []string{"tasks", "shell", "power"}) {
		t.Fatalf("RollbackStages() = %v, want [tasks shell power]", got)
	}
}

func TestJournalCompleteRecoveryKeepsStateAfterMandatoryRestoreFailure(t *testing.T) {
	store := NewJournalStore(t.TempDir())
	want := recoveryJournalFixture()
	if err := store.Save(want); err != nil {
		t.Fatal(err)
	}

	restoreErr := errors.New("shell restore failed")
	if err := store.CompleteRecovery(func(got RecoveryJournal) error {
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("restored journal = %+v, want %+v", got, want)
		}
		return restoreErr
	}); !errors.Is(err, restoreErr) {
		t.Fatalf("CompleteRecovery() error = %v, want %v", err, restoreErr)
	}

	if _, err := store.Load(); err != nil {
		t.Fatalf("Load() after mandatory restore failure = %v, want retained journal", err)
	}
}

func TestJournalCompleteRecoveryRemovesStateOnlyAfterSuccessAndIsRepeatable(t *testing.T) {
	store := NewJournalStore(t.TempDir())
	if err := store.Save(recoveryJournalFixture()); err != nil {
		t.Fatal(err)
	}

	restoreCalls := 0
	restore := func(RecoveryJournal) error {
		restoreCalls++
		return nil
	}
	if err := store.CompleteRecovery(restore); err != nil {
		t.Fatalf("CompleteRecovery() error = %v", err)
	}
	if err := store.CompleteRecovery(restore); err != nil {
		t.Fatalf("repeated CompleteRecovery() error = %v", err)
	}
	if restoreCalls != 1 {
		t.Fatalf("restore calls = %d, want 1", restoreCalls)
	}
	if _, err := store.Load(); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Load() error = %v, want os.ErrNotExist after recovery", err)
	}
}

func recoveryJournalFixture() RecoveryJournal {
	return RecoveryJournal{
		SessionKind:       "provider",
		OwnerPID:          1234,
		ActivePowerScheme: "11111111-1111-1111-1111-111111111111",
		Ollama: OllamaProcessSettings{
			Executable: `C:\\Users\\example\\AppData\\Local\\Programs\\Ollama\\ollama.exe`,
			Priority:   "Normal",
			Environment: map[string]*string{
				"OLLAMA_NUM_PARALLEL":  stringPointer("2"),
				"OLLAMA_KV_CACHE_TYPE": nil,
			},
		},
	}
}

func stringPointer(value string) *string { return &value }
