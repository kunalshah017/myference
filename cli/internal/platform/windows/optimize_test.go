package windows

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
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

func TestPlanFocusStopsOnlyConfiguredTargetsAndNeverExplorer(t *testing.T) {
	config := DefaultConfig()
	config.StopProcesses = []string{"Discord", "explorer"}
	config.StopServices = []string{"Spooler"}
	snapshot := HostSnapshot{OnACPower: true, Processes: []ProcessSnapshot{
		{PID: 10, Name: "Discord.exe", Executable: `C:\Discord\Discord.exe`},
		{PID: 11, Name: "notepad.exe", Executable: `C:\Windows\notepad.exe`},
		{PID: 12, Name: "explorer.exe", Executable: `C:\Windows\explorer.exe`},
	}, RunningServices: []string{"Spooler", "OtherService"}}
	operations, err := PlanFocus(config, snapshot)
	if err == nil || !strings.Contains(err.Error(), "Explorer") {
		t.Fatalf("configured Explorer error=%v", err)
	}
	config.StopProcesses = []string{"Discord"}
	operations, err = PlanFocus(config, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var stoppedProcesses, stoppedServices []string
	for _, operation := range operations {
		switch operation.Kind {
		case OperationStopProcess:
			stoppedProcesses = append(stoppedProcesses, operation.Name)
			if operation.Stage != "focus:process:10" || !reflect.DeepEqual(operation.Args, []string{"/PID", "10", "/T", "/F"}) {
				t.Fatalf("process=%+v", operation)
			}
		case OperationStopService:
			stoppedServices = append(stoppedServices, operation.Name)
			if operation.Stage != "focus:service:Spooler" || !reflect.DeepEqual(operation.Args, []string{"stop", "Spooler"}) {
				t.Fatalf("service=%+v", operation)
			}
		}
	}
	if !reflect.DeepEqual(stoppedProcesses, []string{"Discord.exe"}) || !reflect.DeepEqual(stoppedServices, []string{"Spooler"}) {
		t.Fatalf("processes=%v services=%v", stoppedProcesses, stoppedServices)
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

func TestFocusOverlayUsesActiveProviderJournalAndRestoresOnlyFocus(t *testing.T) {
	store := NewJournalStore(t.TempDir())
	journal := recoveryJournalFixture()
	journal.AppliedStages = []string{"power-plan", "keep-awake"}
	if err := store.Save(journal); err != nil {
		t.Fatal(err)
	}
	runner := &recordingTuningRunner{store: &store, snapshot: HostSnapshot{OnACPower: true, Journal: journal, Processes: []ProcessSnapshot{{PID: 10, Name: "Discord.exe", Executable: `C:\Discord\Discord.exe`}}, RunningServices: []string{"Spooler"}}}
	config := DefaultConfig()
	config.StopProcesses = []string{"Discord"}
	config.StopServices = []string{"Spooler"}
	if err := StartFocus(context.Background(), config, store, runner); err != nil {
		t.Fatal(err)
	}
	active, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(active.AppliedStages, []string{"power-plan", "keep-awake", "focus:service:Spooler", "focus:process:10"}) {
		t.Fatalf("active stages=%v", active.AppliedStages)
	}
	if err := RestoreFocus(context.Background(), store, runner); err != nil {
		t.Fatal(err)
	}
	active, err = store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(active.AppliedStages, []string{"power-plan", "keep-awake"}) || len(active.StoppedProcesses) != 0 || len(active.StoppedServices) != 0 {
		t.Fatalf("restored provider journal=%+v", active)
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

func (runner *recordingTuningRunner) Snapshot(context.Context, Config, SnapshotOptions) (HostSnapshot, error) {
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
