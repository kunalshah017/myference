//go:build windows

package windows

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

const (
	executionStateContinuous     = 0x80000000
	executionStateSystemRequired = 0x00000001
	processQueryLimitedInfo      = 0x1000
	processSetInformation        = 0x0200
	priorityNormal               = 0x20
	priorityAboveNormal          = 0x8000
	priorityHigh                 = 0x80
)

var powerSchemePattern = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

type NativeRunner struct {
	snapshot   HostSnapshot
	keepCancel context.CancelFunc
	keepDone   chan struct{}
	keepMutex  sync.Mutex
}

func NewNativeRunner() *NativeRunner { return &NativeRunner{} }

func (runner *NativeRunner) Snapshot(ctx context.Context, config Config) (HostSnapshot, error) {
	if err := config.Validate(); err != nil {
		return HostSnapshot{}, err
	}
	powerOutput, err := runWindowsCommand(ctx, "powercfg.exe", "/getactivescheme")
	if err != nil {
		return HostSnapshot{}, fmt.Errorf("read active power scheme: %w", err)
	}
	powerScheme, err := parseActivePowerScheme(string(powerOutput))
	if err != nil {
		return HostSnapshot{}, err
	}
	processOutput, err := runWindowsCommand(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "Get-CimInstance Win32_Process | Select-Object ProcessId,Name,ExecutablePath | ConvertTo-Json -Compress")
	if err != nil {
		return HostSnapshot{}, fmt.Errorf("list processes: %w", err)
	}
	processes, err := parseProcessSnapshots(bytes.NewReader(processOutput))
	if err != nil {
		return HostSnapshot{}, err
	}
	onAC, known := nativeACPower()
	if !known {
		return HostSnapshot{}, errors.New("could not determine AC/battery state")
	}
	journal := RecoveryJournal{SessionKind: "provider", OwnerPID: os.Getpid(), ActivePowerScheme: powerScheme}
	for _, process := range processes {
		name := trimExecutableSuffix(process.Name)
		if strings.EqualFold(name, "ollama") && journal.Ollama.Executable == "" {
			journal.Ollama.Executable = process.Executable
			journal.Ollama.Priority = processPriority(process.PID)
		}
	}
	journal.Ollama.Environment = originalOllamaEnvironment()
	snapshot := HostSnapshot{OnACPower: onAC, Journal: journal}
	runner.snapshot = snapshot
	return snapshot, nil
}

func (runner *NativeRunner) Apply(ctx context.Context, operation Operation) error {
	switch operation.Kind {
	case OperationPowerPlan:
		_, err := runWindowsCommand(ctx, operation.Program, operation.Args...)
		return err
	case OperationKeepAwake:
		return runner.startKeepAwake()
	case OperationOllamaEnvironment:
		if runner.snapshot.Journal.Ollama.Executable == "" {
			return nil
		}
		return runner.restartOllama(ctx, runner.snapshot.Journal.Ollama.Executable, environmentValues(operation.Environment), "")
	case OperationProcessPriority:
		return setRunningOllamaPriority(ctx, operation.Priority)
	default:
		return fmt.Errorf("unsupported Windows operation %q", operation.Kind)
	}
}

func (runner *NativeRunner) Restore(ctx context.Context, journal RecoveryJournal) error {
	stages := journal.RollbackStages()
	restoredOllama, restoredAwake, restoredPower := false, false, false
	for _, stage := range stages {
		switch {
		case (stage == string(OperationProcessPriority) || stage == string(OperationOllamaEnvironment)) && !restoredOllama:
			if journal.Ollama.Executable != "" {
				if err := runner.restartOllama(ctx, journal.Ollama.Executable, journal.Ollama.Environment, journal.Ollama.Priority); err != nil {
					return err
				}
			}
			restoredOllama = true
		case stage == string(OperationKeepAwake) && !restoredAwake:
			runner.stopKeepAwake()
			restoredAwake = true
		case stage == string(OperationPowerPlan) && !restoredPower:
			if _, err := runWindowsCommand(ctx, "powercfg.exe", "/setactive", journal.ActivePowerScheme); err != nil {
				return fmt.Errorf("restore power scheme: %w", err)
			}
			restoredPower = true
		}
	}
	return nil
}

func (runner *NativeRunner) startKeepAwake() error {
	runner.keepMutex.Lock()
	defer runner.keepMutex.Unlock()
	if runner.keepCancel != nil {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan error, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		proc := syscall.NewLazyDLL("kernel32.dll").NewProc("SetThreadExecutionState")
		result, _, callErr := proc.Call(executionStateContinuous | executionStateSystemRequired)
		if result == 0 {
			ready <- callErr
			return
		}
		ready <- nil
		<-ctx.Done()
		_, _, _ = proc.Call(executionStateContinuous)
	}()
	if err := <-ready; err != nil {
		cancel()
		<-done
		return err
	}
	runner.keepCancel, runner.keepDone = cancel, done
	return nil
}

func (runner *NativeRunner) stopKeepAwake() {
	runner.keepMutex.Lock()
	cancel, done := runner.keepCancel, runner.keepDone
	runner.keepCancel, runner.keepDone = nil, nil
	runner.keepMutex.Unlock()
	if cancel != nil {
		cancel()
		<-done
	}
}

func (runner *NativeRunner) restartOllama(ctx context.Context, executable string, environment map[string]*string, priority string) error {
	processes, err := runner.processes(ctx)
	if err != nil {
		return err
	}
	for _, process := range processes {
		if strings.EqualFold(trimExecutableSuffix(process.Name), "ollama") {
			_, _ = runWindowsCommand(ctx, "taskkill.exe", "/PID", strconv.Itoa(process.PID), "/T", "/F")
		}
	}
	command := exec.CommandContext(ctx, executable)
	command.Env = mergeEnvironment(os.Environ(), environment)
	if err := command.Start(); err != nil {
		return fmt.Errorf("restart Ollama: %w", err)
	}
	pid := command.Process.Pid
	_ = command.Process.Release()
	if priority != "" {
		return setProcessPriority(pid, priority)
	}
	return nil
}

