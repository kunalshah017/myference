package windows

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
)

type OperationKind string

const (
	OperationPowerPlan         OperationKind = "power-plan"
	OperationKeepAwake         OperationKind = "keep-awake"
	OperationOllamaEnvironment OperationKind = "ollama-environment"
	OperationProcessPriority   OperationKind = "process-priority"
)

type TuningOptions struct {
	AllowBattery bool
}

type ProcessSnapshot struct {
	PID        int
	Name       string
	Executable string
}

type HostSnapshot struct {
	OnACPower bool
	Journal   RecoveryJournal
}

type Operation struct {
	Stage       string
	Kind        OperationKind
	Program     string
	Args        []string
	Environment map[string]string
	Priority    string
}

type OptimizationRunner interface {
	Snapshot(context.Context, Config) (HostSnapshot, error)
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

func StartProviderTuning(ctx context.Context, config Config, options TuningOptions, store JournalStore, runner OptimizationRunner) error {
	snapshot, err := runner.Snapshot(ctx, config)
	if err != nil {
		return fmt.Errorf("capture Windows recovery state: %w", err)
	}
	operations, err := PlanProviderTuning(config, options, snapshot)
	if err != nil {
		return err
	}
	if err := store.Save(snapshot.Journal); err != nil {
		if errors.Is(err, ErrActiveRecoveryJournal) {
			return fmt.Errorf("%w; stop the existing provider session before starting another", err)
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

func trimExecutableSuffix(name string) string {
	return strings.TrimSuffix(strings.TrimSpace(strings.ToLower(name)), ".exe")
}

func boolString(value bool) string {
	if value {
		return "1"
	}
	return "0"
}
