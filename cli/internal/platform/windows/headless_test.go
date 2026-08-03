package windows

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestPlanHeadlessUsesAbsoluteProviderTaskAndDashboardShell(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "myference.exe")
	configPath := filepath.Join(root, "myference.json")
	installer := filepath.Join(root, "install-windows.ps1")
	operations, err := PlanHeadless(HeadlessOptions{Executable: executable, ConfigPath: configPath, Installer: installer})
	if err != nil {
		t.Fatal(err)
	}
	wantKinds := []OperationKind{OperationInstallHeadlessTasks, OperationHeadlessShell, OperationACLidAction, OperationDCLidAction}
	if got := operationKinds(operations); !reflect.DeepEqual(got, wantKinds) {
		t.Fatalf("kinds=%v want=%v", got, wantKinds)
	}
	if operations[0].Program != "powershell.exe" || !strings.Contains(strings.Join(operations[0].Args, " "), executable) || !strings.Contains(strings.Join(operations[0].Args, " "), configPath) {
		t.Fatalf("task operation=%+v", operations[0])
	}
	if !strings.Contains(operations[1].Value, `windows dashboard`) || !strings.Contains(operations[1].Value, `--config`) || !strings.Contains(operations[1].Value, executable) {
		t.Fatalf("shell operation=%+v", operations[1])
	}
	for _, operation := range operations {
		if strings.Contains(strings.ToLower(operation.Value), "explorer") {
			t.Fatalf("headless setup starts Explorer: %+v", operation)
		}
	}
	if _, err := PlanHeadless(HeadlessOptions{Executable: "myference.exe", ConfigPath: configPath, Installer: installer}); err == nil {
		t.Fatal("relative executable accepted")
	}
}

func TestInstallHeadlessPreflightsAndJournalsBeforeMutation(t *testing.T) {
	store := NewJournalStore(t.TempDir())
	journal := recoveryJournalFixture()
	journal.SessionKind = "headless"
	runner := &recordingHeadlessRunner{store: &store, journal: journal, elevated: true}
	root := t.TempDir()
	options := HeadlessOptions{Executable: filepath.Join(root, "myference.exe"), ConfigPath: filepath.Join(root, "myference.json"), Installer: filepath.Join(root, "install-windows.ps1")}
	if err := InstallHeadless(context.Background(), options, store, runner); err != nil {
		t.Fatal(err)
	}
	if !runner.journalSeenBeforeApply || len(runner.applied) != 4 {
		t.Fatalf("journalBefore=%v applied=%v", runner.journalSeenBeforeApply, runner.applied)
	}
	active, err := store.Load()
	if err != nil || active.SessionKind != "headless" || !reflect.DeepEqual(active.InstalledTaskNames, []string{"Myference Headless Provider"}) {
		t.Fatalf("journal=%+v err=%v", active, err)
	}
	if err := InstallHeadless(context.Background(), options, store, runner); !errors.Is(err, ErrActiveRecoveryJournal) {
		t.Fatalf("second install=%v", err)
	}
}

func TestInstallHeadlessRefusesWithoutElevationBeforeJournal(t *testing.T) {
	store := NewJournalStore(t.TempDir())
	runner := &recordingHeadlessRunner{elevated: false}
	root := t.TempDir()
	err := InstallHeadless(context.Background(), HeadlessOptions{Executable: filepath.Join(root, "myference.exe"), ConfigPath: filepath.Join(root, "myference.json"), Installer: filepath.Join(root, "install.ps1")}, store, runner)
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "administrator") {
		t.Fatalf("error=%v", err)
	}
	if _, loadErr := store.Load(); !errors.Is(loadErr, os.ErrNotExist) {
		t.Fatalf("journal created before elevation check: %v", loadErr)
	}
}

type recordingHeadlessRunner struct {
	store                  *JournalStore
	journal                RecoveryJournal
	elevated               bool
	journalSeenBeforeApply bool
	applied                []OperationKind
}

func (runner *recordingHeadlessRunner) Elevated(context.Context) bool { return runner.elevated }
func (runner *recordingHeadlessRunner) SnapshotHeadless(context.Context) (RecoveryJournal, error) {
	return runner.journal, nil
}
func (runner *recordingHeadlessRunner) Apply(_ context.Context, operation Operation) error {
	runner.applied = append(runner.applied, operation.Kind)
	if runner.store != nil {
		_, err := runner.store.Load()
		runner.journalSeenBeforeApply = runner.journalSeenBeforeApply || err == nil
	}
	return nil
}
func (runner *recordingHeadlessRunner) Restore(context.Context, RecoveryJournal) error { return nil }
