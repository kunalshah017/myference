package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
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
	if len(loaded.Backends) != 1 || !loaded.Backends[0].Enabled {
		t.Fatalf("unexpected backend state: %+v", loaded.Backends)
	}
}

func TestBackendAddSupportsCloudAndCommandAgentsWithoutPersistingSecrets(t *testing.T) {
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
	capacity := v1.Capacity{Available: 1, Offers: []v1.OfferCapacity{offerCapacity(config.Backend{Name: "local", Kind: "ollama", Model: "qwen", PriceVersion: 1})}}
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
	offer := offerCapacity(config.Backend{Name: "local", Kind: "ollama", Model: "qwen", PriceVersion: 4, Enabled: true})
	if offer.PriceVersion != 4 || len(offer.OfferHash) != 66 || len(offer.ModelHash) != 66 || len(offer.CapabilityHash) != 66 || offer.BackendKind != "ollama" || !slices.Equal(offer.Capabilities, []string{"stream", "text"}) {
		t.Fatalf("unexpected offer proof keys: %+v", offer)
	}
	agent := offerCapacity(config.Backend{Name: "agent", Kind: "codex", Model: "codex", PriceVersion: 1, Enabled: true})
	if !slices.Equal(agent.Capabilities, []string{"stream", "text", "workspace"}) {
		t.Fatalf("command agent lacks workspace capability: %+v", agent)
	}
}
