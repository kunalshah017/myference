package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/kunalshah017/myference/cli/internal/backend"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func New(baseURL string, httpClient *http.Client) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	host := parsed.Hostname()
	if parsed.Scheme != "http" || (host != "127.0.0.1" && host != "localhost" && host != "::1") {
		return nil, errors.New("Ollama URL must use loopback HTTP")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), http: httpClient}, nil
}

func (c *Client) Models(ctx context.Context) ([]backend.Model, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	response, err := c.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, statusError(response)
	}
	var payload struct {
		Models []backend.Model `json:"models"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 4<<20))
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}
	return payload.Models, nil
}

func (c *Client) Generate(ctx context.Context, input backend.Request, onContent func(string) error) (backend.Usage, error) {
	if input.Model == "" || input.Prompt == "" || onContent == nil || len(input.Workspace) != 0 {
		return backend.Usage{}, errors.New("model, prompt, and content callback are required")
	}
	maximumOutputTokens := input.MaximumOutputTokens
	if maximumOutputTokens == 0 {
		maximumOutputTokens = 4096
	}
	body, err := json.Marshal(map[string]any{
		"model":   input.Model,
		"prompt":  input.Prompt,
		"stream":  true,
		"options": map[string]any{"temperature": 0, "num_predict": maximumOutputTokens},
	})
	if err != nil {
		return backend.Usage{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return backend.Usage{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return backend.Usage{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return backend.Usage{}, statusError(response)
	}

	var usage backend.Usage
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	for scanner.Scan() {
		var chunk struct {
			Response        string `json:"response"`
			Done            bool   `json:"done"`
			PromptEvalCount uint64 `json:"prompt_eval_count"`
			EvalCount       uint64 `json:"eval_count"`
			TotalDuration   uint64 `json:"total_duration"`
			Error           string `json:"error"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			return backend.Usage{}, fmt.Errorf("decode Ollama stream: %w", err)
		}
		if chunk.Error != "" {
			return backend.Usage{}, errors.New(chunk.Error)
		}
		if chunk.Response != "" {
			if err := onContent(chunk.Response); err != nil {
				return backend.Usage{}, err
			}
		}
		if chunk.Done {
			usage = backend.Usage{InputTokens: chunk.PromptEvalCount, OutputTokens: chunk.EvalCount, ComputeMilliseconds: chunk.TotalDuration / 1_000_000}
			return usage, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return backend.Usage{}, err
	}
	return backend.Usage{}, io.ErrUnexpectedEOF
}

func statusError(response *http.Response) error {
	payload, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
	return fmt.Errorf("Ollama returned %s: %s", response.Status, strings.TrimSpace(string(payload)))
}
