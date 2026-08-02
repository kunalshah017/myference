package v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

const ProtocolVersion uint16 = 1

const (
	MessageHello            = "hello"
	MessageCapacity         = "capacity"
	MessageJobOffer         = "job_offer"
	MessageJobAccept        = "job_accept"
	MessageOutputChunk      = "output_chunk"
	MessageCancel           = "cancel"
	MessageReceiptProposal  = "receipt_proposal"
	MessageReceiptSignature = "receipt_signature"
)

var (
	ErrInvalidMessage     = errors.New("invalid protocol message")
	ErrMessageTooLarge    = errors.New("protocol message too large")
	ErrUnsupportedVersion = errors.New("unsupported protocol version")
	ErrChunkSequence      = errors.New("invalid output chunk sequence")
)

type Validatable interface {
	Validate() error
}

type Envelope struct {
	Version uint16          `json:"version"`
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	Body    json.RawMessage `json:"body"`
}

func NewEnvelope(id, messageType string, message Validatable) (Envelope, error) {
	if strings.TrimSpace(id) == "" || !knownMessageType(messageType) || message == nil {
		return Envelope{}, ErrInvalidMessage
	}
	if err := message.Validate(); err != nil {
		return Envelope{}, err
	}
	body, err := json.Marshal(message)
	if err != nil {
		return Envelope{}, fmt.Errorf("marshal message body: %w", err)
	}
	return Envelope{Version: ProtocolVersion, Type: messageType, ID: id, Body: body}, nil
}

func DecodeEnvelope(reader io.Reader, maximumBytes int64) (Envelope, error) {
	if maximumBytes <= 0 {
		return Envelope{}, ErrInvalidMessage
	}
	data, err := io.ReadAll(io.LimitReader(reader, maximumBytes+1))
	if err != nil {
		return Envelope{}, fmt.Errorf("read protocol message: %w", err)
	}
	if int64(len(data)) > maximumBytes {
		return Envelope{}, ErrMessageTooLarge
	}
	var envelope Envelope
	if err := decodeStrict(data, &envelope); err != nil {
		return Envelope{}, fmt.Errorf("%w: %v", ErrInvalidMessage, err)
	}
	if envelope.Version != ProtocolVersion {
		return Envelope{}, ErrUnsupportedVersion
	}
	if strings.TrimSpace(envelope.ID) == "" || !knownMessageType(envelope.Type) || len(envelope.Body) == 0 {
		return Envelope{}, ErrInvalidMessage
	}
	return envelope, nil
}

func (e Envelope) DecodeBody(target Validatable) error {
	if target == nil {
		return ErrInvalidMessage
	}
	if err := decodeStrict(e.Body, target); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidMessage, err)
	}
	return target.Validate()
}

func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func knownMessageType(messageType string) bool {
	switch messageType {
	case MessageHello, MessageCapacity, MessageJobOffer, MessageJobAccept, MessageOutputChunk, MessageCancel, MessageReceiptProposal, MessageReceiptSignature:
		return true
	default:
		return false
	}
}

type Hello struct {
	MachineID string `json:"machine_id"`
}

func (m Hello) Validate() error {
	if strings.TrimSpace(m.MachineID) == "" {
		return ErrInvalidMessage
	}
	return nil
}

type OfferCapacity struct {
	OfferID      string `json:"offer_id"`
	Model        string `json:"model"`
	PriceVersion uint64 `json:"price_version"`
}

func (o OfferCapacity) Validate() error {
	if strings.TrimSpace(o.OfferID) == "" || strings.TrimSpace(o.Model) == "" || o.PriceVersion == 0 {
		return ErrInvalidMessage
	}
	return nil
}

type Capacity struct {
	Available uint32          `json:"available"`
	Offers    []OfferCapacity `json:"offers"`
}

func (m Capacity) Validate() error {
	for _, offer := range m.Offers {
		if err := offer.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type JobOffer struct {
	RequestID      string    `json:"request_id"`
	Model          string    `json:"model"`
	OfferID        string    `json:"offer_id"`
	Prompt         string    `json:"prompt"`
	PriceVersion   uint64    `json:"price_version"`
	MaximumSpend   uint64    `json:"maximum_spend"`
	LeaseExpiresAt time.Time `json:"lease_expires_at"`
}

func (m JobOffer) Validate() error {
	if strings.TrimSpace(m.RequestID) == "" || strings.TrimSpace(m.Model) == "" || strings.TrimSpace(m.OfferID) == "" || strings.TrimSpace(m.Prompt) == "" || m.PriceVersion == 0 || m.MaximumSpend == 0 || !m.LeaseExpiresAt.After(time.Now()) {
		return ErrInvalidMessage
	}
	return nil
}

type JobAccept struct {
	RequestID string `json:"request_id"`
}

func (m JobAccept) Validate() error {
	if strings.TrimSpace(m.RequestID) == "" {
		return ErrInvalidMessage
	}
	return nil
}

type OutputChunk struct {
	RequestID           string `json:"request_id"`
	Sequence            uint64 `json:"sequence"`
	Data                string `json:"data,omitempty"`
	Done                bool   `json:"done,omitempty"`
	InputTokens         uint64 `json:"input_tokens,omitempty"`
	OutputTokens        uint64 `json:"output_tokens,omitempty"`
	ComputeMilliseconds uint64 `json:"compute_milliseconds,omitempty"`
}

func (m OutputChunk) Validate() error {
	if strings.TrimSpace(m.RequestID) == "" || m.Sequence == 0 || (m.Data == "" && !m.Done) {
		return ErrInvalidMessage
	}
	return nil
}

type Cancel struct {
	RequestID string `json:"request_id"`
	Reason    string `json:"reason"`
}

func (m Cancel) Validate() error {
	if strings.TrimSpace(m.RequestID) == "" || strings.TrimSpace(m.Reason) == "" {
		return ErrInvalidMessage
	}
	return nil
}

type ReceiptProposal struct {
	RequestID string  `json:"request_id"`
	Receipt   Receipt `json:"receipt"`
}

func (m ReceiptProposal) Validate() error {
	if strings.TrimSpace(m.RequestID) == "" {
		return ErrInvalidMessage
	}
	return m.Receipt.Validate()
}

type ReceiptSignature struct {
	RequestID string  `json:"request_id"`
	Signer    Address `json:"signer"`
	Signature []byte  `json:"signature"`
}

func (m ReceiptSignature) Validate() error {
	if strings.TrimSpace(m.RequestID) == "" || m.Signer.IsZero() || len(m.Signature) != 65 {
		return ErrInvalidMessage
	}
	return nil
}

type ChunkTracker struct {
	mu     sync.Mutex
	last   map[string]uint64
	closed map[string]bool
}

func NewChunkTracker() *ChunkTracker {
	return &ChunkTracker{last: make(map[string]uint64), closed: make(map[string]bool)}
}

func (t *ChunkTracker) Accept(chunk OutputChunk) error {
	if err := chunk.Validate(); err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed[chunk.RequestID] || chunk.Sequence != t.last[chunk.RequestID]+1 {
		return ErrChunkSequence
	}
	t.last[chunk.RequestID] = chunk.Sequence
	if chunk.Done {
		t.closed[chunk.RequestID] = true
	}
	return nil
}
