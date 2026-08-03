package command

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kunalshah017/myference/cli/internal/backend"
)

func TestContainerArgumentsIsolateHostAndNeverExposeLongLivedCredential(t *testing.T) {
	image := "agent-image@sha256:" + strings.Repeat("a", 64)
	runner, err := newRunner("/usr/bin/docker", "/usr/bin/myference-agent-proxy", image, []string{"codex", "exec", "-"}, "codex", "agent", "long-lived-secret", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	arguments := runner.dockerArguments("job", "/private/workspace", "job-internal", "http://job-proxy:8080", "job-token")
	joined := strings.Join(arguments, " ")
	for _, required := range []string{"--network job-internal", "--read-only", "--cap-drop=ALL", "no-new-privileges", "--pids-limit=256", "type=bind,source=/private/workspace,target=/workspace", "OPENAI_API_KEY=job-token", "OPENAI_BASE_URL=http://job-proxy:8080"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("missing isolation argument %q in %q", required, joined)
		}
	}
	if strings.Contains(joined, "long-lived-secret") || strings.Contains(joined, os.Getenv("HOME")) {
		t.Fatalf("host credential or home leaked into container arguments: %q", joined)
	}
	if strings.Contains(joined, "host.docker.internal") {
		t.Fatalf("agent can reach the host gateway: %q", joined)
	}
	if !slices.Contains(arguments, image) {
		t.Fatal("pinned agent image missing")
	}
}

func TestCredentialProxyInjectsSecretWithoutReturningItToAgent(t *testing.T) {
	seen := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		seen <- request.Header.Get("Authorization")
		_, _ = io.WriteString(response, "data: safe\n\n")
	}))
	defer upstream.Close()
	proxy, err := newCredentialProxyForTarget("codex", "real-provider-secret", "remote-model", 8, upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer proxy.Close()
	request, _ := http.NewRequest(http.MethodPost, proxy.url+"/v1/chat/completions", strings.NewReader(`{"model":"remote-model","max_tokens":8}`))
	request.Header.Set("Authorization", "Bearer "+proxy.token)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if <-seen != "Bearer real-provider-secret" || strings.Contains(string(raw), "real-provider-secret") {
		t.Fatalf("credential proxy leaked or failed to inject secret: %q", raw)
	}
}

func TestCredentialProxyEnforcesModelEndpointAndCumulativeTokenBudget(t *testing.T) {
	upstreamCalls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		upstreamCalls++
		response.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	proxy, err := newCredentialProxyForTarget("codex", "secret", "allowed-model", 10, upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer proxy.Close()
	do := func(path, body string) int {
		request, _ := http.NewRequest(http.MethodPost, proxy.url+path, strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+proxy.token)
		response, requestErr := http.DefaultClient.Do(request)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		defer response.Body.Close()
		return response.StatusCode
	}
	if status := do("/v1/chat/completions", `{"model":"other","max_tokens":1}`); status != http.StatusForbidden {
		t.Fatalf("wrong model status=%d", status)
	}
	if status := do("/v1/files", `{"model":"allowed-model","max_tokens":1}`); status != http.StatusForbidden {
		t.Fatalf("wrong endpoint status=%d", status)
	}
	if status := do("/v1/chat/completions", `{"model":"allowed-model","max_tokens":6}`); status != http.StatusOK {
		t.Fatalf("first request status=%d", status)
	}
	if status := do("/v1/responses", `{"model":"allowed-model","max_output_tokens":5}`); status != http.StatusPaymentRequired {
		t.Fatalf("over-budget request status=%d", status)
	}
	if status := do("/v1/messages", `{"model":"allowed-model","max_tokens":4}`); status != http.StatusOK {
		t.Fatalf("remaining-budget request status=%d", status)
	}
	if upstreamCalls != 2 {
		t.Fatalf("upstream calls=%d, want 2", upstreamCalls)
	}
}

func TestWorkspaceMaterializationRejectsEscapeAndUsesPrivateFiles(t *testing.T) {
	if _, err := materializeWorkspace([]backend.WorkspaceFile{{Path: "../escape", Content: []byte("x")}}); err == nil {
		t.Fatal("workspace escape accepted")
	}
	workspace, err := materializeWorkspace([]backend.WorkspaceFile{{Path: "src/main.go", Content: []byte("package main")}})
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(workspace)
	info, err := os.Stat(workspace + "/src/main.go")
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("workspace file mode=%v", info.Mode())
	}
}
