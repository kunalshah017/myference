package windows

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
)

type OperationKind string

const (
	OperationPowerPlan            OperationKind = "power-plan"
	OperationKeepAwake            OperationKind = "keep-awake"
	OperationOllamaEnvironment    OperationKind = "ollama-environment"
	OperationProcessPriority      OperationKind = "process-priority"
	OperationStopService          OperationKind = "stop-service"
	OperationStopProcess          OperationKind = "stop-process"
	OperationInstallHeadlessTasks OperationKind = "install-headless-tasks"
	OperationHeadlessShell        OperationKind = "headless-shell"
	OperationACLidAction          OperationKind = "ac-lid-action"
	OperationDCLidAction          OperationKind = "dc-lid-action"
)

type TuningOptions struct {
	AllowBattery bool
}

type SnapshotOptions struct {
	Focus    bool
	Headless bool
}

type ProcessSnapshot struct {
	PID        int
	Name       string
	Executable string
}

type HostSnapshot struct {
	OnACPower       bool
	Processes       []ProcessSnapshot
	RunningServices []string
	Journal         RecoveryJournal
}

type Operation struct {
	Stage       string
	Kind        OperationKind
	Program     string
	Args        []string
	Name        string
	PID         int
	Executable  string
	Environment map[string]string
	Priority    string
	Value       string
}

func StartHeadlessProviderSession(ctx context.Context, config Config, options TuningOptions, store JournalStore, runner OptimizationRunner) (func() error, error) {
	active, err := store.Load()
	if err != nil {
		return nil, err
	}
	if active.SessionKind != "headless" {
		return nil, fmt.Errorf("headless provider requires a headless recovery journal")
	}
	snapshot, err := runner.Snapshot(ctx, config, SnapshotOptions{})
	if err != nil {
		return nil, err
	}
	operations, err := PlanProviderTuning(config, options, snapshot)
	if err != nil {
		return nil, err
	}
	if err := store.Update(func(journal *RecoveryJournal) error {
		journal.OwnerPID = os.Getpid()
		journal.Ollama = snapshot.Journal.Ollama
		return nil
	}); err != nil {
		return nil, err
	}
	for _, operation := range operations {
		if err := applyJournaledOperation(ctx, store, runner, operation); err != nil {
			_ = RestoreProviderTuning(ctx, store, runner)
			return nil, err
		}
	}
	// Provider tuning may activate a different power scheme. Apply the already
	// journaled headless lid overlay to that active scheme as the last step.
	for _, operation := range []Operation{
		{Stage: "headless:lid:ac", Kind: OperationACLidAction, Program: "powercfg.exe", Args: []string{"/setacvalueindex", "SCHEME_CURRENT", "SUB_BUTTONS", "LIDACTION", "0"}},
		{Stage: "headless:lid:dc", Kind: OperationDCLidAction, Program: "powercfg.exe", Args: []string{"/setdcvalueindex", "SCHEME_CURRENT", "SUB_BUTTONS", "LIDACTION", "0"}},
	} {
		if err := runner.Apply(ctx, operation); err != nil {
			_ = RestoreProviderTuning(ctx, store, runner)
			return nil, fmt.Errorf("apply %s: %w", operation.Stage, err)
		}
	}
	var once sync.Once
	var restoreErr error
	return func() error {
		once.Do(func() { restoreErr = RestoreProviderTuning(context.Background(), store, runner) })
		return restoreErr
	}, nil
}

type OptimizationRunner interface {
	Snapshot(context.Context, Config, SnapshotOptions) (HostSnapshot, error)
	Apply(context.Context, Operation) error
	Restore(context.Context, RecoveryJournal) error
}

