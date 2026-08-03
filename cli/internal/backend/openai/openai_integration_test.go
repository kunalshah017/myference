package openai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kunalshah017/myference/cli/internal/backend"
)

func TestClientUsesCompletionTokenLimitField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if _, present := payload["max_tokens"]; present {
			http.Error(w, "legacy max_tokens field is not supported", http.StatusBadRequest)
			return
		}
		if got := payload["max_completion_tokens"]; got != float64(17) {
			http.Error(w, fmt.Sprintf("max_completion_tokens=%v", got), http.StatusBadRequest)
			return
		}
		w.Header().Set("content-type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}],\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":1}}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()

	client, err := New(server.URL, "provider-secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Generate(t.Context(), backend.Request{Model: "remote-model", Prompt: "hello", MaximumOutputTokens: 17}, func(string) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
}

func TestClientUsesLocalSecretAndStreamsRealCompatibleEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer provider-secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/models":
			w.Header().Set("content-type", "application/json")
			fmt.Fprint(w, `{"data":[{"id":"remote-model"}]}`)
		case "/v1/chat/completions":
			w.Header().Set("content-type", "text/event-stream")
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hel\"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":1}}\n\ndata: [DONE]\n\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client, err := New(server.URL, "provider-secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	models, err := client.Models(t.Context())
	if err != nil || len(models) != 1 || models[0].Name != "remote-model" {
		t.Fatalf("models=%+v err=%v", models, err)
	}
	var output strings.Builder
	usage, err := client.Generate(t.Context(), backend.Request{Model: "remote-model", Prompt: "say hello"}, func(chunk string) error { output.WriteString(chunk); return nil })
	if err != nil || output.String() != "hello" || usage.InputTokens != 4 || usage.OutputTokens != 1 {
		t.Fatalf("output=%q usage=%+v err=%v", output.String(), usage, err)
	}
}

func TestClientRejectsRemotePlaintextEndpointAndEmptySecret(t *testing.T) {
	if _, err := New("http://example.com", "secret", nil); err == nil {
		t.Fatal("accepted remote plaintext")
	}
	if _, err := New("https://example.com", "", nil); err == nil {
		t.Fatal("accepted empty secret")
	}
}

func TestClientRejectsStreamedOutputWithoutProviderUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("content-type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"unmetered\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()
	client, err := New(server.URL, "provider-secret", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Generate(t.Context(), backend.Request{Model: "remote-model", Prompt: "hello", MaximumOutputTokens: 8}, func(string) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "usage") {
		t.Fatalf("expected missing usage error, got %v", err)
	}
}
