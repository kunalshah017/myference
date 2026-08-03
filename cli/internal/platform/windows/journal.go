package windows

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const recoveryJournalFileName = "recovery.json"

// ErrActiveRecoveryJournal prevents a later provider, focus, or headless session from
// hiding the state that an earlier session still needs to restore.
var ErrActiveRecoveryJournal = errors.New("an active Myference recovery journal already exists")

// OllamaProcessSettings captures the process settings that a later restore must
// put back after an optimization session.
type OllamaProcessSettings struct {
	Executable  string             `json:"executable"`
	Priority    string             `json:"priority"`
	Environment map[string]*string `json:"environment"`
}

// RecoveryJournal is the complete pre-mutation snapshot for reversible Windows
// host controls. Zero lid-action values are meaningful ("do nothing").
type RecoveryJournal struct {
	SessionKind        string                `json:"sessionKind"`
	OwnerPID           int                   `json:"ownerPid"`
	ActivePowerScheme  string                `json:"activePowerScheme"`
	ACLidAction        int                   `json:"acLidAction"`
	DCLidAction        int                   `json:"dcLidAction"`
	HadShellPolicy     bool                  `json:"hadShellPolicy"`
	ShellPolicy        string                `json:"shellPolicy"`
	StoppedProcesses   []string              `json:"stoppedProcesses"`
	StoppedServices    []string              `json:"stoppedServices"`
	Ollama             OllamaProcessSettings `json:"ollama"`
	InstalledTaskNames []string              `json:"installedTaskNames"`
	AppliedStages      []string              `json:"appliedStages"`
}

// JournalStore persists one active recovery journal in StateDir. Its zero value
// is not configured; use NewJournalStore or DefaultJournalStore.
type JournalStore struct {
	StateDir string
	replace  func(source, destination string) error
}

// NewJournalStore creates a store rooted at stateDir. Tests and callers that
// already know their state directory should use this constructor.
func NewJournalStore(stateDir string) JournalStore {
	return JournalStore{StateDir: stateDir, replace: replaceJournalFile}
}

// DefaultJournalStore places durable recovery state under LOCALAPPDATA, never
// next to the executable or in a shared temporary directory.
func DefaultJournalStore() (JournalStore, error) {
	localAppData := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
	if localAppData == "" {
		return JournalStore{}, errors.New("LOCALAPPDATA is not set; cannot locate Myference recovery state")
	}
	return NewJournalStore(filepath.Join(localAppData, "Myference", "state")), nil
}

// Save creates the initial journal. It refuses to replace an existing one so a
// second start cannot discard the recovery data owned by an active session.
func (store JournalStore) Save(journal RecoveryJournal) error {
	if err := journal.validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(store.StateDir, 0o700); err != nil {
		return fmt.Errorf("create recovery state directory: %w", err)
	}

	path := store.path()
	if _, err := os.Lstat(path); err == nil {
		return ErrActiveRecoveryJournal
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect recovery journal: %w", err)
	}

	temporary, err := store.writeTemporary(journal)
	if err != nil {
		return err
	}
	defer os.Remove(temporary)
	if err := os.Link(temporary, path); err != nil {
		if _, statErr := os.Lstat(path); statErr == nil {
			return ErrActiveRecoveryJournal
		}
		return fmt.Errorf("commit recovery journal: %w", err)
	}
	return nil
}

// Load reads the active recovery journal. os.ErrNotExist means recovery is
// already complete (or no reversible mutation has been started).
func (store JournalStore) Load() (RecoveryJournal, error) {
	file, err := os.Open(store.path())
	if err != nil {
		return RecoveryJournal{}, err
	}
	defer file.Close()

	var journal RecoveryJournal
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&journal); err != nil {
		return RecoveryJournal{}, fmt.Errorf("decode recovery journal: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return RecoveryJournal{}, err
	}
	if err := journal.validate(); err != nil {
		return RecoveryJournal{}, fmt.Errorf("validate recovery journal: %w", err)
	}
	return journal, nil
}

// CompleteStage durably records an applied mutation after that mutation has
// succeeded. Recording a stage twice is safe for resumed or retried setup.
func (store JournalStore) CompleteStage(stage string) error {
	stage = strings.TrimSpace(stage)
	if stage == "" {
		return errors.New("recovery stage is required")
	}
	journal, err := store.Load()
	if err != nil {
		return err
	}
	for _, applied := range journal.AppliedStages {
		if applied == stage {
			return nil
		}
	}
	journal.AppliedStages = append(journal.AppliedStages, stage)
	return store.replaceExisting(journal)
}

