package windows

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
)

const defaultOllamaTimeout = 5 * time.Minute

type LoadedModel struct {
	Name string
	ID   string
}

type OllamaHostClient struct {
	baseURL string
	http    *http.Client
}

func NewOllamaHostClient(baseURL string, httpClient *http.Client) (*OllamaHostClient, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	host := parsed.Hostname()
	if parsed.Scheme != "http" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || !isLoopbackHost(host) {
		return nil, errors.New("Ollama URL must use loopback HTTP")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultOllamaTimeout}
	} else if httpClient.Timeout <= 0 {
		clone := *httpClient
		clone.Timeout = defaultOllamaTimeout
		httpClient = &clone
	}
	return &OllamaHostClient{baseURL: strings.TrimRight(baseURL, "/"), http: httpClient}, nil
}

func (client *OllamaHostClient) InstalledModels(ctx context.Context) ([]string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, client.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	response, err := client.http.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, ollamaStatusError(response)
	}
	var payload struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&payload); err != nil {
		return nil, err
	}
	models := make([]string, 0, len(payload.Models))
	for _, model := range payload.Models {
		if name := strings.TrimSpace(model.Name); name != "" {
			models = append(models, name)
		}
	}
	slices.Sort(models)
	return models, nil
}

func (client *OllamaHostClient) Preload(ctx context.Context, model string, config Config) error {
	if strings.TrimSpace(model) == "" {
		return errors.New("preload model is required")
	}
	if err := config.Validate(); err != nil {
		return err
	}
	payload := struct {
		Model     string         `json:"model"`
		Prompt    string         `json:"prompt"`
		Stream    bool           `json:"stream"`
		KeepAlive string         `json:"keep_alive"`
		Options   map[string]int `json:"options"`
	}{Model: model, Prompt: "", Stream: false, KeepAlive: config.KeepAlive, Options: map[string]int{"num_ctx": config.ContextLength}}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ollamaStatusError(response)
	}
	var result struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return err
	}
	if result.Error != "" {
		return errors.New(result.Error)
	}
	return nil
}

func (client *OllamaHostClient) GenerateTest(ctx context.Context, model string) (string, error) {
	if strings.TrimSpace(model) == "" {
		return "", errors.New("test model is required")
	}
	payload := struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
		Stream bool   `json:"stream"`
	}{Model: model, Prompt: "Reply with only: myference works", Stream: false}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.http.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", ollamaStatusError(response)
	}
	var result struct {
		Response string `json:"response"`
		Error    string `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&result); err != nil {
		return "", err
	}
	if result.Error != "" {
		return "", errors.New(result.Error)
	}
	return strings.TrimSpace(result.Response), nil
}

func ParseOllamaPS(input io.Reader) ([]LoadedModel, error) {
	scanner := bufio.NewScanner(input)
	models := make([]LoadedModel, 0)
	line := 0
	for scanner.Scan() {
		line++
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 || strings.EqualFold(fields[0], "NAME") {
			continue
		}
		if len(fields) < 2 {
			return nil, fmt.Errorf("parse ollama ps line %d: expected name and ID", line)
		}
		models = append(models, LoadedModel{Name: fields[0], ID: fields[1]})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return models, nil
}

func SelectInstalledModel(installed []string, requested string) (string, error) {
	models := append([]string(nil), installed...)
	slices.Sort(models)
	if requested != "" {
		if slices.Contains(models, requested) {
			return requested, nil
		}
		return "", fmt.Errorf("model %q is not installed in Ollama", requested)
	}
	if len(models) == 0 {
		return "", errors.New("Ollama has no installed models; install one with `ollama pull <model>`")
	}
	return models[0], nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	address := net.ParseIP(host)
	return address != nil && address.IsLoopback()
}

func ollamaStatusError(response *http.Response) error {
	payload, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
	return fmt.Errorf("Ollama returned %s: %s", response.Status, strings.TrimSpace(string(payload)))
}
