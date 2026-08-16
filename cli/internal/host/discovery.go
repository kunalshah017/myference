package host

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/kunalshah017/myference/cli/internal/backend"
	"github.com/kunalshah017/myference/cli/internal/backend/ollama"
	openaiBackend "github.com/kunalshah017/myference/cli/internal/backend/openai"
)

const (
	StateReady = "ready"
)

var ErrModelCatalogUnavailable = errors.New("provider model catalog unavailable")

type Candidate struct {
	ID       string
	Kind     string
	Name     string
	URL      string
	Model    string
	Digest   string
	Image    string
	State    string
	Hint     string
	Size     int64
	Selected bool
}

type Result struct {
	Source     string
	Candidates []Candidate
	Err        error
}

type Detector interface {
	Detect(context.Context) Result
}

type ModelSource interface {
	Models(context.Context) ([]backend.Model, error)
}

func Discover(ctx context.Context, detectors []Detector) <-chan Result {
	output := make(chan Result)
	var wait sync.WaitGroup
	for _, detector := range detectors {
		wait.Add(1)
		go func(current Detector) {
			defer wait.Done()
			select {
			case output <- current.Detect(ctx):
			case <-ctx.Done():
			}
		}(detector)
	}
	go func() {
		wait.Wait()
		close(output)
	}()
	return output
}

type OllamaDetector struct {
	Endpoint string
	Client   *http.Client
	New      func(string, *http.Client) (ModelSource, error)
}

func (detector OllamaDetector) Detect(parent context.Context) Result {
	endpoint := detector.Endpoint
	if endpoint == "" {
		endpoint = "http://127.0.0.1:11434"
	}
	newClient := detector.New
	if newClient == nil {
		newClient = func(endpoint string, client *http.Client) (ModelSource, error) {
			return ollama.New(endpoint, client)
		}
	}
	client, err := newClient(endpoint, detector.Client)
	if err != nil {
		return Result{Source: "ollama", Err: err}
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	models, err := client.Models(ctx)
	if err != nil {
		return Result{Source: "ollama", Err: fmt.Errorf("Ollama is installed but not running: %w", err)}
	}
	candidates := make([]Candidate, 0, len(models))
	for _, model := range models {
		if strings.TrimSpace(model.Name) == "" {
			continue
		}
		candidate := Candidate{Kind: "ollama", Name: "Ollama", URL: endpoint, Model: model.Name, Digest: model.Digest, Size: model.Size, State: StateReady}
		candidate.ID = StableID(candidate)
		candidates = append(candidates, candidate)
	}
	slices.SortFunc(candidates, func(a, b Candidate) int { return strings.Compare(a.Model, b.Model) })
	if len(candidates) == 0 {
		return Result{Source: "ollama", Err: errors.New("Ollama has no installed models; run `ollama pull <model>`")}
	}
	return Result{Source: "ollama", Candidates: candidates}
}

func DefaultDetectors(endpoint string, client *http.Client) []Detector {
	return []Detector{
		OllamaDetector{Endpoint: endpoint, Client: client},
	}
}

func ListAPIModels(ctx context.Context, baseURL, secret string, client *http.Client) ([]string, error) {
	configured, err := openaiBackend.New(baseURL, secret, client)
	if err != nil {
		return nil, err
	}
	models, err := configured.Models(ctx)
	if err != nil {
		return nil, ErrModelCatalogUnavailable
	}
	seen := make(map[string]bool, len(models))
	result := make([]string, 0, len(models))
	for _, model := range models {
		name := strings.TrimSpace(model.Name)
		if name != "" && !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}
	slices.Sort(result)
	if len(result) == 0 {
		return nil, ErrModelCatalogUnavailable
	}
	return result, nil
}

func StableID(candidate Candidate) string {
	return strings.Join([]string{strings.ToLower(strings.TrimSpace(candidate.Kind)), strings.TrimRight(strings.ToLower(strings.TrimSpace(candidate.URL)), "/"), strings.TrimSpace(candidate.Model)}, "|")
}
