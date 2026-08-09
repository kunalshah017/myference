package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/kunalshah017/myference/cli/internal/backend"
	"github.com/kunalshah017/myference/cli/internal/config"
	v1 "github.com/kunalshah017/myference/protocol/v1"
)

func TestBackendCommandsAddListStartStopAndStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(path, config.Config{ServerURL: "https://api.myference.network", AccountID: "acct-1", MachineID: "mach-1"}); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	for _, args := range [][]string{
		{"backend", "add", "--config", path, "--name", "local", "--model", "qwen2.5:0.5b", "--url", "http://127.0.0.1:11434"},
		{"backend", "version", "--config", path, "--name", "local", "--price-version", "2"},
		{"backend", "stop", "--config", path, "--name", "local"},
		{"backend", "start", "--config", path, "--name", "local"},
		{"backend", "list", "--config", path},
		{"status", "--config", path},
	} {
		if err := run(args, &output); err != nil {
			t.Fatalf("run(%v): %v", args, err)
		}
	}
	text := output.String()
	for _, expected := range []string{"local", "qwen2.5:0.5b", "enabled"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("output %q does not contain %q", text, expected)
		}
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Backends) != 1 || !loaded.Backends[0].Enabled || loaded.Backends[0].PriceVersion != 2 {
		t.Fatalf("unexpected backend state: %+v", loaded.Backends)
	}
}

func TestEntryModeChoosesTUIOnlyForInteractiveNoArgumentUse(t *testing.T) {
	for _, test := range []struct {
		name                string
		args                []string
		stdinTTY, stdoutTTY bool
		want                applicationEntryMode
	}{
		{name: "interactive", stdinTTY: true, stdoutTTY: true, want: entryTUI},
		{name: "piped input", stdinTTY: false, stdoutTTY: true, want: entryUsage},
		{name: "piped output", stdinTTY: true, stdoutTTY: false, want: entryUsage},
		{name: "command", args: []string{"status"}, want: entryCommand},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := entryMode(test.args, test.stdinTTY, test.stdoutTTY); got != test.want {
				t.Fatalf("entryMode=%v want=%v", got, test.want)
			}
		})
	}
}

func TestRunApplicationInvokesTUIForInteractiveNoArgumentUse(t *testing.T) {
	called := false
	err := runApplication(context.Background(), nil, strings.NewReader(""), &bytes.Buffer{}, true, true, func(context.Context, io.Reader, io.Writer) error {
		called = true
		return nil
	})
	if err != nil || !called {
		t.Fatalf("called=%v err=%v", called, err)
	}
}

func TestCodexCommandArgumentsAreEphemeralReadOnlyAndNonInteractive(t *testing.T) {
	want := []string{"exec", "--ephemeral", "--sandbox", "read-only", "--skip-git-repo-check", "--model", "gpt-codex", "-"}
	if got := commandArguments("codex", "gpt-codex"); !reflect.DeepEqual(got, want) {
		t.Fatalf("commandArguments(codex)=%v, want %v", got, want)
	}
}

func TestBackendAddReplacesOpenAIWithNativeCodex(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	initial := config.Config{ServerURL: "https://api.myference.network", AccountID: "acct-1", MachineID: "mach-1", Backends: []config.Backend{{Name: "codex-cli-terra", Kind: "openai", URL: "https://api.openai.com", Model: "gpt-5.6-terra", PriceVersion: 1, Enabled: true}}}
	if err := config.Save(path, initial); err != nil {
		t.Fatal(err)
	}
	deleted := ""
	dependencies := backendCommandDependencies{
		SaveCredential:   func(string, string, string) error { t.Fatal("native Codex stored a credential"); return nil },
		DeleteCredential: func(service, account string) error { deleted = service + ":" + account; return nil },
		NewNativeCodex:   func(string, time.Duration) (backend.Backend, error) { return staticBackend{}, nil },
	}
	args := []string{"add", "--replace", "--config", path, "--name", "codex-cli-terra", "--kind", "codex", "--model", "gpt-5.6-terra", "--price-version", "1"}
	if err := runBackendWithDependencies(args, &bytes.Buffer{}, dependencies); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Backends) != 1 || loaded.Backends[0].Kind != "codex" || loaded.Backends[0].Image != "" || loaded.Backends[0].URL != "" || loaded.Backends[0].PriceVersion != 1 {
		t.Fatalf("backends=%+v", loaded.Backends)
	}
	if deleted != "myference.backend:mach-1/codex-cli-terra" {
		t.Fatalf("deleted=%q", deleted)
	}
}

