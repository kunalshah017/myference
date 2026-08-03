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
		DockerEngineReady:        true,
		DockerEngineOS:           "linux",
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
		DockerEngineReady:        true,
		DockerEngineOS:           "linux",
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

func TestDoctorFindingsRequireReadyLinuxEngineAndConfiguredImages(t *testing.T) {
	findings := DoctorFindings(DoctorState{
		WindowsVersion:    "Windows 11",
		DockerRequired:    true,
		DockerPath:        `C:\\Program Files\\Docker\\docker.exe`,
		DockerEngineReady: true,
		DockerEngineOS:    "linux",
		DockerMissingImages: []string{
			"ghcr.io/example/codex@sha256:missing",
		},
		OnACPowerKnown:           true,
		OnACPower:                true,
		CredentialStoreAvailable: true,
		ServiceInstalled:         true,
		ConfigReadable:           true,
	})
	for _, finding := range findings {
		if finding.Name == "Docker" {
			if finding.OK || !strings.Contains(finding.Message, "missing") || !strings.Contains(finding.Action, "serve") {
				t.Fatalf("Docker finding=%+v", finding)
			}
			return
		}
	}
	t.Fatal("Docker finding missing")
}

func TestRenderHeadlessStatusIncludesReadOnlyDockerReadiness(t *testing.T) {
	status := DockerStatus{DockerPath: "docker.exe", EngineReady: true, EngineOS: "linux", MissingImages: []string{"missing-image"}}
	got := RenderHeadlessStatus(true, status)
	for _, expected := range []string{"Windows headless mode installed", "Docker engine ready (linux)", "missing-image"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("status %q missing %q", got, expected)
		}
	}
}
