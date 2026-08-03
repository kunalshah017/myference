package windows

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseOllamaPSModels(t *testing.T) {
	input := `NAME ID SIZE PROCESSOR UNTIL
qwen2.5:7b abc 5.2 GB 100% GPU 4 minutes from now
llama3:latest def 4.7 GB 40%/60% CPU/GPU Forever
`
	got, err := ParseOllamaPS(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	want := []LoadedModel{{Name: "qwen2.5:7b", ID: "abc"}, {Name: "llama3:latest", ID: "def"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseOllamaPS() = %+v, want %+v", got, want)
	}
}

func TestSelectInstalledModel(t *testing.T) {
	models := []string{"zeta:latest", "alpha:latest"}
	if got, err := SelectInstalledModel(models, ""); err != nil || got != "alpha:latest" {
		t.Fatalf("automatic selection = %q, %v", got, err)
	}
	if _, err := SelectInstalledModel(models, "missing:latest"); err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("missing explicit model error = %v", err)
	}
}

func TestOllamaPreloadSendsExactRequest(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/api/generate" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		if err := json.NewDecoder(request.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"done":true}`))
	}))
	defer server.Close()

	client, err := NewOllamaHostClient(server.URL, &http.Client{Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	config := DefaultConfig()
	config.KeepAlive = "15m"
	config.ContextLength = 8192
	if err := client.Preload(context.Background(), "qwen2.5:7b", config); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"model":      "qwen2.5:7b",
		"prompt":     "",
		"stream":     false,
		"keep_alive": "15m",
		"options":    map[string]any{"num_ctx": float64(8192)},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("preload request = %#v, want %#v", got, want)
	}
}

func TestOllamaHostClientRejectsNonLoopback(t *testing.T) {
	if _, err := NewOllamaHostClient("http://192.168.1.5:11434", nil); err == nil {
		t.Fatal("non-loopback Ollama endpoint accepted")
	}
}

func TestOllamaGenerateTestUsesNonStreamingLoopbackRequest(t *testing.T) {
	var stream any
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		stream = payload["stream"]
		_, _ = response.Write([]byte(`{"response":"myference works","done":true}`))
	}))
	defer server.Close()
	client, err := NewOllamaHostClient(server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.GenerateTest(context.Background(), "qwen:latest")
	if err != nil || got != "myference works" || stream != false {
		t.Fatalf("GenerateTest() = %q, %v, stream=%v", got, err, stream)
	}
}