func PlanProviderTuning(config Config, options TuningOptions, snapshot HostSnapshot) ([]Operation, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if err := config.ValidatePower(snapshot.OnACPower, options.AllowBattery); err != nil {
		return nil, err
	}
	operations := make([]Operation, 0, 4)
	if config.PerformancePowerPlan {
		operations = append(operations, Operation{Stage: string(OperationPowerPlan), Kind: OperationPowerPlan, Program: "powercfg.exe", Args: []string{"/setactive", "SCHEME_MIN"}})
	}
	operations = append(operations, Operation{Stage: string(OperationKeepAwake), Kind: OperationKeepAwake})
	operations = append(operations, Operation{
		Stage: string(OperationOllamaEnvironment), Kind: OperationOllamaEnvironment,
		Environment: map[string]string{
			"OLLAMA_MAX_LOADED_MODELS": strconv.Itoa(config.MaxLoadedModels),
			"OLLAMA_NUM_PARALLEL":      strconv.Itoa(config.NumParallel),
			"OLLAMA_FLASH_ATTENTION":   boolString(config.FlashAttention),
			"OLLAMA_KV_CACHE_TYPE":     config.KVCacheType,
		},
	})
	operations = append(operations, Operation{Stage: string(OperationProcessPriority), Kind: OperationProcessPriority, Priority: config.ProcessPriority})
	return operations, nil
}

func PlanFocus(config Config, snapshot HostSnapshot) ([]Operation, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	operations := make([]Operation, 0)
	for _, service := range snapshot.RunningServices {
		if containsFold(config.StopServices, service) {
			operations = append(operations, Operation{Stage: "focus:service:" + service, Kind: OperationStopService, Program: "sc.exe", Args: []string{"stop", service}, Name: service})
		}
	}
	for _, process := range snapshot.Processes {
		if containsFold(config.StopProcesses, trimExecutableSuffix(process.Name)) {
			operations = append(operations, Operation{Stage: "focus:process:" + strconv.Itoa(process.PID), Kind: OperationStopProcess, Program: "taskkill.exe", Args: []string{"/PID", strconv.Itoa(process.PID), "/T", "/F"}, Name: process.Name, PID: process.PID, Executable: process.Executable})
		}
	}
	return operations, nil
}

func StartProviderTuning(ctx context.Context, config Config, options TuningOptions, store JournalStore, runner OptimizationRunner) error {
	snapshot, err := runner.Snapshot(ctx, config, SnapshotOptions{})
	if err != nil {
		return fmt.Errorf("capture Windows recovery state: %w", err)
	}
	operations, err := PlanProviderTuning(config, options, snapshot)
	if err != nil {
		return err
	}
	if err := store.Save(snapshot.Journal); err != nil {
		if errors.Is(err, ErrActiveRecoveryJournal) {
			return fmt.Errorf("%w; run `myference windows restore` before starting another provider", err)
		}
		return err
	}
	for _, operation := range operations {
		if err := applyJournaledOperation(ctx, store, runner, operation); err != nil {
			rollbackErr := store.CompleteRecovery(func(journal RecoveryJournal) error { return runner.Restore(ctx, journal) })
			if rollbackErr != nil {
				return fmt.Errorf("%v; rollback failed: %w", err, rollbackErr)
			}
			return err
		}
	}
	return nil
}

func StartProviderSession(ctx context.Context, config Config, options TuningOptions, store JournalStore, runner OptimizationRunner) (func() error, error) {
	if err := StartProviderTuning(ctx, config, options, store, runner); err != nil {
		return nil, err
	}
	var once sync.Once
	var restoreErr error
	return func() error {
		once.Do(func() { restoreErr = RestoreProviderTuning(context.Background(), store, runner) })
		return restoreErr
	}, nil
}

func RestoreProviderTuning(ctx context.Context, store JournalStore, runner OptimizationRunner) error {
	return store.CompleteRecovery(func(journal RecoveryJournal) error { return runner.Restore(ctx, journal) })
}

