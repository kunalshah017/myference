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
	dockerOK := !state.DockerRequired || state.DockerPath != ""
	powerOK := state.OnACPowerKnown && state.OnACPower
	return []DoctorFinding{
		finding("Windows", state.WindowsVersion != "", valueOr(state.WindowsVersion, "Windows version unavailable"), "Run this command on a supported Windows host"),
		finding("Ollama", state.OllamaPath != "", valueOr(state.OllamaPath, "Ollama executable not found"), "Install Ollama and ensure ollama.exe is on PATH"),
		finding("Model", modelInstalled, modelMessage(state, modelInstalled), modelAction(state)),
		finding("Docker", dockerOK, dockerMessage(state, dockerOK), "Install Docker Desktop or disable command-agent backends"),
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
		return state.DockerPath
	}
	return "Docker is required by an enabled command-agent backend but was not found"
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
