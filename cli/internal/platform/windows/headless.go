package windows

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const HeadlessProviderTask = "Myference Headless Provider"

type HeadlessOptions struct {
	Executable string
	ConfigPath string
	Installer  string
}

type HeadlessRunner interface {
	Elevated(context.Context) bool
	SnapshotHeadless(context.Context) (RecoveryJournal, error)
	Apply(context.Context, Operation) error
	Restore(context.Context, RecoveryJournal) error
}

func PlanHeadless(options HeadlessOptions) ([]Operation, error) {
	for name, path := range map[string]string{"executable": options.Executable, "config": options.ConfigPath, "installer": options.Installer} {
		if path == "" || !filepath.IsAbs(path) {
			return nil, fmt.Errorf("headless %s path must be absolute", name)
		}
		if strings.Contains(path, `"`) {
			return nil, fmt.Errorf("headless %s path contains an unsupported quote", name)
		}
	}
	shell := `"` + options.Executable + `" windows dashboard --wait 30s --config "` + options.ConfigPath + `"`
	return []Operation{
		{Stage: "headless:tasks", Kind: OperationInstallHeadlessTasks, Program: "powershell.exe", Args: []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", options.Installer, "-Executable", options.Executable, "-Config", options.ConfigPath, "-Headless"}},
		{Stage: "headless:shell", Kind: OperationHeadlessShell, Program: "reg.exe", Args: []string{"add", `HKCU\Software\Microsoft\Windows NT\CurrentVersion\Winlogon`, "/v", "Shell", "/t", "REG_SZ", "/d", shell, "/f"}, Value: shell},
		{Stage: "headless:lid:ac", Kind: OperationACLidAction, Program: "powercfg.exe", Args: []string{"/setacvalueindex", "SCHEME_CURRENT", "SUB_BUTTONS", "LIDACTION", "0"}},
		{Stage: "headless:lid:dc", Kind: OperationDCLidAction, Program: "powercfg.exe", Args: []string{"/setdcvalueindex", "SCHEME_CURRENT", "SUB_BUTTONS", "LIDACTION", "0"}},
	}, nil
}

func InstallHeadless(ctx context.Context, options HeadlessOptions, store JournalStore, runner HeadlessRunner) error {
	operations, err := PlanHeadless(options)
	if err != nil {
		return err
	}
	if !runner.Elevated(ctx) {
		return errors.New("headless mode requires an Administrator terminal")
	}
	journal, err := runner.SnapshotHeadless(ctx)
	if err != nil {
		return fmt.Errorf("capture headless recovery state: %w", err)
	}
	journal.SessionKind = "headless"
	journal.OwnerPID = os.Getpid()
	journal.InstalledTaskNames = []string{HeadlessProviderTask}
	if err := store.Save(journal); err != nil {
		return err
	}
	for _, operation := range operations {
		if err := runner.Apply(ctx, operation); err != nil {
			rollbackErr := store.CompleteRecovery(func(journal RecoveryJournal) error { return runner.Restore(ctx, journal) })
			if rollbackErr != nil {
				return fmt.Errorf("apply %s: %v; rollback failed: %w", operation.Stage, err, rollbackErr)
			}
			return fmt.Errorf("apply %s: %w", operation.Stage, err)
		}
		if err := store.CompleteStage(operation.Stage); err != nil {
			journal, loadErr := store.Load()
			if loadErr == nil {
				journal.AppliedStages = append(journal.AppliedStages, operation.Stage)
				if restoreErr := runner.Restore(ctx, journal); restoreErr != nil {
					return fmt.Errorf("record %s: %v; rollback failed: %w", operation.Stage, err, restoreErr)
				}
				_ = store.CompleteRecovery(func(RecoveryJournal) error { return nil })
			}
			return fmt.Errorf("record %s: %w", operation.Stage, err)
		}
	}
	return nil
}

func HeadlessStatus(store JournalStore) (bool, error) {
	journal, err := store.Load()
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return journal.SessionKind == "headless", nil
}