func TestBackendAddNativeCodexRequiresReplaceForDuplicate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(path, config.Config{ServerURL: "https://api.myference.network", AccountID: "acct-1", MachineID: "mach-1", Backends: []config.Backend{{Name: "code", Kind: "openai", Model: "code", Enabled: true}}}); err != nil {
		t.Fatal(err)
	}
	dependencies := backendCommandDependencies{NewNativeCodex: func(string, time.Duration) (backend.Backend, error) { return staticBackend{}, nil }}
	err := runBackendWithDependencies([]string{"add", "--config", path, "--name", "code", "--kind", "codex", "--model", "code"}, &bytes.Buffer{}, dependencies)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("error=%v", err)
	}
}

func TestBackendRemoveDeletesBackendAndCredential(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	initial := config.Config{ServerURL: "https://api.myference.network", AccountID: "acct-1", MachineID: "mach-1", Backends: []config.Backend{
		{Name: "retired", Kind: "openai", Model: "retired-model", Enabled: false},
		{Name: "kept", Kind: "codex", Model: "gpt-5.6-terra", Enabled: true},
	}}
	if err := config.Save(path, initial); err != nil {
		t.Fatal(err)
	}
	deleted := ""
	dependencies := backendCommandDependencies{DeleteCredential: func(service, account string) error {
		deleted = service + ":" + account
		return nil
	}}
	if err := runBackendWithDependencies([]string{"remove", "--config", path, "--name", "retired"}, &bytes.Buffer{}, dependencies); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Backends) != 1 || loaded.Backends[0].Name != "kept" {
		t.Fatalf("backends=%+v", loaded.Backends)
	}
	if deleted != "myference.backend:mach-1/retired" {
		t.Fatalf("deleted=%q", deleted)
	}
}

func TestBackendRemoveNativeCodexDoesNotDeleteCredential(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(path, config.Config{ServerURL: "https://api.myference.network", AccountID: "acct-1", MachineID: "mach-1", Backends: []config.Backend{{Name: "code", Kind: "codex", Model: "gpt-5.6-terra"}}}); err != nil {
		t.Fatal(err)
	}
	dependencies := backendCommandDependencies{DeleteCredential: func(string, string) error {
		t.Fatal("native Codex removal deleted a credential")
		return nil
	}}
	if err := runBackendWithDependencies([]string{"remove", "--config", path, "--name", "code"}, &bytes.Buffer{}, dependencies); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Backends) != 0 {
		t.Fatalf("backends=%+v", loaded.Backends)
	}
}

func TestBackendRemoveRequiresExistingName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(path, config.Config{ServerURL: "https://api.myference.network", AccountID: "acct-1", MachineID: "mach-1"}); err != nil {
		t.Fatal(err)
	}
	err := runBackendWithDependencies([]string{"remove", "--config", path, "--name", "missing"}, &bytes.Buffer{}, backendCommandDependencies{})
	if err == nil || !strings.Contains(err.Error(), "backend not found") {
		t.Fatalf("error=%v", err)
	}
}

func TestConfiguredBackendSelectsNativeCodexWithoutCredential(t *testing.T) {
	called := false
	result, err := configuredBackendWithNative(config.Backend{Name: "code", Kind: "codex", Model: "gpt-5.6-terra"}, "machine", func(string, string) (string, error) {
		t.Fatal("native Codex loaded a backend credential")
		return "", nil
	}, func(model string, _ time.Duration) (backend.Backend, error) {
		called = model == "gpt-5.6-terra"
		return staticBackend{}, nil
	})
	if err != nil || result == nil || !called {
		t.Fatalf("result=%v called=%v err=%v", result, called, err)
	}
}

func TestConfiguredBackendSelectsNativeClaudeWithoutCredential(t *testing.T) {
	called := false
	result, err := configuredBackendWithNatives(config.Backend{Name: "claude", Kind: "claude", Model: "sonnet"}, "machine", func(string, string) (string, error) {
		t.Fatal("native Claude loaded a backend credential")
		return "", nil
	}, func(string, time.Duration) (backend.Backend, error) {
		return staticBackend{}, nil
	}, func(model string, _ time.Duration) (backend.Backend, error) {
		called = model == "sonnet"
		return staticBackend{}, nil
	})
	if err != nil || result == nil || !called {
		t.Fatalf("result=%v called=%v err=%v", result, called, err)
	}
}

