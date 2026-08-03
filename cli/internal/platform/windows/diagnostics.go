package windows

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

type DoctorState struct {
	WindowsVersion           string
	OllamaPath               string
	ConfiguredModel          string
	InstalledModels          []string
	DockerRequired           bool
	DockerPath               string
	DockerEngineReady        bool
	DockerEngineOS           string
	DockerMissingImages      []string
	DockerError              string
	CredentialStoreAvailable bool
	OnACPowerKnown           bool
	OnACPower                bool
	ServiceInstalled         bool
	ConfigReadable           bool
}

type DoctorFinding struct {
	Name    string
	OK      bool
	Message string
	Action  string
}

func DoctorFindings(state DoctorState) []DoctorFinding {
	modelInstalled := state.ConfiguredModel != "" && slices.Contains(state.InstalledModels, state.ConfiguredModel)
	dockerOK := !state.DockerRequired || (state.DockerPath != "" && state.DockerEngineReady && state.DockerEngineOS == "linux" && len(state.DockerMissingImages) == 0 && state.DockerError == "")
	powerOK := state.OnACPowerKnown && state.OnACPower
	return []DoctorFinding{
		finding("Windows", state.WindowsVersion != "", valueOr(state.WindowsVersion, "Windows version unavailable"), "Run this command on a supported Windows host"),
		finding("Ollama", state.OllamaPath != "", valueOr(state.OllamaPath, "Ollama executable not found"), "Install Ollama and ensure ollama.exe is on PATH"),
		finding("Model", modelInstalled, modelMessage(state, modelInstalled), modelAction(state)),
		finding("Docker", dockerOK, dockerMessage(state, dockerOK), dockerAction(state)),
		finding("Credentials", state.CredentialStoreAvailable, boolMessage(state.CredentialStoreAvailable, "Windows Credential Manager is available", "credential storage is unavailable"), "Enable Windows Credential Manager and sign in with `myference login`"),
		finding("Power", powerOK, powerMessage(state), "Connect AC power or explicitly use --allow-battery"),
		finding("Service", state.ServiceInstalled, boolMessage(state.ServiceInstalled, "Myference Provider task is installed", "provider service is not installed"), "Run `myference service install`"),
		finding("Config", state.ConfigReadable, boolMessage(state.ConfigReadable, "provider configuration is readable", "provider configuration is missing or invalid"), "Run `myference login` and configure a backend"),
	}
}

func WriteDoctor(output io.Writer, findings []DoctorFinding) error {
	for _, item := range findings {
		status := "PASS"
		if !item.OK {
			status = "FAIL"
		}
		if _, err := fmt.Fprintf(output, "[%s] %s: %s\\n", status, item.Name, item.Message); err != nil {
			return err
		}
		if !item.OK && item.Action != "" {
			if _, err := fmt.Fprintf(output, "       Action: %s\\n", item.Action); err != nil {
				return err
			}
		}
	}
	return nil
}

func finding(name string, ok bool, message, action string) DoctorFinding {
	return DoctorFinding{Name: name, OK: ok, Message: message, Action: action}
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func boolMessage(ok bool, success, failure string) string {
	if ok {
		return success
	}
	return failure
}

func modelMessage(state DoctorState, installed bool) string {
	if state.ConfiguredModel == "" {
		return "no Ollama model is configured"
	}
	if !installed {
		return fmt.Sprintf("configured model %q is not installed", state.ConfiguredModel)
	}
	return fmt.Sprintf("configured model %q is installed", state.ConfiguredModel)
}

func modelAction(state DoctorState) string {
	if state.ConfiguredModel == "" {
		return "Configure an installed Ollama model with `myference host --model NAME`"
	}
	return "Run `ollama pull " + state.ConfiguredModel + "`"
}

func dockerMessage(state DoctorState, ok bool) string {
	if !state.DockerRequired {
		return "Docker is not required by enabled backends"
	}
	if ok {
		return fmt.Sprintf("Docker Linux engine is ready at %s", state.DockerPath)
	}
	if state.DockerError != "" {
		return state.DockerError
	}
	if state.DockerPath == "" {
		return "Docker is required by an enabled command-agent backend but was not found"
	}
	if !state.DockerEngineReady {
		return "Docker Desktop is installed but its engine is not ready"
	}
	if state.DockerEngineOS != "linux" {
		return fmt.Sprintf("Docker Desktop is using %q containers instead of Linux containers", state.DockerEngineOS)
	}
	return "missing digest-pinned command-agent image(s): " + strings.Join(state.DockerMissingImages, ", ")
}

func dockerAction(state DoctorState) string {
	if !state.DockerRequired {
		return ""
	}
	if state.DockerPath == "" {
		return "Install Docker Desktop or disable command-agent backends"
	}
	if state.DockerEngineReady && state.DockerEngineOS != "linux" {
		return "Switch Docker Desktop to Linux containers"
	}
	return "Run `myference serve`; it starts Docker Desktop and pulls missing digest-pinned images"
}

func RenderHeadlessStatus(active bool, docker DockerStatus) string {
	state := "inactive"
	if active {
		state = "installed"
	}
	var output strings.Builder
	fmt.Fprintf(&output, "Windows headless mode %s\n", state)
	if docker.DockerPath == "" && docker.Error == "" && !docker.EngineReady && len(docker.MissingImages) == 0 {
		return output.String()
	}
	if docker.EngineReady {
		fmt.Fprintf(&output, "Docker engine ready (%s)\n", docker.EngineOS)
	} else if docker.Error != "" {
		fmt.Fprintf(&output, "Docker not ready: %s\n", docker.Error)
	}
	if len(docker.MissingImages) > 0 {
		fmt.Fprintf(&output, "Missing command-agent images: %s\n", strings.Join(docker.MissingImages, ", "))
	}
	return output.String()
}

func powerMessage(state DoctorState) string {
	if !state.OnACPowerKnown {
		return "AC/battery state is unavailable"
	}
	if state.OnACPower {
		return "running on AC power"
	}
	return "running on battery power"
}