func StartFocus(ctx context.Context, config Config, store JournalStore, runner OptimizationRunner) error {
	active, err := store.Load()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errors.New("focus requires an active provider; start `myference serve` first")
		}
		return err
	}
	if active.SessionKind != "provider" {
		return fmt.Errorf("focus requires a provider recovery journal, found %q", active.SessionKind)
	}
	if slices.ContainsFunc(active.AppliedStages, func(stage string) bool { return strings.HasPrefix(stage, "focus:") }) {
		return errors.New("focus is already active")
	}
	snapshot, err := runner.Snapshot(ctx, config, SnapshotOptions{Focus: true})
	if err != nil {
		return fmt.Errorf("capture Windows focus state: %w", err)
	}
	operations, err := PlanFocus(config, snapshot)
	if err != nil {
		return err
	}
	if err := store.Update(func(journal *RecoveryJournal) error {
		for _, operation := range operations {
			switch operation.Kind {
			case OperationStopProcess:
				if operation.Executable != "" {
					journal.StoppedProcesses = appendUniqueFold(journal.StoppedProcesses, operation.Executable)
				}
			case OperationStopService:
				journal.StoppedServices = appendUniqueFold(journal.StoppedServices, operation.Name)
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("record Windows focus recovery state: %w", err)
	}
	for _, operation := range operations {
		if err := applyJournaledOperation(ctx, store, runner, operation); err != nil {
			if rollbackErr := RestoreFocus(ctx, store, runner); rollbackErr != nil {
				return fmt.Errorf("%v; focus rollback failed: %w", err, rollbackErr)
			}
			return err
		}
	}
	return nil
}

func RestoreFocus(ctx context.Context, store JournalStore, runner OptimizationRunner) error {
	journal, err := store.Load()
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	focus := journal
	focus.AppliedStages = focus.AppliedStages[:0]
	for _, stage := range journal.AppliedStages {
		if strings.HasPrefix(stage, "focus:") {
			focus.AppliedStages = append(focus.AppliedStages, stage)
		}
	}
	if len(focus.AppliedStages) > 0 {
		if err := runner.Restore(ctx, focus); err != nil {
			return err
		}
	}
	return store.Update(func(active *RecoveryJournal) error {
		active.StoppedProcesses = nil
		active.StoppedServices = nil
		kept := active.AppliedStages[:0]
		for _, stage := range active.AppliedStages {
			if !strings.HasPrefix(stage, "focus:") {
				kept = append(kept, stage)
			}
		}
		active.AppliedStages = kept
		return nil
	})
}

func applyJournaledOperation(ctx context.Context, store JournalStore, runner OptimizationRunner, operation Operation) error {
	if err := runner.Apply(ctx, operation); err != nil {
		return fmt.Errorf("apply %s: %w", operation.Stage, err)
	}
	if err := store.CompleteStage(operation.Stage); err != nil {
		// The mutation happened even if the durable stage update failed. Keep the
		// stage in memory so the caller's immediate rollback includes it.
		journal, loadErr := store.Load()
		if loadErr == nil && !slices.Contains(journal.AppliedStages, operation.Stage) {
			journal.AppliedStages = append(journal.AppliedStages, operation.Stage)
			if restoreErr := runner.Restore(ctx, journal); restoreErr != nil {
				return fmt.Errorf("record %s: %v; immediate restore failed: %w", operation.Stage, err, restoreErr)
			}
		}
		return fmt.Errorf("record %s: %w", operation.Stage, err)
	}
	return nil
}

func ReverseOperations(operations []Operation) []Operation {
	reversed := make([]Operation, len(operations))
	for index, operation := range operations {
		reversed[len(operations)-1-index] = operation
	}
	return reversed
}

func appendUniqueFold(values []string, value string) []string {
	if value == "" || containsFold(values, value) {
		return values
	}
	return append(values, value)
}

func containsFold(values []string, target string) bool {
	return slices.ContainsFunc(values, func(value string) bool { return strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(target)) })
}

func trimExecutableSuffix(name string) string {
	return strings.TrimSuffix(strings.TrimSpace(strings.ToLower(name)), ".exe")
}

func boolString(value bool) string {
	if value {
		return "1"
	}
	return "0"
}