func TestBackendAddNativeClaudeUsesExistingLogin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(path, config.Config{ServerURL: "https://api.myference.network", AccountID: "acct-1", MachineID: "mach-1"}); err != nil {
		t.Fatal(err)
	}
	dependencies := backendCommandDependencies{
		SaveCredential: func(string, string, string) error { t.Fatal("native Claude stored a credential"); return nil },
		NewNativeClaude: func(model string, _ time.Duration) (backend.Backend, error) {
			if model != "sonnet" {
				t.Fatalf("model=%q", model)
			}
			return staticBackend{}, nil
		},
	}
	if err := runBackendWithDependencies([]string{"add", "--config", path, "--name", "claude", "--kind", "claude", "--model", "sonnet"}, &bytes.Buffer{}, dependencies); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(path)
	if err != nil || len(loaded.Backends) != 1 || loaded.Backends[0].Image != "" || loaded.Backends[0].Kind != "claude" {
		t.Fatalf("backends=%+v err=%v", loaded.Backends, err)
	}
}

func TestCodexDenyToolCreatesMarkerWithoutEchoingInput(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "tool-attempted")
	t.Setenv("MYFERENCE_CODEX_TOOL_MARKER", marker)
	var output bytes.Buffer
	secretInput := `{"tool_name":"Bash","tool_input":{"command":"echo secret-value"}}`
	if err := runCodexDenyTool(strings.NewReader(secretInput), &output); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "secret-value") || !strings.Contains(output.String(), `"permissionDecision":"deny"`) {
		t.Fatalf("output=%q", output.String())
	}
}

type staticBackend struct{}

func (staticBackend) Models(context.Context) ([]backend.Model, error) {
	return []backend.Model{{Name: "model"}}, nil
}
func (staticBackend) Generate(context.Context, backend.Request, func(string) error) (backend.Usage, error) {
	return backend.Usage{}, nil
}

func TestConfigureLocalHostDiscoversOllamaAndIsIdempotent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/tags" {
			http.NotFound(response, request)
			return
		}
		response.Header().Set("content-type", "application/json")
		_, _ = response.Write([]byte(`{"models":[{"name":"qwen2.5:0.5b","digest":"sha256:real-runtime","size":1024}]}`))
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(path, config.Config{ServerURL: "https://api.myference.network", AccountID: "acct-1", MachineID: "mach-1"}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		selected, err := configureLocalHost(context.Background(), path, server.URL, "", server.Client())
		if err != nil {
			t.Fatal(err)
		}
		if selected.Name != "qwen2.5:0.5b" || selected.Digest != "sha256:real-runtime" {
			t.Fatalf("selected=%+v", selected)
		}
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Backends) != 1 || loaded.Backends[0].Kind != "ollama" || loaded.Backends[0].Model != "qwen2.5:0.5b" || !loaded.Backends[0].Enabled {
		t.Fatalf("backends=%+v", loaded.Backends)
	}
}

