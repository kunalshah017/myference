//go:build windows

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/kunalshah017/myference/cli/internal/config"
	platform "github.com/kunalshah017/myference/cli/internal/platform/windows"
)

func runPlatformCommand(command string, args []string, output io.Writer) error {
	if command == "windows" {
		windowsCommand, err := platform.ParseCommand(args)
		if err != nil {
			return err
		}
		switch windowsCommand.Action {
		case "doctor":
			return runWindowsDoctor(context.Background(), windowsCommand.Args, output)
		case "models":
			return runWindowsModels(context.Background(), windowsCommand.Args, output)
		case "test":
			return runWindowsTest(context.Background(), windowsCommand.Args, output)
		case "focus":
			return runWindowsFocus(context.Background(), windowsCommand.Args, output)
		case "restore":
			return runWindowsRestore(context.Background(), windowsCommand.Args, output)
		default:
			return fmt.Errorf("windows %s is not implemented", windowsCommand.Action)
		}
	}

	if command == "service" {
		if len(args) == 0 {
			return errors.New("usage: myference service <install|start|stop|status|uninstall>")
		}
		flags := flag.NewFlagSet("service "+args[0], flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		path := flags.String("config", defaultConfigPath(), "configuration path")
		if err := flags.Parse(args[1:]); err != nil {
			return err
		}
		service, err := platform.DiscoverService(*path)
		if err != nil {
			return err
		}
		ctx := context.Background()
		switch args[0] {
		case "install":
			return service.Install(ctx)
		case "start":
			return service.Start(ctx)
		case "stop":
			return service.Stop(ctx)
		case "status":
			return service.Status(ctx)
		case "uninstall":
			return service.Uninstall(ctx)
		default:
			return errors.New("unknown service action")
		}
	}
	lifecycle, err := platform.Discover()
	if err != nil {
		return err
	}
	switch command {
	case "legacy-start":
		return lifecycle.Start(context.Background())
	case "legacy-status":
		return lifecycle.Status(context.Background())
	default:
		return lifecycle.Stop(context.Background())
	}
}

func startPlatformProviderSession(ctx context.Context, _ config.Config, _ io.Writer) (func() error, error) {
	store, err := platform.DefaultJournalStore()
	if err != nil {
		return nil, err
	}
	return platform.StartProviderSession(ctx, platform.DefaultConfig(), platform.TuningOptions{}, store, platform.NewNativeRunner())
}

func runWindowsFocus(ctx context.Context, args []string, output io.Writer) error {
	if len(args) != 1 {
		return errors.New("usage: myference windows focus <start|status|restore>")
	}
	store, err := platform.DefaultJournalStore()
	if err != nil {
		return err
	}
	runner := platform.NewNativeRunner()
	switch args[0] {
	case "start":
		if err := platform.StartFocus(ctx, platform.DefaultConfig(), store, runner); err != nil {
			return err
		}
		_, err = fmt.Fprintln(output, "Windows focus mode active")
		return err
	case "status":
		journal, loadErr := store.Load()
		if errors.Is(loadErr, os.ErrNotExist) {
			_, err = fmt.Fprintln(output, "Windows focus mode inactive")
			return err
		}
		if loadErr != nil {
			return loadErr
		}
		active := slices.ContainsFunc(journal.AppliedStages, func(stage string) bool { return strings.HasPrefix(stage, "focus:") })
		state := "inactive"
		if active {
			state = "active"
		}
		_, err = fmt.Fprintf(output, "Windows focus mode %s\n", state)
		return err
	case "restore":
		if err := platform.RestoreFocus(ctx, store, runner); err != nil {
			return err
		}
		_, err = fmt.Fprintln(output, "Windows focus mode restored")
		return err
	default:
		return fmt.Errorf("unknown Windows focus action %q", args[0])
	}
}

func runWindowsRestore(ctx context.Context, args []string, output io.Writer) error {
	if len(args) != 0 {
		return errors.New("usage: myference windows restore")
	}
	store, err := platform.DefaultJournalStore()
	if err != nil {
		return err
	}
	if err := platform.RestoreProviderTuning(ctx, store, platform.NewNativeRunner()); err != nil {
		return err
	}
	_, err = fmt.Fprintln(output, "Windows provider host state restored")
	return err
}

func runWindowsModels(ctx context.Context, args []string, output io.Writer) error {
	flags := flag.NewFlagSet("windows models", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	endpoint := flags.String("ollama-url", "http://127.0.0.1:11434", "loopback Ollama URL")
	if err := flags.Parse(args); err != nil {
		return err
	}
	client, err := platform.NewOllamaHostClient(*endpoint, nil)
	if err != nil {
		return err
	}
	models, err := client.InstalledModels(ctx)
	if err != nil {
		return fmt.Errorf("list Ollama models: %w", err)
	}
	if len(models) == 0 {
		return errors.New("Ollama has no installed models; install one with `ollama pull <model>`")
	}
	for _, model := range models {
		if _, err := fmt.Fprintln(output, model); err != nil {
			return err
		}
	}
	return nil
}

func runWindowsTest(ctx context.Context, args []string, output io.Writer) error {
	flags := flag.NewFlagSet("windows test", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	endpoint := flags.String("ollama-url", "http://127.0.0.1:11434", "loopback Ollama URL")
	requested := flags.String("model", "", "installed model name")
	if err := flags.Parse(args); err != nil {
		return err
	}
	client, err := platform.NewOllamaHostClient(*endpoint, nil)
	if err != nil {
		return err
	}
	models, err := client.InstalledModels(ctx)
	if err != nil {
		return fmt.Errorf("list Ollama models: %w", err)
	}
	model, err := platform.SelectInstalledModel(models, *requested)
	if err != nil {
		return err
	}
	response, err := client.GenerateTest(ctx, model)
	if err != nil {
		return fmt.Errorf("test Ollama model %q: %w", model, err)
	}
	_, err = fmt.Fprintf(output, "%s: %s\\n", model, response)
	return err
}

func runWindowsDoctor(ctx context.Context, args []string, output io.Writer) error {
	flags := flag.NewFlagSet("windows doctor", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	configPath := flags.String("config", defaultConfigPath(), "provider configuration path")
	endpoint := flags.String("ollama-url", "http://127.0.0.1:11434", "loopback Ollama URL")
	if err := flags.Parse(args); err != nil {
		return err
	}
	state := collectWindowsDoctorState(ctx, *configPath, *endpoint)
	return platform.WriteDoctor(output, platform.DoctorFindings(state))
}

func collectWindowsDoctorState(ctx context.Context, configPath, endpoint string) platform.DoctorState {
	state := platform.DoctorState{WindowsVersion: windowsVersion()}
	state.OllamaPath, _ = exec.LookPath("ollama.exe")
	if state.OllamaPath == "" {
		state.OllamaPath, _ = exec.LookPath("ollama")
	}
	_, credentialErr := exec.LookPath("cmdkey.exe")
	state.CredentialStoreAvailable = credentialErr == nil
	state.OnACPower, state.OnACPowerKnown = currentACPower()
	state.ServiceInstalled = exec.CommandContext(ctx, "schtasks.exe", "/Query", "/TN", "Myference Provider").Run() == nil
	cfg, err := config.Load(configPath)
	state.ConfigReadable = err == nil
	if err == nil {
		for _, backend := range cfg.Backends {
			if !backend.Enabled {
				continue
			}
			if backend.Kind == "ollama" && state.ConfiguredModel == "" {
				state.ConfiguredModel = backend.Model
			}
			if slices.Contains([]string{"codex", "claude", "kimi"}, backend.Kind) {
				state.DockerRequired = true
			}
		}
	}
	if state.DockerRequired {
		state.DockerPath, _ = exec.LookPath("docker.exe")
	}
	client, clientErr := platform.NewOllamaHostClient(endpoint, &http.Client{Timeout: 5 * time.Second})
	if clientErr == nil {
		state.InstalledModels, _ = client.InstalledModels(ctx)
	}
	return state
}

func preparePlatformBackends(ctx context.Context, cfg config.Config) error {
	return prepareWindowsBackends(ctx, cfg, platform.DefaultConfig(), nil)
}

func prepareWindowsBackends(ctx context.Context, cfg config.Config, hostConfig platform.Config, httpClient *http.Client) error {
	for _, backend := range cfg.Backends {
		if !backend.Enabled || backend.Kind != "ollama" {
			continue
		}
		client, err := platform.NewOllamaHostClient(backend.URL, httpClient)
		if err != nil {
			return fmt.Errorf("configure Ollama backend %q: %w", backend.Name, err)
		}
		models, err := client.InstalledModels(ctx)
		if err != nil {
			return fmt.Errorf("discover Ollama backend %q before preload: %w", backend.Name, err)
		}
		model, err := platform.SelectInstalledModel(models, backend.Model)
		if err != nil {
			return fmt.Errorf("select Ollama backend %q model: %w", backend.Name, err)
		}
		if err := client.Preload(ctx, model, hostConfig); err != nil {
			return fmt.Errorf("preload Ollama backend %q: %w", backend.Name, err)
		}
	}
	return nil
}

func windowsVersion() string {
	version, err := syscall.GetVersion()
	if err != nil {
		return "Windows " + runtime.GOARCH
	}
	major := byte(version)
	minor := uint8(version >> 8)
	build := uint16(version >> 16)
	return fmt.Sprintf("Windows %d.%d build %d", major, minor, build)
}

func currentACPower() (bool, bool) {
	type powerStatus struct {
		ACLineStatus        byte
		BatteryFlag         byte
		BatteryLifePercent  byte
		SystemStatusFlag    byte
		BatteryLifeTime     uint32
		BatteryFullLifeTime uint32
	}
	var status powerStatus
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("GetSystemPowerStatus")
	result, _, _ := proc.Call(uintptr(unsafe.Pointer(&status)))
	if result == 0 || status.ACLineStatus == 255 {
		return false, false
	}
	return status.ACLineStatus == 1, true
}

func openBrowser(uri string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", uri).Start()
}
