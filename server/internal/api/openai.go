package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	v1 "github.com/kunalshah017/myference/protocol/v1"
	"github.com/kunalshah017/myference/server/internal/relay"
	"github.com/kunalshah017/myference/server/internal/router"
)

type Principal struct {
	AccountID      string
	SessionID      string
	SessionBalance uint64
}

type Proposal struct {
	RequestID, SessionID, MachineID, OfferID, Model string
	PriceVersion                                    uint64
	InputTokens, OutputTokens, ComputeMilliseconds  uint64
	InputHash, OutputHash                           [32]byte
	CompletedAt                                     time.Time
}

type Dependencies struct {
	Hub        *relay.Hub
	Authorize  func(context.Context, string, string, string, uint64) (Principal, error)
	Candidates func(context.Context, string) ([]router.Candidate, error)
	Persist    func(context.Context, Proposal) error
	Now        func() time.Time
}

type OpenAI struct {
	dependencies Dependencies
	broker       *eventBroker
}

func NewOpenAI(dependencies Dependencies) http.Handler {
	if dependencies.Now == nil {
		dependencies.Now = time.Now
	}
	handler := &OpenAI{dependencies: dependencies, broker: newEventBroker(dependencies.Hub)}
	return handler
}

type chatRequest struct {
	Model    string        `json:"model"`
	Stream   bool          `json:"stream"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (h *OpenAI) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost || request.URL.Path != "/v1/chat/completions" {
		http.NotFound(response, request)
		return
	}
	if h.dependencies.Hub == nil || h.dependencies.Authorize == nil || h.dependencies.Candidates == nil || h.dependencies.Persist == nil {
		http.Error(response, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	request.Body = http.MaxBytesReader(response, request.Body, 1<<20)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var input chatRequest
	if err := decoder.Decode(&input); err != nil || !input.Stream || strings.TrimSpace(input.Model) == "" || len(input.Messages) == 0 {
		http.Error(response, "invalid streaming chat request", http.StatusBadRequest)
		return
	}
	maximumSpend, err := strconv.ParseUint(request.Header.Get("X-Myference-Max-Spend"), 10, 64)
	if err != nil || maximumSpend == 0 {
		http.Error(response, "valid X-Myference-Max-Spend is required", http.StatusBadRequest)
		return
	}
	token := strings.TrimSpace(strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer "))
	principal, err := h.dependencies.Authorize(request.Context(), token, input.Model, request.URL.Path, maximumSpend)
	if err != nil {
		http.Error(response, "unauthorized", http.StatusUnauthorized)
		return
	}
	candidates, err := h.dependencies.Candidates(request.Context(), input.Model)
	if err != nil {
		http.Error(response, "routing unavailable", http.StatusServiceUnavailable)
		return
	}
	selected, err := router.Select(router.Request{Model: input.Model, Capabilities: []string{"text", "stream"}, MaximumSpend: maximumSpend, SessionBalance: principal.SessionBalance, PinMachineID: request.Header.Get("X-Myference-Machine")}, candidates)
	if err != nil || !h.dependencies.Hub.Connected(selected.MachineID) {
		http.Error(response, "no eligible provider", http.StatusServiceUnavailable)
		return
	}
	requestID, err := randomID()
	if err != nil {
		http.Error(response, "request creation failed", http.StatusInternalServerError)
		return
	}
	events := h.broker.register(requestID)
	defer h.broker.unregister(requestID)
	prompt := renderPrompt(input.Messages)
	envelope, err := v1.NewEnvelope("offer-"+requestID, v1.MessageJobOffer, &v1.JobOffer{RequestID: requestID, Model: input.Model, OfferID: selected.OfferID, Prompt: prompt, PriceVersion: selected.PriceVersion, MaximumSpend: maximumSpend, LeaseExpiresAt: h.dependencies.Now().Add(30 * time.Second)})
	if err != nil || h.dependencies.Hub.Send(selected.MachineID, envelope) != nil {
		http.Error(response, "provider unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := waitAccepted(request.Context(), events); err != nil {
		http.Error(response, "provider did not accept", http.StatusGatewayTimeout)
		return
	}
	flusher, ok := response.(http.Flusher)
	if !ok {
		http.Error(response, "streaming unavailable", http.StatusInternalServerError)
		return
	}
	response.Header().Set("Content-Type", "text/event-stream")
	response.Header().Set("Cache-Control", "no-cache")
	response.Header().Set("X-Request-ID", requestID)
	response.WriteHeader(http.StatusOK)
	var output strings.Builder
	for {
		select {
		case <-request.Context().Done():
			h.cancel(selected.MachineID, requestID)
			return
		case event := <-events:
			if event.Type != v1.MessageOutputChunk {
				continue
			}
			var chunk v1.OutputChunk
			if event.Envelope.DecodeBody(&chunk) != nil {
				return
			}
			if chunk.Data != "" {
				output.WriteString(chunk.Data)
				if writeSSE(response, map[string]any{"id": requestID, "object": "chat.completion.chunk", "choices": []any{map[string]any{"index": 0, "delta": map[string]string{"content": chunk.Data}}}}) != nil {
					h.cancel(selected.MachineID, requestID)
					return
				}
				flusher.Flush()
			}
			if chunk.Done {
				proposal := Proposal{RequestID: requestID, SessionID: principal.SessionID, MachineID: selected.MachineID, OfferID: selected.OfferID, Model: input.Model, PriceVersion: selected.PriceVersion, InputTokens: chunk.InputTokens, OutputTokens: chunk.OutputTokens, ComputeMilliseconds: chunk.ComputeMilliseconds, InputHash: saltedHash(prompt), OutputHash: saltedHash(output.String()), CompletedAt: h.dependencies.Now()}
				if h.dependencies.Persist(request.Context(), proposal) != nil {
					return
				}
				_ = writeSSE(response, map[string]any{"id": requestID, "object": "chat.completion.chunk", "choices": []any{map[string]any{"index": 0, "delta": map[string]string{}, "finish_reason": "stop"}}})
				fmt.Fprint(response, "data: [DONE]\n\n")
				flusher.Flush()
				return
			}
		}
	}
}

func waitAccepted(ctx context.Context, events <-chan relay.Event) error {
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return errors.New("accept timeout")
		case event := <-events:
			if event.Type == v1.MessageJobAccept {
				return nil
			}
		}
	}
}

func (h *OpenAI) cancel(machineID, requestID string) {
	envelope, err := v1.NewEnvelope("cancel-"+requestID, v1.MessageCancel, &v1.Cancel{RequestID: requestID, Reason: "customer_cancelled"})
	if err == nil {
		_ = h.dependencies.Hub.Send(machineID, envelope)
	}
}

func renderPrompt(messages []chatMessage) string {
	var result strings.Builder
	for _, message := range messages {
		fmt.Fprintf(&result, "%s: %s\n", message.Role, message.Content)
	}
	return result.String()
}

func writeSSE(writer http.ResponseWriter, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(writer, "data: %s\n\n", payload)
	return err
}

func randomID() (string, error) {
	value := make([]byte, 18)
	_, err := rand.Read(value)
	return "req_" + base64.RawURLEncoding.EncodeToString(value), err
}

func saltedHash(value string) [32]byte {
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return sha256.Sum256([]byte(value))
	}
	return sha256.Sum256(append(salt, []byte(value)...))
}

type eventBroker struct {
	hub     *relay.Hub
	mu      sync.RWMutex
	streams map[string]chan relay.Event
}

func newEventBroker(hub *relay.Hub) *eventBroker {
	broker := &eventBroker{hub: hub, streams: make(map[string]chan relay.Event)}
	if hub != nil {
		go broker.run()
	}
	return broker
}

func (b *eventBroker) register(requestID string) <-chan relay.Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	stream := make(chan relay.Event, 32)
	b.streams[requestID] = stream
	return stream
}

func (b *eventBroker) unregister(requestID string) {
	b.mu.Lock()
	delete(b.streams, requestID)
	b.mu.Unlock()
}

func (b *eventBroker) run() {
	for event := range b.hub.Events() {
		requestID := eventRequestID(event)
		if requestID == "" {
			continue
		}
		b.mu.RLock()
		stream := b.streams[requestID]
		b.mu.RUnlock()
		if stream != nil {
			select {
			case stream <- event:
			default:
			}
		}
	}
}

func eventRequestID(event relay.Event) string {
	switch event.Type {
	case v1.MessageJobAccept:
		var message v1.JobAccept
		if event.Envelope.DecodeBody(&message) == nil {
			return message.RequestID
		}
	case v1.MessageOutputChunk:
		var message v1.OutputChunk
		if event.Envelope.DecodeBody(&message) == nil {
			return message.RequestID
		}
	}
	return ""
}
