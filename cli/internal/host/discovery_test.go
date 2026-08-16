package host

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/kunalshah017/myference/cli/internal/backend"
)

type detectorFunc func(context.Context) Result

func (fn detectorFunc) Detect(ctx context.Context) Result { return fn(ctx) }

func TestDiscoverKeepsIndependentResults(t *testing.T) {
	results := Discover(context.Background(), []Detector{
		detectorFunc(func(context.Context) Result {
			return Result{Source: "ollama", Candidates: []Candidate{{Kind: "ollama", Model: "qwen"}}}
		}),
		detectorFunc(func(context.Context) Result {
			return Result{Source: "openai", Err: errors.New("catalog unavailable")}
		}),
	})
	var got []Result
	for result := range results {
		got = append(got, result)
	}
	if len(got) != 2 {
		t.Fatalf("results=%+v", got)
	}
	slices.SortFunc(got, func(a, b Result) int { return strings.Compare(a.Source, b.Source) })
	if got[0].Source != "ollama" || got[1].Err == nil || len(got[0].Candidates) != 1 {
		t.Fatalf("results=%+v", got)
	}
}

type modelSource struct {
	models []backend.Model
	err    error
}

func (source modelSource) Models(context.Context) ([]backend.Model, error) {
	return source.models, source.err
}

func TestOllamaDetectorReturnsEveryInstalledModel(t *testing.T) {
	detector := OllamaDetector{Endpoint: "http://127.0.0.1:11434", New: func(string, *http.Client) (ModelSource, error) {
		return modelSource{models: []backend.Model{{Name: "qwen", Digest: "sha256:q", Size: 42}, {Name: "llama", Digest: "sha256:l", Size: 84}}}, nil
	}}
	result := detector.Detect(context.Background())
	if result.Err != nil || len(result.Candidates) != 2 || result.Candidates[0].Digest == "" || result.Candidates[1].Size == 0 {
		t.Fatalf("result=%+v", result)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

func TestListAPIModelsSortsAndDeduplicates(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != "https://provider.example/v1/models" || request.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("request=%s authorization=%q", request.URL, request.Header.Get("Authorization"))
		}
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: ioNopCloser(`{"data":[{"id":"zeta"},{"id":"alpha"},{"id":"alpha"}]}`)}, nil
	})}
	models, err := ListAPIModels(context.Background(), "https://provider.example", "secret", client)
	if err != nil || !reflect.DeepEqual(models, []string{"alpha", "zeta"}) {
		t.Fatalf("models=%v err=%v", models, err)
	}
}

type stringCloser struct{ *strings.Reader }

func (stringCloser) Close() error           { return nil }
func ioNopCloser(value string) stringCloser { return stringCloser{strings.NewReader(value)} }