func (runner *NativeRunner) processes(ctx context.Context) ([]ProcessSnapshot, error) {
	output, err := runWindowsCommand(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "Get-CimInstance Win32_Process | Select-Object ProcessId,Name,ExecutablePath | ConvertTo-Json -Compress")
	if err != nil {
		return nil, err
	}
	return parseProcessSnapshots(bytes.NewReader(output))
}

func parseActivePowerScheme(output string) (string, error) {
	match := powerSchemePattern.FindString(output)
	if match == "" {
		return "", errors.New("active power scheme GUID was not reported by powercfg")
	}
	return strings.ToLower(match), nil
}

func parseProcessSnapshots(input io.Reader) ([]ProcessSnapshot, error) {
	raw, err := io.ReadAll(io.LimitReader(input, 16<<20))
	if err != nil {
		return nil, err
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, nil
	}
	type item struct {
		ProcessID  int    `json:"ProcessId"`
		Name       string `json:"Name"`
		Executable string `json:"ExecutablePath"`
	}
	var values []item
	if raw[0] == '[' {
		if err := json.Unmarshal(raw, &values); err != nil {
			return nil, err
		}
	} else {
		var value item
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		values = []item{value}
	}
	result := make([]ProcessSnapshot, 0, len(values))
	for _, value := range values {
		if value.ProcessID > 0 && value.Name != "" {
			result = append(result, ProcessSnapshot{PID: value.ProcessID, Name: value.Name, Executable: value.Executable})
		}
	}
	return result, nil
}

func runWindowsCommand(ctx context.Context, program string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, program, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %v: %w: %s", program, args, err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func originalOllamaEnvironment() map[string]*string {
	result := make(map[string]*string)
	for _, name := range []string{"OLLAMA_MAX_LOADED_MODELS", "OLLAMA_NUM_PARALLEL", "OLLAMA_FLASH_ATTENTION", "OLLAMA_KV_CACHE_TYPE"} {
		if value, found := os.LookupEnv(name); found {
			copy := value
			result[name] = &copy
		} else {
			result[name] = nil
		}
	}
	return result
}

func environmentValues(values map[string]string) map[string]*string {
	result := make(map[string]*string, len(values))
	for name, value := range values {
		copy := value
		result[name] = &copy
	}
	return result
}

func mergeEnvironment(base []string, overrides map[string]*string) []string {
	result := make([]string, 0, len(base)+len(overrides))
	for _, item := range base {
		name, _, _ := strings.Cut(item, "=")
		if _, replaced := overrides[strings.ToUpper(name)]; !replaced {
			result = append(result, item)
		}
	}
	for name, value := range overrides {
		if value != nil {
			result = append(result, name+"="+*value)
		}
	}
	return result
}

func restartOptionalExecutables(paths []string) {
	seen := make(map[string]bool)
	for _, path := range paths {
		key := strings.ToLower(strings.TrimSpace(path))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			continue
		}
		command := exec.Command(path)
		if command.Start() == nil {
			_ = command.Process.Release()
		}
	}
}

func nativeACPower() (bool, bool) {
	type status struct {
		AC, BatteryFlag, BatteryPercent, System byte
		Life, FullLife                          uint32
	}
	var value status
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GetSystemPowerStatus")
	result, _, _ := proc.Call(uintptr(unsafe.Pointer(&value)))
	if result == 0 || value.AC == 255 {
		return false, false
	}
	return value.AC == 1, true
}

func processPriority(pid int) string {
	proc := syscall.NewLazyDLL("kernel32.dll")
	handle, _, _ := proc.NewProc("OpenProcess").Call(processQueryLimitedInfo, 0, uintptr(pid))
	if handle == 0 {
		return "Normal"
	}
	defer proc.NewProc("CloseHandle").Call(handle)
	value, _, _ := proc.NewProc("GetPriorityClass").Call(handle)
	switch value {
	case priorityHigh:
		return "High"
	case priorityAboveNormal:
		return "AboveNormal"
	default:
		return "Normal"
	}
}

func setRunningOllamaPriority(ctx context.Context, priority string) error {
	output, err := runWindowsCommand(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "Get-CimInstance Win32_Process | Select-Object ProcessId,Name,ExecutablePath | ConvertTo-Json -Compress")
	if err != nil {
		return err
	}
	processes, err := parseProcessSnapshots(bytes.NewReader(output))
	if err != nil {
		return err
	}
	for _, process := range processes {
		if strings.EqualFold(trimExecutableSuffix(process.Name), "ollama") {
			if err := setProcessPriority(process.PID, priority); err != nil {
				return err
			}
		}
	}
	return nil
}

func setProcessPriority(pid int, priority string) error {
	class := uintptr(priorityNormal)
	switch priority {
	case "AboveNormal":
		class = priorityAboveNormal
	case "High":
		class = priorityHigh
	case "Normal":
	default:
		return fmt.Errorf("unsupported process priority %q", priority)
	}
	proc := syscall.NewLazyDLL("kernel32.dll")
	handle, _, callErr := proc.NewProc("OpenProcess").Call(processSetInformation|processQueryLimitedInfo, 0, uintptr(pid))
	if handle == 0 {
		return callErr
	}
	defer proc.NewProc("CloseHandle").Call(handle)
	result, _, callErr := proc.NewProc("SetPriorityClass").Call(handle, class)
	if result == 0 {
		return callErr
	}
	return nil
}
