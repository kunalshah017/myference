package windows

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteDoctorPrintsActionForFailures(t *testing.T) {
	var output bytes.Buffer
	if err := WriteDoctor(&output, []DoctorFinding{{Name: "Ollama", OK: false, Message: "not found", Action: "Install Ollama"}}); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); !strings.Contains(got, "[FAIL] Ollama: not found") || !strings.Contains(got, "Action: Install Ollama") {
		t.Fatalf("doctor output = %q", got)
	}
}

func TestDoctorFindingsAreActionable(t *testing.T) {
	findings := DoctorFindings(DoctorState{
		WindowsVersion:           "Windows 11",
		ConfiguredModel:          "missing:latest",
		InstalledModels:          []string{"qwen:latest"},
		DockerRequired:           true,
		OnACPowerKnown:           true,
		CredentialStoreAvailable: false,
		ConfigReadable:           false,
	})

	wantFailures := map[string]string{
		"Ollama":      "Install Ollama",
		"Model":       "ollama pull missing:latest",
		"Docker":      "Install Docker Desktop",
		"Credentials": "Windows Credential Manager",
		"Power":       "Connect AC power",
		"Service":     "myference service install",
		"Config":      "myference login",
	}
	for _, finding := range findings {
		want, tracked := wantFailures[finding.Name]
		if !tracked {
			continue
		}
		if finding.OK || !strings.Contains(finding.Action, want) {
			t.Fatalf("finding %q = %+v, want failed action containing %q", finding.Name, finding, want)
		}
		delete(wantFailures, finding.Name)
	}
	if len(wantFailures) != 0 {
		t.Fatalf("missing findings: %v", wantFailures)
	}
}

func TestDoctorFindingsAcceptHealthyHost(t *testing.T) {
	findings := DoctorFindings(DoctorState{
		WindowsVersion:           "Windows 11",
		OllamaPath:               `C:\\Program Files\\Ollama\\ollama.exe`,
		ConfiguredModel:          "qwen:latest",
		InstalledModels:          []string{"qwen:latest"},
		DockerRequired:           true,
		DockerPath:               `C:\\Program Files\\Docker\\docker.exe`,
		CredentialStoreAvailable: true,
		OnACPowerKnown:           true,
		OnACPower:                true,
		ServiceInstalled:         true,
		ConfigReadable:           true,
	})
	for _, finding := range findings {
		if !finding.OK {
			t.Fatalf("healthy finding failed: %+v", finding)
		}
	}
}
