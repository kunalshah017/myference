//go:build integration

package ollama

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kunalshah017/myference/cli/internal/backend"
)

func TestRealOllamaDiscoveryStreamingUsageAndCancellation(t *testing.T) {
	model := os.Getenv("MYFERENCE_TEST_OLLAMA_MODEL")
	if model == "" {
		t.Fatal("MYFERENCE_TEST_OLLAMA_MODEL is required")
	}
	client, err := New("http://127.0.0.1:11434", nil)
	if err != nil {
		t.Fatal(err)
	}
	models, err := client.Models(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, installed := range models {
		found = found || installed.Name == model
	}
	if !found {
		t.Fatalf("real installed model %q not discovered: %+v", model, models)
	}

	var output strings.Builder
	usage, err := client.Generate(context.Background(), backend.Request{
		Model:  model,
		Prompt: "Reply with exactly the single word OK.",
	}, func(content string) error {
		output.WriteString(content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if output.Len() == 0 || usage.OutputTokens == 0 || usage.ComputeMilliseconds == 0 {
		t.Fatalf("output=%q usage=%+v", output.String(), usage)
	}

	cancelCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err = client.Generate(cancelCtx, backend.Request{
		Model:  model,
		Prompt: "Count upward forever, emitting one number at a time.",
	}, func(string) error {
		cancel()
		return context.Canceled
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}
