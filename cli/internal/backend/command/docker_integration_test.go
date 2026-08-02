//go:build integration

package command

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kunalshah017/myference/cli/internal/backend"
)

func TestRunnerExecutesOnlyInsideRealReadOnlyDockerContainer(t *testing.T) {
	image := os.Getenv("MYFERENCE_TEST_AGENT_IMAGE")
	if image == "" {
		t.Skip("MYFERENCE_TEST_AGENT_IMAGE must be a digest-pinned image with /bin/sh")
	}
	listener, err := net.Listen("tcp4", "0.0.0.0:0")
	if err != nil {
		t.Fatal(err)
	}
	hostServer := &http.Server{Handler: http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) { _, _ = response.Write([]byte("HOST_REACHABLE")) })}
	go hostServer.Serve(listener)
	defer hostServer.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	probe := fmt.Sprintf("gateway=$(ip route | awk '/default/{print $3}'); wget -q -T 1 -O - http://$gateway:%d 2>/dev/null || printf HOST_BLOCKED; pwd; cat src/main.go; printf '%%s' \"$OPENAI_API_KEY\"", port)
	runner, err := New(image, []string{"sh", "-c", probe}, "codex", "agent", "long-lived-secret", 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	var output strings.Builder
	request := backend.Request{Model: "agent", Prompt: "inspect", MaximumOutputTokens: 100, Workspace: []backend.WorkspaceFile{{Path: "src/main.go", Content: []byte("package main")}}}
	if _, err := runner.Generate(t.Context(), request, func(value string) error { output.WriteString(value); return nil }); err != nil {
		t.Fatal(err)
	}
	if text := output.String(); !strings.Contains(text, "HOST_BLOCKED") || strings.Contains(text, "HOST_REACHABLE") || !strings.Contains(text, "/workspace") || !strings.Contains(text, "package main") || !strings.Contains(text, "myference-job-") || strings.Contains(text, "long-lived-secret") {
		t.Fatalf("unexpected isolated output: %q", text)
	}
}
