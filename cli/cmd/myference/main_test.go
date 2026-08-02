package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kunalshah017/myference/cli/internal/config"
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

func TestLoginUsesBrowserDeviceFlowAndKeepsTokenOutOfConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth/device":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"device_code": "device-secret", "user_code": "ABCD1234", "verification_uri": "https://app.myference.test/devices", "expires_at": time.Now().Add(time.Minute)})
		case "/auth/device/token":
			_ = json.NewEncoder(w).Encode(map[string]any{"machine": map[string]string{"id": "mach-1", "account_id": "acct-1", "name": "spare-mac"}, "token": "mach-1.machine-secret"})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	path := filepath.Join(t.TempDir(), "config.json")
	var opened, savedAccount, savedSecret string
	var output bytes.Buffer
	err := runLogin(context.Background(), []string{"--server", server.URL, "--name", "spare-mac", "--config", path}, &output, loginDependencies{
		HTTPClient:     server.Client(),
		OpenBrowser:    func(uri string) error { opened = uri; return nil },
		SaveCredential: func(_, account, secret string) error { savedAccount, savedSecret = account, secret; return nil },
		Wait:           func(context.Context, time.Duration) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if opened != "https://app.myference.test/devices" || savedAccount != "mach-1" || savedSecret != "mach-1.machine-secret" {
		t.Fatalf("opened=%q account=%q secret=%q", opened, savedAccount, savedSecret)
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ServerURL != server.URL || loaded.AccountID != "acct-1" || loaded.MachineID != "mach-1" {
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
