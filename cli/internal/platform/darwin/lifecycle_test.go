//go:build darwin

package darwin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLifecycleWritesSecretFreeLaunchAgentAndUsesLaunchctl(t *testing.T) {
	dir := t.TempDir()
	executable := filepath.Join(dir, "myference")
	if err := os.WriteFile(executable, []byte("binary"), 0o700); err != nil {
		t.Fatal(err)
	}
	config := filepath.Join(dir, "config.json")
	if err := os.WriteFile(config, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	var commands [][]string
	lifecycle, err := New(executable, config, filepath.Join(dir, "com.myference.provider.plist"), filepath.Join(dir, "provider.log"), func(name string, args ...string) error {
		commands = append(commands, append([]string{name}, args...))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.Install(); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.Install(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(lifecycle.PlistPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, expected := range []string{executable, "serve", "--config", config, "KeepAlive", "ProcessType", "Background"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("plist missing %q: %s", expected, text)
		}
	}
	for _, forbidden := range []string{"private_key", "Bearer ", "machine-secret"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("plist leaked %q", forbidden)
		}
	}
	if err := lifecycle.Start(); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.Status(); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.Uninstall(); err != nil {
		t.Fatal(err)
	}
	if err := lifecycle.Uninstall(); err != nil {
		t.Fatal(err)
	}
	if len(commands) != 12 {
		t.Fatalf("commands=%v", commands)
	}
}

func TestLifecycleRejectsRelativePaths(t *testing.T) {
	if _, err := New("myference", "config", "agent.plist", "log", nil); err == nil {
		t.Fatal("accepted relative paths")
	}
}
