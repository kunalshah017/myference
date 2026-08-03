//go:build windows

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kunalshah017/myference/cli/internal/config"
	platform "github.com/kunalshah017/myference/cli/internal/platform/windows"
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
	var gotModel string
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/tags":
			_, _ = response.Write([]byte(`{"models":[{"name":"qwen:latest"}]}`))
		case "/api/generate":
			var payload struct {
				Model string `json:"model"`
			}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			gotModel = payload.Model
			_, _ = response.Write([]byte(`{"done":true}`))
		}
	}))
	defer server.Close()
	cfg := config.Config{Backends: []config.Backend{{Name: "local", Kind: "ollama", URL: server.URL, Model: "qwen:latest", Enabled: true}}}
	if err := prepareWindowsBackends(context.Background(), cfg, platform.DefaultConfig(), server.Client()); err != nil {
		t.Fatal(err)
	}
	if gotModel != "qwen:latest" {
		t.Fatalf("preloaded model = %q", gotModel)
	}
}
