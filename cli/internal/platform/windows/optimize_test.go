package windows

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
)

func TestPlanProviderTuningUsesExplicitCommandsAndOllamaSettings(t *testing.T) {
	config := DefaultConfig()
	operations, err := PlanProviderTuning(config, TuningOptions{}, HostSnapshot{OnACPower: true})
	if err != nil {
		t.Fatal(err)
	}
	wantKinds := []OperationKind{OperationPowerPlan, OperationKeepAwake, OperationOllamaEnvironment, OperationProcessPriority}
	if got := operationKinds(operations); !reflect.DeepEqual(got, wantKinds) {
		t.Fatalf("kinds=%v want=%v", got, wantKinds)
	}
	if operations[0].Program != "powercfg.exe" || !reflect.DeepEqual(operations[0].Args, []string{"/setactive", "SCHEME_MIN"}) {
		t.Fatalf("power=%+v", operations[0])
	}
	wantEnvironment := map[string]string{
		"OLLAMA_MAX_LOADED_MODELS": "1", "OLLAMA_NUM_PARALLEL": "1",
		"OLLAMA_FLASH_ATTENTION": "1", "OLLAMA_KV_CACHE_TYPE": "q8_0",
	}
	if !reflect.DeepEqual(operations[2].Environment, wantEnvironment) || operations[3].Priority != "High" {
		t.Fatalf("Ollama operations=%+v %+v", operations[2], operations[3])
	}
}

func TestPlanProviderTuningEnforcesACPolicy(t *testing.T) {
	config := DefaultConfig()
	if _, err := PlanProviderTuning(config, TuningOptions{}, HostSnapshot{OnACPower: false}); err == nil {
		t.Fatal("battery tuning accepted")
	}
	if _, err := PlanProviderTuning(config, TuningOptions{AllowBattery: true}, HostSnapshot{OnACPower: false}); err != nil {
		t.Fatalf("battery override rejected: %v", err)
	}
}

func TestProviderTuningJournalsBeforeMutationAndRollsBackReverseOrder(t *testing.T) {
	store := NewJournalStore(t.TempDir())
	runner := &recordingTuningRunner{store: &store, snapshot: HostSnapshot{OnACPower: true, Journal: recoveryJournalFixture()}, failKind: OperationProcessPriority}
	err := StartProviderTuning(context.Background(), DefaultConfig(), TuningOptions{}, store, runner)
	if err == nil {
		t.Fatal("tuning succeeded despite priority failure")
	}
	wantApplied := []OperationKind{OperationPowerPlan, OperationKeepAwake, OperationOllamaEnvironment, OperationProcessPriority}
	if !reflect.DeepEqual(runner.applied, wantApplied) || !runner.journalSeenBeforeApply {
		t.Fatalf("applied=%v journalBefore=%v", runner.applied, runner.journalSeenBeforeApply)
	}
	if !reflect.DeepEqual(runner.restoredStages, []string{"ollama-environment", "keep-awake", "power-plan"}) {
		t.Fatalf("restored=%v", runner.restoredStages)
	}
	if _, loadErr := store.Load(); !errors.Is(loadErr, os.ErrNotExist) {
		t.Fatalf("journal after rollback=%v", loadErr)
	}
}

func TestProviderSessionCleanupRestoresExactlyOnce(t *testing.T) {
	store := NewJournalStore(t.TempDir())
	runner := &recordingTuningRunner{store: &store, snapshot: HostSnapshot{OnACPower: true, Journal: recoveryJournalFixture()}}
	cleanup, err := StartProviderSession(context.Background(), DefaultConfig(), TuningOptions{}, store, runner)
	if err != nil {
		t.Fatal(err)
	}
	if err := cleanup(); err != nil {
		t.Fatal(err)
	}
	if err := cleanup(); err != nil {
		t.Fatal(err)
	}
	if runner.restoreCalls != 1 {
		t.Fatalf("restore calls=%d", runner.restoreCalls)
	}
}

func operationKinds(operations []Operation) []OperationKind {
	result := make([]OperationKind, len(operations))
	for i := range operations {
		result[i] = operations[i].Kind
	}
	return result
}

type recordingTuningRunner struct {
	store                  *JournalStore
	snapshot               HostSnapshot
	failKind               OperationKind
	applied                []OperationKind
	restoredStages         []string
	journalSeenBeforeApply bool
	restoreCalls           int
}

func (runner *recordingTuningRunner) Snapshot(context.Context, Config) (HostSnapshot, error) {
	return runner.snapshot, nil
}
func (runner *recordingTuningRunner) Apply(_ context.Context, operation Operation) error {
	runner.applied = append(runner.applied, operation.Kind)
	if runner.store != nil {
		_, err := runner.store.Load()
		runner.journalSeenBeforeApply = runner.journalSeenBeforeApply || err == nil
	}
	if operation.Kind == runner.failKind {
		return errors.New("simulated apply failure")
	}
	return nil
}
func (runner *recordingTuningRunner) Restore(_ context.Context, journal RecoveryJournal) error {
	runner.restoreCalls++
	runner.restoredStages = journal.RollbackStages()
	return nil
}
