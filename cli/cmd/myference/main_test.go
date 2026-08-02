package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

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
