package openai

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
	"time"

	"github.com/kunalshah017/myference/cli/internal/backend"
)

type Client struct {
	baseURL, secret string
	http            *http.Client
}

func New(baseURL, secret string, httpClient *http.Client) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" {
		return nil, errors.New("invalid OpenAI-compatible URL")
	}
	host := parsed.Hostname()
	loopback := host == "127.0.0.1" || host == "localhost" || host == "::1"
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && loopback) {
		return nil, errors.New("OpenAI-compatible URL must use HTTPS except on loopback")
	}
	if strings.TrimSpace(secret) == "" {
		return nil, errors.New("provider API secret is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 2 * time.Minute}
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), secret: secret, http: httpClient}, nil
}

func (c *Client) Models(ctx context.Context) ([]backend.Model, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/models", nil)
	if err != nil {
		return nil, err
	}
	c.authorize(request)
	response, err := c.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, statusError(response)
	}
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&payload); err != nil {
		return nil, err
	}
	models := make([]backend.Model, 0, len(payload.Data))
	for _, item := range payload.Data {
		if item.ID != "" {
			models = append(models, backend.Model{Name: item.ID})
		}
	}
	return models, nil
}

func (c *Client) Generate(ctx context.Context, input backend.Request, emit func(string) error) (backend.Usage, error) {
	if input.Model == "" || input.Prompt == "" || emit == nil || len(input.Workspace) != 0 {
		return backend.Usage{}, errors.New("model, prompt, and callback are required")
	}
	maximumOutputTokens := input.MaximumOutputTokens
	if maximumOutputTokens == 0 {
		maximumOutputTokens = 4096
	}
	body, _ := json.Marshal(map[string]any{"model": input.Model, "stream": true, "messages": []map[string]string{{"role": "user", "content": input.Prompt}}, "stream_options": map[string]bool{"include_usage": true}, "max_completion_tokens": maximumOutputTokens})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return backend.Usage{}, err
	}
	request.Header.Set("content-type", "application/json")
	c.authorize(request)
	started := time.Now()
	response, err := c.http.Do(request)
	if err != nil {
		return backend.Usage{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return backend.Usage{}, statusError(response)
	}
	var usage backend.Usage
	done := false
	outputSeen := false
	usageSeen := false
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			done = true
			break
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				Prompt     uint64 `json:"prompt_tokens"`
				Completion uint64 `json:"completion_tokens"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return backend.Usage{}, fmt.Errorf("decode OpenAI-compatible stream: %w", err)
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				outputSeen = true
				if err := emit(choice.Delta.Content); err != nil {
					return backend.Usage{}, err
				}
			}
		}
		if chunk.Usage != nil {
			usageSeen = true
			usage.InputTokens = chunk.Usage.Prompt
			usage.OutputTokens = chunk.Usage.Completion
		}
	}
	if err := scanner.Err(); err != nil {
		return backend.Usage{}, err
	}
	if !done {
		return backend.Usage{}, io.ErrUnexpectedEOF
	}
	if outputSeen && !usageSeen {
		return backend.Usage{}, errors.New("OpenAI-compatible provider omitted usage for streamed output")
	}
	if outputSeen && usage.OutputTokens == 0 {
		return backend.Usage{}, errors.New("OpenAI-compatible provider reported zero usage for streamed output")
	}
	if usage.OutputTokens > maximumOutputTokens {
		return backend.Usage{}, errors.New("OpenAI-compatible provider reported usage above the output token limit")
	}
	usage.ComputeMilliseconds = uint64(time.Since(started).Milliseconds())
	return usage, nil
}

func (c *Client) authorize(request *http.Request) {
	request.Header.Set("authorization", "Bearer "+c.secret)
}
func statusError(response *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
	return fmt.Errorf("OpenAI-compatible provider returned %s: %s", response.Status, strings.TrimSpace(string(raw)))
}
