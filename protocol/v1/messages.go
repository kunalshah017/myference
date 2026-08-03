package v1

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	MaximumWorkspaceFiles = 64
	MaximumWorkspaceBytes = 512 << 10
)

var windowsAbsolutePath = regexp.MustCompile(`^[A-Za-z]:[\\/]`)

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
	OfferID        string   `json:"offer_id"`
	Model          string   `json:"model"`
	PriceVersion   uint64   `json:"price_version"`
	BackendKind    string   `json:"backend_kind,omitempty"`
	OfferHash      string   `json:"offer_hash,omitempty"`
	ModelHash      string   `json:"model_hash,omitempty"`
	CapabilityHash string   `json:"capability_hash,omitempty"`
	Capabilities   []string `json:"capabilities,omitempty"`
	EvidenceKind   string   `json:"evidence_kind,omitempty"`
	EvidenceDigest string   `json:"evidence_digest,omitempty"`
	MeteringMode   string   `json:"metering_mode,omitempty"`
}

func (o OfferCapacity) Validate() error {
	if strings.TrimSpace(o.OfferID) == "" || strings.TrimSpace(o.Model) == "" || o.PriceVersion == 0 {
		return ErrInvalidMessage
	}
	hashes := []string{o.OfferHash, o.ModelHash, o.CapabilityHash}
	provided := 0
	for _, hash := range hashes {
		if hash != "" {
			provided++
			if len(hash) != 66 || !strings.HasPrefix(hash, "0x") {
				return ErrInvalidMessage
			}
		}
	}
	if provided != 0 && (provided != len(hashes) || strings.TrimSpace(o.BackendKind) == "") {
		return ErrInvalidMessage
	}
	previous := ""
	for _, capability := range o.Capabilities {
		if (capability != "stream" && capability != "text" && capability != "workspace") || capability <= previous {
			return ErrInvalidMessage
		}
		previous = capability
	}
	if provided != 0 && len(o.Capabilities) == 0 {
		return ErrInvalidMessage
	}
	if o.EvidenceKind != "" || o.EvidenceDigest != "" || o.MeteringMode != "" {
		if o.EvidenceDigest == "" || (o.EvidenceKind != "ollama_digest" && o.EvidenceKind != "upstream_model" && o.EvidenceKind != "runtime_image") {
			return ErrInvalidMessage
		}
		if o.MeteringMode != "tokens_and_compute" && o.MeteringMode != "compute_only" {
			return ErrInvalidMessage
		}
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
	RequestID           string          `json:"request_id"`
	Model               string          `json:"model"`
	OfferID             string          `json:"offer_id"`
	Prompt              string          `json:"prompt"`
	PriceVersion        uint64          `json:"price_version"`
	MaximumSpend        uint64          `json:"maximum_spend"`
	MaximumOutputTokens uint64          `json:"maximum_output_tokens"`
	LeaseExpiresAt      time.Time       `json:"lease_expires_at"`
	Workspace           []WorkspaceFile `json:"workspace,omitempty"`
}

type WorkspaceFile struct {
	Path          string `json:"path"`
	ContentBase64 string `json:"content_base64"`
}

func (m JobOffer) Validate() error {
	if strings.TrimSpace(m.RequestID) == "" || strings.TrimSpace(m.Model) == "" || strings.TrimSpace(m.OfferID) == "" || strings.TrimSpace(m.Prompt) == "" || m.PriceVersion == 0 || m.MaximumSpend == 0 || m.MaximumOutputTokens == 0 || !m.LeaseExpiresAt.After(time.Now()) {
		return ErrInvalidMessage
	}
	if len(m.Workspace) > MaximumWorkspaceFiles {
		return ErrInvalidMessage
	}
	total := 0
	paths := make([]string, 0, len(m.Workspace))
	seen := make(map[string]struct{}, len(m.Workspace))
	for _, file := range m.Workspace {
		name := strings.TrimSpace(file.Path)
		clean := path.Clean(strings.ReplaceAll(name, `\`, "/"))
		if name == "" || strings.ContainsRune(name, '\\') || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") || windowsAbsolutePath.MatchString(name) || clean != name {
			return ErrInvalidMessage
		}
		if _, exists := seen[clean]; exists {
			return ErrInvalidMessage
		}
		seen[clean] = struct{}{}
		paths = append(paths, clean)
		decoded, err := base64.StdEncoding.DecodeString(file.ContentBase64)
		if err != nil || len(decoded) > MaximumWorkspaceBytes-total {
			return ErrInvalidMessage
		}
		total += len(decoded)
	}
	sort.Strings(paths)
	for index := 1; index < len(paths); index++ {
		if strings.HasPrefix(paths[index], paths[index-1]+"/") {
			return ErrInvalidMessage
		}
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
	ErrorCode           string `json:"error_code,omitempty"`
}

func (m OutputChunk) Validate() error {
	if strings.TrimSpace(m.RequestID) == "" || m.Sequence == 0 || (m.Data == "" && !m.Done) || (m.ErrorCode != "" && (!m.Done || m.Data != "" || m.InputTokens != 0 || m.OutputTokens != 0 || m.ComputeMilliseconds != 0)) {
		return ErrInvalidMessage
	}
	if m.ErrorCode != "" && m.ErrorCode != "backend_failed" && m.ErrorCode != "cancelled" {
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
	ChainID   uint64  `json:"chain_id"`
	Contract  Address `json:"contract"`
	Receipt   Receipt `json:"receipt"`
}

func (m ReceiptProposal) Validate() error {
	if strings.TrimSpace(m.RequestID) == "" || m.ChainID == 0 || m.Contract.IsZero() {
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
