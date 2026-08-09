package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
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

type Reservation struct {
	RequestID, SessionID, AccountID, MachineID, OfferID string
	PriceVersion, MaximumSpend                          uint64
	MaximumInputTokens, MaximumOutputTokens             uint64
	MaximumComputeMilliseconds                          uint64
}

type Dependencies struct {
	Hub        *relay.Hub
	Authorize  func(context.Context, string, string, string, uint64) (Principal, error)
	Candidates func(context.Context, string) ([]router.Candidate, error)
	Reserve    func(context.Context, Reservation) error
	Transition func(context.Context, string, string) error
	Abort      func(context.Context, string, string) error
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
	Model               string             `json:"model"`
	Stream              bool               `json:"stream"`
	Messages            []chatMessage      `json:"messages"`
	Workspace           []v1.WorkspaceFile `json:"myference_workspace,omitempty"`
	MaxTokens           uint64             `json:"max_tokens,omitempty"`
	MaxCompletionTokens uint64             `json:"max_completion_tokens,omitempty"`
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
	if h.dependencies.Hub == nil || h.dependencies.Authorize == nil || h.dependencies.Candidates == nil || h.dependencies.Reserve == nil || h.dependencies.Transition == nil || h.dependencies.Abort == nil || h.dependencies.Persist == nil {
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
	maximumOutputTokens := input.MaxCompletionTokens
	if maximumOutputTokens == 0 {
		maximumOutputTokens = input.MaxTokens
	}
	if maximumOutputTokens == 0 {
		maximumOutputTokens = 4096
	}
	if maximumOutputTokens > 1_000_000 || (input.MaxTokens != 0 && input.MaxCompletionTokens != 0) {
		http.Error(response, "invalid output token limit", http.StatusBadRequest)
		return
	}
	maximumSpend, err := strconv.ParseUint(request.Header.Get("X-Myference-Max-Spend"), 10, 64)
	if err != nil || maximumSpend == 0 {
		http.Error(response, "valid X-Myference-Max-Spend is required", http.StatusBadRequest)
		return
	}
	token := strings.TrimSpace(strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer "))
	principal, err := h.dependencies.Authorize(request.Context(), token, input.Model, authorizedEndpoint(request.Context(), request.URL.Path), maximumSpend)
	if err != nil {
		http.Error(response, "unauthorized", http.StatusUnauthorized)
		return
	}
	candidates, err := h.dependencies.Candidates(request.Context(), input.Model)
	if err != nil {
		http.Error(response, "routing unavailable", http.StatusServiceUnavailable)
		return
	}
	requiredCapabilities := []string{"text", "stream"}
	if len(input.Workspace) != 0 {
		requiredCapabilities = append(requiredCapabilities, "workspace")
	}
	prompt := renderPrompt(input.Messages)
	// Byte length bounds tokenizer output, while the fixed allowance covers provider chat templates
	// and special tokens that are not present in the customer-visible prompt.
	maximumInputTokens := uint64(len([]byte(prompt)))*4 + 256
	const maximumComputeMilliseconds uint64 = 120_000
	for index := range candidates {
		if !h.dependencies.Hub.Connected(candidates[index].MachineID) {
			candidates[index].Capacity = 0
			continue
		}
		if !router.ValidRate(candidates[index].InputPerMillion) || !router.ValidRate(candidates[index].OutputPerMillion) || !router.ValidRate(candidates[index].ComputePerSecond) {
			candidates[index].MaximumCost = 0
			continue
		}
		if !router.HasPricing(candidates[index]) {
			candidates[index].MaximumCost = 0
			continue
		}
		cost, costErr := router.WorstCaseCost(candidates[index], maximumInputTokens, maximumOutputTokens, maximumComputeMilliseconds)
		if costErr != nil {
			candidates[index].MaximumCost = 0
		} else {
			candidates[index].MaximumCost = cost
		}
	}
	selected, err := router.Select(router.Request{Model: input.Model, Capabilities: requiredCapabilities, MaximumSpend: maximumSpend, SessionBalance: principal.SessionBalance, PinMachineID: request.Header.Get("X-Myference-Machine")}, candidates)
	if errors.Is(err, router.ErrInsufficientBudget) {
		http.Error(response, "request or session budget is below the required reservation", http.StatusPaymentRequired)
		return
	}
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
	reservation := Reservation{RequestID: requestID, SessionID: principal.SessionID, AccountID: principal.AccountID, MachineID: selected.MachineID, OfferID: selected.OfferID, PriceVersion: selected.PriceVersion, MaximumSpend: selected.MaximumCost, MaximumInputTokens: maximumInputTokens, MaximumOutputTokens: maximumOutputTokens, MaximumComputeMilliseconds: maximumComputeMilliseconds}
	if err := h.dependencies.Reserve(request.Context(), reservation); err != nil {
		http.Error(response, "reservation unavailable", http.StatusPaymentRequired)
		return
	}
	envelope, err := v1.NewEnvelope("offer-"+requestID, v1.MessageJobOffer, &v1.JobOffer{RequestID: requestID, Model: input.Model, OfferID: selected.OfferID, Prompt: prompt, PriceVersion: selected.PriceVersion, MaximumSpend: selected.MaximumCost, MaximumOutputTokens: maximumOutputTokens, LeaseExpiresAt: h.dependencies.Now().Add(time.Duration(maximumComputeMilliseconds) * time.Millisecond), Workspace: input.Workspace})
	if err != nil || h.dependencies.Hub.Send(selected.MachineID, envelope) != nil {
		_ = h.dependencies.Abort(request.Context(), requestID, "failed")
		http.Error(response, "provider unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := waitAccepted(request.Context(), events); err != nil {
		h.cancel(selected.MachineID, requestID)
		_ = h.dependencies.Abort(context.WithoutCancel(request.Context()), requestID, "failed")
		http.Error(response, "provider did not accept", http.StatusGatewayTimeout)
		return
	}
	if err := h.dependencies.Transition(request.Context(), requestID, "accepted"); err != nil {
		h.cancel(selected.MachineID, requestID)
		_ = h.dependencies.Abort(context.WithoutCancel(request.Context()), requestID, "failed")
		http.Error(response, "reservation transition failed", http.StatusInternalServerError)
		return
	}
	flusher, ok := response.(http.Flusher)
	if !ok {
		h.cancel(selected.MachineID, requestID)
		_ = h.dependencies.Abort(context.WithoutCancel(request.Context()), requestID, "failed")
		http.Error(response, "streaming unavailable", http.StatusInternalServerError)
		return
	}
	response.Header().Set("Content-Type", "text/event-stream")
	response.Header().Set("Cache-Control", "no-cache")
	response.Header().Set("X-Request-ID", requestID)
	response.WriteHeader(http.StatusOK)
	flusher.Flush()
	var output strings.Builder
	streaming := false
	generationTimer := time.NewTimer(time.Duration(maximumComputeMilliseconds) * time.Millisecond)
	defer generationTimer.Stop()
	for {
		select {
		case <-generationTimer.C:
			h.cancel(selected.MachineID, requestID)
			_ = h.dependencies.Abort(context.WithoutCancel(request.Context()), requestID, "failed")
			return
		case <-request.Context().Done():
			h.cancel(selected.MachineID, requestID)
			_ = h.dependencies.Abort(context.WithoutCancel(request.Context()), requestID, "cancelled")
			return
		case <-events.overflow:
			h.cancel(selected.MachineID, requestID)
			_ = h.dependencies.Abort(context.WithoutCancel(request.Context()), requestID, "failed")
			return
		case event := <-events.events:
			if event.Type != v1.MessageOutputChunk {
				continue
			}
			var chunk v1.OutputChunk
			if event.Envelope.DecodeBody(&chunk) != nil {
				h.cancel(selected.MachineID, requestID)
				_ = h.dependencies.Abort(context.WithoutCancel(request.Context()), requestID, "failed")
				return
			}
			if chunk.Data != "" {
				if !streaming {
					if h.dependencies.Transition(request.Context(), requestID, "streaming") != nil {
						h.cancel(selected.MachineID, requestID)
						_ = h.dependencies.Abort(context.WithoutCancel(request.Context()), requestID, "failed")
						return
					}
					streaming = true
				}
				output.WriteString(chunk.Data)
				if writeSSE(response, map[string]any{"id": requestID, "object": "chat.completion.chunk", "choices": []any{map[string]any{"index": 0, "delta": map[string]string{"content": chunk.Data}}}}) != nil {
					h.cancel(selected.MachineID, requestID)
					_ = h.dependencies.Abort(context.WithoutCancel(request.Context()), requestID, "failed")
					return
				}
				flusher.Flush()
			}
			if chunk.ErrorCode != "" {
				_ = h.dependencies.Abort(context.WithoutCancel(request.Context()), requestID, "failed")
				_ = writeSSE(response, map[string]any{"error": map[string]string{"type": "provider_error", "code": chunk.ErrorCode, "message": "provider execution failed"}})
				fmt.Fprint(response, "data: [DONE]\n\n")
				flusher.Flush()
				return
			}
			if chunk.Done {
				if !streaming {
					if h.dependencies.Transition(request.Context(), requestID, "streaming") != nil {
						_ = h.dependencies.Abort(context.WithoutCancel(request.Context()), requestID, "failed")
						return
					}
					streaming = true
				}
				if chunk.InputTokens > maximumInputTokens || chunk.OutputTokens > maximumOutputTokens || chunk.ComputeMilliseconds > maximumComputeMilliseconds {
					h.cancel(selected.MachineID, requestID)
					_ = h.dependencies.Abort(context.WithoutCancel(request.Context()), requestID, "failed")
					_ = writeSSE(response, map[string]any{"error": map[string]string{"type": "provider_error", "code": "usage_limit_exceeded", "message": "provider usage exceeded reserved limits"}})
					fmt.Fprint(response, "data: [DONE]\n\n")
					flusher.Flush()
					return
				}
				proposal := Proposal{RequestID: requestID, SessionID: principal.SessionID, MachineID: selected.MachineID, OfferID: selected.OfferID, Model: input.Model, PriceVersion: selected.PriceVersion, InputTokens: chunk.InputTokens, OutputTokens: chunk.OutputTokens, ComputeMilliseconds: chunk.ComputeMilliseconds, InputHash: saltedHash(prompt), OutputHash: saltedHash(output.String()), CompletedAt: h.dependencies.Now()}
				if h.dependencies.Persist(request.Context(), proposal) != nil {
					_ = h.dependencies.Abort(context.WithoutCancel(request.Context()), requestID, "failed")
					return
				}
				_ = writeSSE(response, map[string]any{"id": requestID, "object": "chat.completion.chunk", "model": input.Model, "choices": []any{map[string]any{"index": 0, "delta": map[string]string{}, "finish_reason": "stop"}}, "usage": map[string]uint64{"prompt_tokens": chunk.InputTokens, "completion_tokens": chunk.OutputTokens, "total_tokens": chunk.InputTokens + chunk.OutputTokens}})
				fmt.Fprint(response, "data: [DONE]\n\n")
				flusher.Flush()
				return
			}
		}
	}
}

func waitAccepted(ctx context.Context, stream *brokerStream) error {
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return errors.New("accept timeout")
		case <-stream.overflow:
			return relay.ErrBackpressure
		case event := <-stream.events:
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
	value := make([]byte, 32)
	_, err := rand.Read(value)
	return "0x" + hex.EncodeToString(value), err
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
	streams map[string]*brokerStream
}

type brokerStream struct {
	events   chan relay.Event
	overflow chan struct{}
}

func newEventBroker(hub *relay.Hub) *eventBroker {
	broker := &eventBroker{hub: hub, streams: make(map[string]*brokerStream)}
	if hub != nil {
		go broker.run()
	}
	return broker
}

func (b *eventBroker) register(requestID string) *brokerStream {
	b.mu.Lock()
	defer b.mu.Unlock()
	stream := &brokerStream{events: make(chan relay.Event, 32), overflow: make(chan struct{}, 1)}
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
			case stream.events <- event:
			default:
				select {
				case stream.overflow <- struct{}{}:
				default:
				}
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
