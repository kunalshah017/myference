//go:build windows

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/kunalshah017/myference/cli/internal/backend"
	"github.com/kunalshah017/myference/cli/internal/config"
	platform "github.com/kunalshah017/myference/cli/internal/platform/windows"
	"github.com/kunalshah017/myference/cli/internal/provider"
	v1 "github.com/kunalshah017/myference/protocol/v1"
)

func TestWindowsModelsAndTestUseLoopbackOllama(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/tags":
			_, _ = response.Write([]byte(`{"models":[{"name":"qwen:latest"}]}`))
		case "/api/generate":
			_, _ = response.Write([]byte(`{"response":"myference works","done":true}`))
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	var output bytes.Buffer
	if err := run([]string{"windows", "models", "--ollama-url", server.URL}, &output); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); !strings.Contains(got, "qwen:latest") {
		t.Fatalf("models output = %q", got)
	}
	output.Reset()
	if err := run([]string{"windows", "test", "--ollama-url", server.URL, "--model", "qwen:latest"}, &output); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); !strings.Contains(got, "myference works") {
		t.Fatalf("test output = %q", got)
	}
}

func TestPrepareWindowsBackendsFailsBeforeReadinessWhenPreloadFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/tags":
			_, _ = response.Write([]byte(`{"models":[{"name":"qwen:latest"}]}`))
		case "/api/generate":
			http.Error(response, "load failed", http.StatusInternalServerError)
		}
	}))
	defer server.Close()
	cfg := config.Config{Backends: []config.Backend{{Name: "local", Kind: "ollama", URL: server.URL, Model: "qwen:latest", Enabled: true}}}
	err := prepareWindowsBackends(context.Background(), cfg, platform.DefaultConfig(), server.Client())
	if err == nil || !strings.Contains(err.Error(), "preload") {
		t.Fatalf("prepareWindowsBackends() error = %v", err)
	}
}

func TestPrepareWindowsBackendsPreloadsConfiguredModel(t *testing.T) {
	var gotModels []string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/tags":
			_, _ = response.Write([]byte(`{"models":[{"name":"qwen:latest"},{"name":"llama:latest"}]}`))
		case "/api/generate":
			var payload struct {
				Model string `json:"model"`
			}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			gotModels = append(gotModels, payload.Model)
			_, _ = response.Write([]byte(`{"done":true}`))
		}
	}))
	defer server.Close()
	cfg := config.Config{Backends: []config.Backend{
		{Name: "qwen", Kind: "ollama", URL: server.URL, Model: "qwen:latest", Enabled: true},
		{Name: "llama", Kind: "ollama", URL: server.URL, Model: "llama:latest", Enabled: true},
	}}
	if err := prepareWindowsBackends(context.Background(), cfg, platform.DefaultConfig(), server.Client()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotModels, []string{"qwen:latest", "llama:latest"}) {
		t.Fatalf("preloaded models = %v", gotModels)
	}
}

func TestPrepareWindowsBackendsSkipsStoppedBackend(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer server.Close()
	cfg := config.Config{Backends: []config.Backend{{Name: "stopped", Kind: "ollama", URL: server.URL, Model: "qwen", Enabled: false}}}
	if err := prepareWindowsBackends(context.Background(), cfg, platform.DefaultConfig(), server.Client()); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("stopped backend received %d requests", calls)
	}
}

func TestReloadBackendsRetainsLastGoodCapacityWhenPreparationFails(t *testing.T) {
	oldOffer := v1.OfferCapacity{OfferID: "old", Model: "old-model", PriceVersion: 1, BackendKind: "ollama", OfferHash: "0x" + strings.Repeat("1", 64), ModelHash: "0x" + strings.Repeat("2", 64), CapabilityHash: "0x" + strings.Repeat("3", 64), Capabilities: []string{"stream", "text"}, EvidenceKind: "ollama_digest", EvidenceDigest: "sha256:old", MeteringMode: "tokens_and_compute"}
	daemon := provider.NewDaemon(provider.Config{Offers: []v1.OfferCapacity{oldOffer}}, map[string]backend.Backend{"old": windowsTestBackend{}})
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/api/tags" {
			_, _ = response.Write([]byte(`{"models":[{"name":"new-model"}]}`))
			return
		}
		http.Error(response, "preload failed", http.StatusInternalServerError)
	}))
	defer server.Close()
	cfg := config.Config{MachineID: "machine", Backends: []config.Backend{{Name: "new", Kind: "ollama", URL: server.URL, Model: "new-model", Enabled: true, PriceVersion: 1}}}
	if err := reloadBackends(context.Background(), cfg, daemon); err == nil {
		t.Fatal("reload succeeded despite preload failure")
	}
	got := daemon.Capacity()
	if len(got.Offers) != 1 || got.Offers[0].OfferID != "old" {
		t.Fatalf("capacity changed after failed reload: %+v", got)
	}
}

type windowsTestBackend struct{}

func (windowsTestBackend) Models(context.Context) ([]backend.Model, error) {
	return []backend.Model{{Name: "old-model", Digest: "sha256:old"}}, nil
}
func (windowsTestBackend) Generate(context.Context, backend.Request, func(string) error) (backend.Usage, error) {
	return backend.Usage{}, nil
}
