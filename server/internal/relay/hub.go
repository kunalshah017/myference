package relay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	v1 "github.com/kunalshah017/myference/protocol/v1"
)

var (
	ErrUnauthorized = errors.New("relay unauthorized")
	ErrDisconnected = errors.New("provider disconnected")
	ErrBackpressure = errors.New("provider outbound queue full")
)

type Authenticator func(context.Context, string) (string, error)

type Options struct {
	QueueSize        int
	HeartbeatTimeout time.Duration
	MaximumMessage   int64
	CapacityHandler  func(string, v1.Capacity) error
}

type Event struct {
	MachineID string
	Type      string
	Envelope  v1.Envelope
}

type session struct {
	connection *websocket.Conn
	outbound   chan v1.Envelope
	lastSeen   time.Time
}

type Hub struct {
	auth    Authenticator
	options Options

	mu            sync.RWMutex
	sessions      map[string]*session
	outputStarted map[string]bool
	chunks        *v1.ChunkTracker
	events        chan Event
	subscribers   map[string]map[chan Event]struct{}
}

func NewHub(auth Authenticator, options Options) *Hub {
	if options.QueueSize <= 0 {
		options.QueueSize = 16
	}
	if options.HeartbeatTimeout <= 0 {
		options.HeartbeatTimeout = 30 * time.Second
	}
	if options.MaximumMessage <= 0 {
		options.MaximumMessage = 1 << 20
	}
	return &Hub{auth: auth, options: options, sessions: make(map[string]*session), outputStarted: make(map[string]bool), chunks: v1.NewChunkTracker(), events: make(chan Event, 64), subscribers: make(map[string]map[chan Event]struct{})}
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" || h.auth == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	machineID, err := h.auth(r.Context(), token)
	if err != nil || machineID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	connection, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	current := &session{connection: connection, outbound: make(chan v1.Envelope, h.options.QueueSize), lastSeen: time.Now()}
	h.mu.Lock()
	if old := h.sessions[machineID]; old != nil {
		old.connection.Close(websocket.StatusPolicyViolation, "replaced by reconnect")
	}
	h.sessions[machineID] = current
	h.mu.Unlock()
	defer h.unregister(machineID, current)
	go h.writeLoop(ctx, current)
	go h.heartbeatLoop(ctx, machineID, current)

	for {
		_, payload, err := connection.Read(ctx)
		if err != nil {
			return
		}
		h.mu.Lock()
		current.lastSeen = time.Now()
		h.mu.Unlock()
		envelope, err := v1.DecodeEnvelope(bytes.NewReader(payload), h.options.MaximumMessage)
		if err != nil || h.AcceptInbound(machineID, envelope) != nil {
			connection.Close(websocket.StatusPolicyViolation, "invalid relay message")
			return
		}
	}
}

func (h *Hub) AcceptInbound(machineID string, envelope v1.Envelope) error {
	switch envelope.Type {
	case v1.MessageHello:
		var hello v1.Hello
		if err := envelope.DecodeBody(&hello); err != nil || hello.MachineID != machineID {
			return v1.ErrInvalidMessage
		}
		return nil
	case v1.MessageCapacity:
		var capacity v1.Capacity
		if err := envelope.DecodeBody(&capacity); err != nil {
			return err
		}
		if h.options.CapacityHandler != nil {
			if err := h.options.CapacityHandler(machineID, capacity); err != nil {
				return err
			}
		}
	case v1.MessageJobAccept:
		var accepted v1.JobAccept
		if err := envelope.DecodeBody(&accepted); err != nil {
			return err
		}
	case v1.MessageOutputChunk:
		var chunk v1.OutputChunk
		if err := envelope.DecodeBody(&chunk); err != nil {
			return err
		}
		if err := h.chunks.Accept(chunk); err != nil {
			return err
		}
		h.mu.Lock()
		h.outputStarted[chunk.RequestID] = true
		h.mu.Unlock()
	case v1.MessageReceiptSignature:
		var signature v1.ReceiptSignature
		if err := envelope.DecodeBody(&signature); err != nil {
			return err
		}
	default:
		return v1.ErrInvalidMessage
	}
	event := Event{MachineID: machineID, Type: envelope.Type, Envelope: envelope}
	h.publish(requestID(envelope), event)
	select {
	case h.events <- event:
		return nil
	default:
		return ErrBackpressure
	}
}

func (h *Hub) Send(machineID string, envelope v1.Envelope) error {
	h.mu.RLock()
	current := h.sessions[machineID]
	h.mu.RUnlock()
	if current == nil {
		return ErrDisconnected
	}
	select {
	case current.outbound <- envelope:
		return nil
	default:
		return ErrBackpressure
	}
}

func (h *Hub) Events() <-chan Event { return h.events }

func (h *Hub) Subscribe(requestID string) (<-chan Event, func()) {
	events := make(chan Event, 8)
	h.mu.Lock()
	if h.subscribers[requestID] == nil {
		h.subscribers[requestID] = make(map[chan Event]struct{})
	}
	h.subscribers[requestID][events] = struct{}{}
	h.mu.Unlock()
	return events, func() {
		h.mu.Lock()
		delete(h.subscribers[requestID], events)
		if len(h.subscribers[requestID]) == 0 {
			delete(h.subscribers, requestID)
		}
		h.mu.Unlock()
	}
}

func (h *Hub) publish(requestID string, event Event) {
	if requestID == "" {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for subscriber := range h.subscribers[requestID] {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func requestID(envelope v1.Envelope) string {
	switch envelope.Type {
	case v1.MessageJobAccept:
		var value v1.JobAccept
		if envelope.DecodeBody(&value) == nil {
			return value.RequestID
		}
	case v1.MessageOutputChunk:
		var value v1.OutputChunk
		if envelope.DecodeBody(&value) == nil {
			return value.RequestID
		}
	case v1.MessageReceiptSignature:
		var value v1.ReceiptSignature
		if envelope.DecodeBody(&value) == nil {
			return value.RequestID
		}
	}
	return ""
}

func (h *Hub) CanRetry(requestID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return !h.outputStarted[requestID]
}

func (h *Hub) Connected(machineID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.sessions[machineID] != nil
}

func (h *Hub) writeLoop(ctx context.Context, current *session) {
	for {
		select {
		case <-ctx.Done():
			return
		case envelope := <-current.outbound:
			payload, err := json.Marshal(envelope)
			if err != nil || current.connection.Write(ctx, websocket.MessageText, payload) != nil {
				current.connection.Close(websocket.StatusInternalError, "relay write failed")
				return
			}
		}
	}
}

func (h *Hub) heartbeatLoop(ctx context.Context, machineID string, current *session) {
	ticker := time.NewTicker(h.options.HeartbeatTimeout / 2)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.mu.RLock()
			lastSeen := current.lastSeen
			h.mu.RUnlock()
			if time.Since(lastSeen) > h.options.HeartbeatTimeout {
				h.unregister(machineID, current)
				current.connection.Close(websocket.StatusGoingAway, "heartbeat expired")
				return
			}
		}
	}
}

func (h *Hub) unregister(machineID string, current *session) {
	h.mu.Lock()
	if h.sessions[machineID] == current {
		delete(h.sessions, machineID)
	}
	h.mu.Unlock()
}