func TestBackendAddSupportsCloudAndCommandAgentsWithoutPersistingSecrets(t *testing.T) {
	dockerDirectory := t.TempDir()
	dockerName := "docker"
	if runtime.GOOS == "windows" {
		dockerName = "docker.exe"
	}
	if err := os.WriteFile(filepath.Join(dockerDirectory, dockerName), []byte("test docker"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dockerDirectory+string(os.PathListSeparator)+os.Getenv("PATH"))
	proxy := filepath.Join(t.TempDir(), "myference-agent-proxy")
	if err := os.WriteFile(proxy, []byte("test proxy"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MYFERENCE_AGENT_PROXY", proxy)
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(path, config.Config{ServerURL: "https://api.myference.network", AccountID: "acct-1", MachineID: "mach-1"}); err != nil {
		t.Fatal(err)
	}
	saved := map[string]string{}
	save := func(service, account, secret string) error { saved[service+":"+account] = secret; return nil }
	for _, args := range [][]string{{"add", "--config", path, "--name", "cloud", "--kind", "openai", "--model", "remote", "--url", "https://provider.example", "--secret", "cloud-secret"}, {"add", "--config", path, "--name", "code", "--kind", "codex", "--model", "codex", "--image", "registry.example/codex@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "--secret", "agent-secret"}} {
		if err := runBackendWithCredentials(args, &bytes.Buffer{}, save); err != nil {
			t.Fatalf("args=%v err=%v", args, err)
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "cloud-secret") || strings.Contains(string(raw), "agent-secret") {
		t.Fatal("backend secret persisted in config")
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Backends) != 2 || loaded.Backends[0].Kind != "openai" || loaded.Backends[1].Kind != "codex" {
		t.Fatalf("backends=%+v", loaded.Backends)
	}
	if saved["myference.backend:mach-1/cloud"] != "cloud-secret" || saved["myference.backend:mach-1/code"] != "agent-secret" {
		t.Fatalf("saved=%v", saved)
	}
}

func TestLoginUsesBrowserDeviceFlowAndKeepsTokenOutOfConfig(t *testing.T) {
	var requestedSigner string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth/device":
			var input map[string]string
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Fatal(err)
			}
			requestedSigner = input["signer_address"]
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"device_code": "device-secret", "user_code": "ABCD1234", "verification_uri": "https://app.myference.test/devices", "expires_at": time.Now().Add(time.Minute), "chain_id": 10143, "contract_address": "0x4444444444444444444444444444444444444444"})
		case "/auth/device/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"machine": map[string]string{"id": "mach-1", "account_id": "acct-1", "name": "spare-mac"}, "token": "mach-1.machine-secret"})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	path := filepath.Join(t.TempDir(), "config.json")
	var opened string
	saved := map[string]string{}
	var output bytes.Buffer
	err := runLogin(context.Background(), []string{"--server", server.URL, "--name", "spare-mac", "--config", path}, &output, loginDependencies{
		HTTPClient:       server.Client(),
		OpenBrowser:      func(uri string) error { opened = uri; return nil },
		SaveCredential:   func(service, account, secret string) error { saved[service+":"+account] = secret; return nil },
		DeleteCredential: func(_, _ string) error { return nil },
		Wait:             func(context.Context, time.Duration) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if opened != "https://app.myference.test/devices" || saved["myference.machine:mach-1"] != "mach-1.machine-secret" || saved["myference.signer:mach-1"] == "" || !strings.HasPrefix(requestedSigner, "0x") || len(requestedSigner) != 42 {
		t.Fatalf("opened=%q signer=%q saved=%v", opened, requestedSigner, saved)
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ServerURL != server.URL || loaded.AccountID != "acct-1" || loaded.MachineID != "mach-1" || loaded.ChainID != 10143 || loaded.ContractAddress != "0x4444444444444444444444444444444444444444" {
		t.Fatalf("unexpected config: %+v", loaded)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "machine-secret") {
		t.Fatal("machine token was written to config")
	}
	if !strings.Contains(output.String(), "ABCD1234") {
		t.Fatalf("user code missing from output: %q", output.String())
	}
}

func TestRelayURLUsesOutboundWebSocketEndpoint(t *testing.T) {
	for input, want := range map[string]string{
		"https://api.myference.network": "wss://api.myference.network/relay",
		"http://127.0.0.1:8080/api/":    "ws://127.0.0.1:8080/api/relay",
	} {
		got, err := relayURL(input)
		if err != nil {
			t.Fatalf("relayURL(%q): %v", input, err)
		}
		if got != want {
			t.Fatalf("relayURL(%q)=%q, want %q", input, got, want)
		}
	}
}

func TestProductionDefaultsUseBrandedDomains(t *testing.T) {
	if defaultServerURL != "https://api.myference.xyz" || defaultWebURL != "https://myference.xyz" {
		t.Fatalf("server=%q web=%q", defaultServerURL, defaultWebURL)
	}
}

func TestWindowsCommandsReachTheNativeDispatchBoundary(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("native Windows dispatch is compiled only on Windows")
	}
	err := run([]string{"windows", "focus"}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "focus <start|status|restore>") {
		t.Fatalf("run(windows focus) error = %v, want focus usage", err)
	}
	err = run([]string{"windows", "headless"}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "headless <install|start|status|restore>") {
		t.Fatalf("run(windows headless) error = %v, want headless usage", err)
	}
}

func TestWindowsCommandRejectsLANAction(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("native Windows dispatch is compiled only on Windows")
	}
	err := run([]string{"windows", "lan-check"}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "unknown Windows action") {
		t.Fatalf("run(windows lan-check) error = %v, want unknown Windows action", err)
	}
}

func TestHostLoginArgumentsPreserveNoBrowser(t *testing.T) {
	got := hostLoginArgs("https://api.myference.xyz", "/tmp/myference.json", true)
	if !slices.Contains(got, "--no-browser") {
		t.Fatalf("headless host login arguments=%v", got)
	}
}

func TestServeFlagsPreserveConfigAndBatteryOverride(t *testing.T) {
	path, allowBattery, err := parseServeFlags([]string{"--config", "provider.json", "--allow-battery"})
	if err != nil || path != "provider.json" || !allowBattery {
		t.Fatalf("path=%q allowBattery=%v err=%v", path, allowBattery, err)
	}
}

func TestServeWithReconnectSurvivesRelayRestart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	var output bytes.Buffer
	err := serveWithReconnect(ctx, &output, time.Millisecond, time.Millisecond, func(context.Context) error {
		calls++
		if calls == 3 {
			cancel()
			return context.Canceled
		}
		return fmt.Errorf("relay restart %d", calls)
	})
	if err != nil || calls != 3 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	if got := strings.Count(output.String(), "relay disconnected"); got != 2 {
		t.Fatalf("reconnect messages=%d output=%q", got, output.String())
	}
}

func TestStatusJSONProvidesPlatformAttestationWithoutSecrets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.Save(path, config.Config{ServerURL: "https://api.myference.network", AccountID: "acct", MachineID: "machine"}); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	cfg, _ := config.Load(path)
	capacity := v1.Capacity{Available: 1, Offers: []v1.OfferCapacity{offerCapacity(config.Backend{Name: "local", Kind: "ollama", Model: "qwen", PriceVersion: 1}, backend.Model{Name: "qwen", Digest: "sha256:test"})}}
	if err := writeStatusJSON(cfg, capacity, &output, func(string, string) (string, error) { return fmt.Sprintf("%x", crypto.FromECDSA(key)), nil }, func() time.Time { return time.Unix(1_700_000_000, 0) }); err != nil {
		t.Fatal(err)
	}
	var status struct {
		MachineID       string      `json:"machine_id"`
		GOOS            string      `json:"goos"`
		GOARCH          string      `json:"goarch"`
		Signature       string      `json:"attestation_signature"`
		Capacity        v1.Capacity `json:"capacity"`
		CapacitySHA256  string      `json:"capacity_sha256"`
		CapacityPayload string      `json:"capacity_payload"`
	}
	if err := json.Unmarshal(output.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.MachineID != "machine" || status.GOOS == "" || status.GOARCH == "" || len(status.Signature) != 132 || status.Capacity.Available != 1 || len(status.CapacitySHA256) != 64 || status.CapacityPayload == "" {
		t.Fatalf("status=%+v", status)
	}
}

func TestOfferCapacityCarriesDeterministicMonadProofKeys(t *testing.T) {
	offer := offerCapacity(config.Backend{Name: "local", Kind: "ollama", Model: "qwen", PriceVersion: 4, Enabled: true}, backend.Model{Name: "qwen", Digest: "sha256:runtime"})
	if offer.PriceVersion != 4 || len(offer.OfferHash) != 66 || len(offer.ModelHash) != 66 || len(offer.CapabilityHash) != 66 || offer.BackendKind != "ollama" || !slices.Equal(offer.Capabilities, []string{"stream", "text"}) {
		t.Fatalf("unexpected offer proof keys: %+v", offer)
	}
	if offer.EvidenceKind != "ollama_digest" || offer.EvidenceDigest != "sha256:runtime" || offer.MeteringMode != "tokens_and_compute" {
		t.Fatalf("ollama runtime evidence missing: %+v", offer)
	}
	nativeCodex := offerCapacity(config.Backend{Name: "native-code", Kind: "codex", Model: "gpt-5.6-terra", PriceVersion: 1, Enabled: true}, backend.Model{Name: "gpt-5.6-terra"})
	if !slices.Equal(nativeCodex.Capabilities, []string{"stream", "text"}) || nativeCodex.EvidenceKind != "upstream_model" || nativeCodex.EvidenceDigest != "gpt-5.6-terra" || nativeCodex.MeteringMode != "tokens_and_compute" {
		t.Fatalf("native Codex evidence is invalid: %+v", nativeCodex)
	}
	if err := nativeCodex.Validate(); err != nil {
		t.Fatalf("native Codex offer is invalid: %v", err)
	}
	nativeClaude := offerCapacity(config.Backend{Name: "native-claude", Kind: "claude", Model: "sonnet", PriceVersion: 1, Enabled: true}, backend.Model{Name: "sonnet"})
	if !slices.Equal(nativeClaude.Capabilities, []string{"stream", "text"}) || nativeClaude.EvidenceKind != "upstream_model" || nativeClaude.MeteringMode != "tokens_and_compute" {
		t.Fatalf("native Claude evidence is invalid: %+v", nativeClaude)
	}
	agent := offerCapacity(config.Backend{Name: "agent", Kind: "codex", Model: "codex", Image: "codex@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", PriceVersion: 1, Enabled: true}, backend.Model{Name: "codex"})
	if !slices.Equal(agent.Capabilities, []string{"stream", "text", "workspace"}) {
		t.Fatalf("command agent lacks workspace capability: %+v", agent)
	}
	if agent.EvidenceKind != "runtime_image" || agent.EvidenceDigest != "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" || agent.MeteringMode != "compute_only" {
		t.Fatalf("command agent evidence is unsafe: %+v", agent)
	}
}
