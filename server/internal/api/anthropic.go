package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	v1 "github.com/kunalshah017/myference/protocol/v1"
)

type anthropicHandler struct{ inference http.Handler }
type scopeEndpointKey struct{}

func NewAnthropic(inference http.Handler) http.Handler {
	return &anthropicHandler{inference: inference}
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens uint64             `json:"max_tokens"`
	Stream    bool               `json:"stream"`
	System    string             `json:"system,omitempty"`
	Messages  []chatMessage      `json:"messages"`
	Workspace []v1.WorkspaceFile `json:"myference_workspace,omitempty"`
}

func (h *anthropicHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost || request.URL.Path != "/v1/messages" {
		http.NotFound(response, request)
		return
	}
	if request.Header.Get("anthropic-version") != "2023-06-01" {
		http.Error(response, "supported anthropic-version is required", http.StatusBadRequest)
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, 1<<20)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var input anthropicRequest
	if err := decoder.Decode(&input); err != nil || input.Model == "" || input.MaxTokens == 0 || !input.Stream || len(input.Messages) == 0 {
		http.Error(response, "invalid streaming messages request", http.StatusBadRequest)
		return
	}
	if extraErr := decoder.Decode(&struct{}{}); extraErr != io.EOF {
		http.Error(response, "invalid streaming messages request", http.StatusBadRequest)
		return
	}
	messages := input.Messages
	if input.System != "" {
		messages = append([]chatMessage{{Role: "system", Content: input.System}}, messages...)
	}
	encoded, err := json.Marshal(chatRequest{Model: input.Model, Stream: true, Messages: messages, Workspace: input.Workspace, MaxTokens: input.MaxTokens})
	if err != nil {
		http.Error(response, "request encoding failed", http.StatusInternalServerError)
		return
	}
	translated := request.Clone(context.WithValue(request.Context(), scopeEndpointKey{}, "/v1/messages"))
	translated.URL.Path = "/v1/chat/completions"
	translated.Body = io.NopCloser(bytes.NewReader(encoded))
	translated.ContentLength = int64(len(encoded))
	if key := strings.TrimSpace(request.Header.Get("x-api-key")); key != "" {
		translated.Header.Set("Authorization", "Bearer "+key)
	}
	h.inference.ServeHTTP(&anthropicResponseWriter{target: response, model: input.Model}, translated)
}

type anthropicResponseWriter struct {
	target           http.ResponseWriter
	buffer           strings.Builder
	status           int
	started, stopped bool
	model            string
}

func (w *anthropicResponseWriter) Header() http.Header { return w.target.Header() }
func (w *anthropicResponseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	if status >= 400 {
		w.target.WriteHeader(status)
		return
	}
	w.target.Header().Set("Content-Type", "text/event-stream")
	w.target.Header().Set("Cache-Control", "no-cache")
	w.target.WriteHeader(status)
	w.started = true
	id := w.target.Header().Get("X-Request-ID")
	w.event("message_start", map[string]any{"type": "message_start", "message": map[string]any{"id": id, "type": "message", "role": "assistant", "content": []any{}, "model": w.model, "stop_reason": nil, "usage": map[string]uint64{"input_tokens": 0, "output_tokens": 0}}})
	w.event("content_block_start", map[string]any{"type": "content_block_start", "index": 0, "content_block": map[string]string{"type": "text", "text": ""}})
}
func (w *anthropicResponseWriter) Write(payload []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if w.status >= 400 {
		return w.target.Write(payload)
	}
	w.buffer.Write(payload)
	w.process()
	return len(payload), nil
}
func (w *anthropicResponseWriter) Flush() {
	w.process()
	if flusher, ok := w.target.(http.Flusher); ok {
		flusher.Flush()
	}
}
func (w *anthropicResponseWriter) process() {
	for {
		raw := w.buffer.String()
		index := strings.Index(raw, "\n\n")
		if index < 0 {
			return
		}
		frame := raw[:index]
		rest := raw[index+2:]
		w.buffer.Reset()
		w.buffer.WriteString(rest)
		data := ""
		for _, line := range strings.Split(frame, "\n") {
			if strings.HasPrefix(line, "data: ") {
				data = strings.TrimPrefix(line, "data: ")
			}
		}
		if data == "" {
			continue
		}
		if data == "[DONE]" {
			if !w.stopped {
				w.event("message_stop", map[string]string{"type": "message_stop"})
				w.stopped = true
			}
			continue
		}
		var chunk struct {
			Error *struct {
				Type    string `json:"type"`
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Usage struct {
				PromptTokens     uint64 `json:"prompt_tokens"`
				CompletionTokens uint64 `json:"completion_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal([]byte(data), &chunk) != nil {
			continue
		}
		if chunk.Error != nil {
			w.event("error", map[string]any{"type": "error", "error": chunk.Error})
			w.stopped = true
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		if chunk.Choices[0].Delta.Content != "" {
			w.event("content_block_delta", map[string]any{"type": "content_block_delta", "index": 0, "delta": map[string]string{"type": "text_delta", "text": chunk.Choices[0].Delta.Content}})
		}
		if chunk.Choices[0].FinishReason != nil {
			w.event("content_block_stop", map[string]any{"type": "content_block_stop", "index": 0})
			w.event("message_delta", map[string]any{"type": "message_delta", "delta": map[string]string{"stop_reason": "end_turn"}, "usage": map[string]uint64{"output_tokens": chunk.Usage.CompletionTokens}})
		}
	}
}
func (w *anthropicResponseWriter) event(name string, value any) {
	payload, _ := json.Marshal(value)
	_, _ = fmt.Fprintf(w.target, "event: %s\ndata: %s\n\n", name, payload)
}

func authorizedEndpoint(ctx context.Context, fallback string) string {
	if endpoint, ok := ctx.Value(scopeEndpointKey{}).(string); ok {
		return endpoint
	}
	return fallback
}