func (store JournalStore) Update(mutate func(*RecoveryJournal) error) error {
	if mutate == nil {
		return errors.New("journal update callback is required")
	}
	journal, err := store.Load()
	if err != nil {
		return err
	}
	if err := mutate(&journal); err != nil {
		return err
	}
	return store.replaceExisting(journal)
}

func (store JournalStore) RemoveStages(prefixes ...string) error {
	if len(prefixes) == 0 {
		return errors.New("at least one recovery stage prefix is required")
	}
	return store.Update(func(journal *RecoveryJournal) error {
		kept := journal.AppliedStages[:0]
		for _, stage := range journal.AppliedStages {
			remove := false
			for _, prefix := range prefixes {
				if prefix != "" && strings.HasPrefix(stage, prefix) {
					remove = true
					break
				}
			}
			if !remove {
				kept = append(kept, stage)
			}
		}
		journal.AppliedStages = kept
		return nil
	})
}

// RollbackStages returns completed setup stages in the only safe restoration
// order: the newest applied mutation is restored first. The returned slice is
// independent of the journal so callers cannot alter durable recovery state.
func (journal RecoveryJournal) RollbackStages() []string {
	stages := make([]string, len(journal.AppliedStages))
	for index, stage := range journal.AppliedStages {
		stages[len(journal.AppliedStages)-1-index] = stage
	}
	return stages
}

// CompleteRecovery calls restore with the recorded state and removes the
// journal only when every mandatory restoration reports success. It is safe to
// call repeatedly after a completed restore.
func (store JournalStore) CompleteRecovery(restore func(RecoveryJournal) error) error {
	if restore == nil {
		return errors.New("recovery callback is required")
	}
	journal, err := store.Load()
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := restore(journal); err != nil {
		return err
	}
	if err := os.Remove(store.path()); err != nil {
		return fmt.Errorf("remove completed recovery journal: %w", err)
	}
	return nil
}

func (store JournalStore) replaceExisting(journal RecoveryJournal) error {
	if err := journal.validate(); err != nil {
		return err
	}
	temporary, err := store.writeTemporary(journal)
	if err != nil {
		return err
	}
	defer os.Remove(temporary)
	replace := store.replace
	if replace == nil {
		replace = replaceJournalFile
	}
	if err := replace(temporary, store.path()); err != nil {
		return fmt.Errorf("replace recovery journal: %w", err)
	}
	return nil
}

func (store JournalStore) writeTemporary(journal RecoveryJournal) (string, error) {
	payload, err := json.Marshal(journal)
	if err != nil {
		return "", fmt.Errorf("encode recovery journal: %w", err)
	}
	temporary, err := os.CreateTemp(store.StateDir, ".recovery-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create temporary recovery journal: %w", err)
	}
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		os.Remove(temporary.Name())
		return "", fmt.Errorf("set recovery journal permissions: %w", err)
	}
	if _, err := temporary.Write(payload); err != nil {
		temporary.Close()
		os.Remove(temporary.Name())
		return "", fmt.Errorf("write temporary recovery journal: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		os.Remove(temporary.Name())
		return "", fmt.Errorf("sync temporary recovery journal: %w", err)
	}
	if err := temporary.Close(); err != nil {
		os.Remove(temporary.Name())
		return "", fmt.Errorf("close temporary recovery journal: %w", err)
	}
	return temporary.Name(), nil
}

func (store JournalStore) path() string {
	return filepath.Join(store.StateDir, recoveryJournalFileName)
}

func (journal RecoveryJournal) validate() error {
	if journal.SessionKind != "provider" && journal.SessionKind != "headless" {
		return fmt.Errorf("recovery session kind %q is not supported", journal.SessionKind)
	}
	if journal.OwnerPID <= 0 {
		return errors.New("recovery session owner PID is required")
	}
	if strings.TrimSpace(journal.ActivePowerScheme) == "" {
		return errors.New("active power scheme is required")
	}
	for _, taskName := range journal.InstalledTaskNames {
		if !isMyferenceTask(taskName) {
			return fmt.Errorf("task %q is not Myference-owned", taskName)
		}
	}
	return nil
}

func isMyferenceTask(taskName string) bool {
	name := strings.ToLower(strings.TrimSpace(taskName))
	return name == "myference" || strings.HasPrefix(name, "myference ") || strings.HasPrefix(name, "myference-") || strings.HasPrefix(name, "myference\\") || strings.HasPrefix(name, "\\myference\\")
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("recovery journal contains trailing JSON data")
		}
		return fmt.Errorf("read recovery journal: %w", err)
	}
	return nil
}
