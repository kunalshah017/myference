package v1

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestMessageRoundTripAndValidation(t *testing.T) {
	lease := time.Now().Add(time.Minute).UTC().Truncate(time.Second)
	messages := []struct {
		name     string
		typeName string
		message  Validatable
		target   func() Validatable
	}{
		{"hello", MessageHello, &Hello{MachineID: "machine-1"}, func() Validatable { return &Hello{} }},
		{"capacity", MessageCapacity, &Capacity{Available: 1, Offers: []OfferCapacity{{OfferID: "offer-1", Model: "qwen", PriceVersion: 1}}}, func() Validatable { return &Capacity{} }},
		{"job offer", MessageJobOffer, &JobOffer{RequestID: "request-1", Model: "qwen", OfferID: "offer-1", PriceVersion: 1, MaximumSpend: 10, MaximumOutputTokens: 100, LeaseExpiresAt: lease, Prompt: "hello", Workspace: []WorkspaceFile{{Path: "src/main.go", ContentBase64: "cGFja2FnZSBtYWlu"}}}, func() Validatable { return &JobOffer{} }},
		{"job accept", MessageJobAccept, &JobAccept{RequestID: "request-1"}, func() Validatable { return &JobAccept{} }},
		{"output chunk", MessageOutputChunk, &OutputChunk{RequestID: "request-1", Sequence: 1, Data: "hello"}, func() Validatable { return &OutputChunk{} }},
		{"output failure", MessageOutputChunk, &OutputChunk{RequestID: "request-1", Sequence: 1, Done: true, ErrorCode: "backend_failed"}, func() Validatable { return &OutputChunk{} }},
		{"cancel", MessageCancel, &Cancel{RequestID: "request-1", Reason: "customer_cancelled"}, func() Validatable { return &Cancel{} }},
		{"receipt proposal", MessageReceiptProposal, &ReceiptProposal{RequestID: "request-1", ChainID: 10143, Contract: addressWithLastByte(9), Receipt: validReceipt()}, func() Validatable { return &ReceiptProposal{} }},
		{"receipt signature", MessageReceiptSignature, &ReceiptSignature{RequestID: "request-1", Signer: addressWithLastByte(1), Signature: bytes.Repeat([]byte{1}, 65)}, func() Validatable { return &ReceiptSignature{} }},
	}

	for _, test := range messages {
		t.Run(test.name, func(t *testing.T) {
			envelope, err := NewEnvelope("message-1", test.typeName, test.message)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(envelope)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := DecodeEnvelope(bytes.NewReader(encoded), int64(len(encoded)))
			if err != nil {
				t.Fatal(err)
			}
			if decoded.Version != ProtocolVersion || decoded.ID != "message-1" || decoded.Type != test.typeName {
				t.Fatalf("unexpected envelope: %+v", decoded)
			}
			if err := decoded.DecodeBody(test.target()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestWorkspaceRejectsTraversalAbsolutePathsAndOversize(t *testing.T) {
	base := JobOffer{RequestID: "request", Model: "agent", OfferID: "offer", PriceVersion: 1, MaximumSpend: 1, MaximumOutputTokens: 100, LeaseExpiresAt: time.Now().Add(time.Minute), Prompt: "work"}
	for _, filePath := range []string{"../secret", "/etc/passwd", `C:\\Users\\secret`, `src\\main.go`} {
		candidate := base
		candidate.Workspace = []WorkspaceFile{{Path: filePath, ContentBase64: "eA=="}}
		if err := candidate.Validate(); !errors.Is(err, ErrInvalidMessage) {
			t.Fatalf("path=%q err=%v", filePath, err)
		}
	}
	candidate := base
	candidate.Workspace = []WorkspaceFile{{Path: "large.bin", ContentBase64: strings.Repeat("A", 14<<20)}}
	if err := candidate.Validate(); !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("oversize err=%v", err)
	}
	for _, files := range [][]WorkspaceFile{
		{{Path: "same", ContentBase64: "eA=="}, {Path: "same", ContentBase64: "eQ=="}},
		{{Path: "src", ContentBase64: "eA=="}, {Path: "src/main.go", ContentBase64: "eQ=="}},
	} {
		candidate := base
		candidate.Workspace = files
		if err := candidate.Validate(); !errors.Is(err, ErrInvalidMessage) {
			t.Fatalf("conflicting workspace paths accepted: %+v", files)
		}
	}
}

func TestMessageRejectsUnknownFieldsAndOversize(t *testing.T) {
	unknown := `{"version":1,"type":"hello","id":"message-1","body":{},"extra":true}`
	if _, err := DecodeEnvelope(strings.NewReader(unknown), int64(len(unknown))); !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("expected ErrInvalidMessage for unknown field, got %v", err)
	}

	valid := `{"version":1,"type":"hello","id":"message-1","body":{"machine_id":"machine-1"}}`
	if _, err := DecodeEnvelope(strings.NewReader(valid), int64(len(valid)-1)); !errors.Is(err, ErrMessageTooLarge) {
		t.Fatalf("expected ErrMessageTooLarge, got %v", err)
	}
}

func TestMessageRejectsInvalidVersionAndEmptyRequestID(t *testing.T) {
	body, _ := json.Marshal(Hello{MachineID: "machine-1"})
	envelope := Envelope{Version: ProtocolVersion + 1, Type: MessageHello, ID: "message-1", Body: body}
	encoded, _ := json.Marshal(envelope)
	if _, err := DecodeEnvelope(bytes.NewReader(encoded), int64(len(encoded))); !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("expected ErrUnsupportedVersion, got %v", err)
	}

	if err := (&JobAccept{}).Validate(); !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("expected ErrInvalidMessage, got %v", err)
	}
	if err := (&ReceiptProposal{RequestID: "request-1", Receipt: validReceipt()}).Validate(); !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("expected receipt domain rejection, got %v", err)
	}
}

func TestOfferCapacityValidatesRuntimeEvidenceAndMetering(t *testing.T) {
	valid := OfferCapacity{OfferID: "offer-1", Model: "qwen", PriceVersion: 1, EvidenceKind: "ollama_digest", EvidenceDigest: "sha256:abc", MeteringMode: "tokens_and_compute"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid runtime evidence rejected: %v", err)
	}
	claimed := OfferCapacity{OfferID: "offer-1", Model: "qwen", PriceVersion: 1, EvidenceKind: "provider_claimed", EvidenceDigest: "qwen", MeteringMode: "tokens_and_compute"}
	if err := claimed.Validate(); err != nil {
		t.Fatalf("honest provider-claimed evidence rejected: %v", err)
	}
	for _, offer := range []OfferCapacity{
		{OfferID: "offer-1", Model: "qwen", PriceVersion: 1, EvidenceKind: "invented", EvidenceDigest: "abc", MeteringMode: "tokens_and_compute"},
		{OfferID: "offer-1", Model: "qwen", PriceVersion: 1, EvidenceKind: "ollama_digest", MeteringMode: "tokens_and_compute"},
		{OfferID: "offer-1", Model: "qwen", PriceVersion: 1, EvidenceKind: "ollama_digest", EvidenceDigest: "abc", MeteringMode: "tokens"},
	} {
		if err := offer.Validate(); err == nil {
			t.Fatalf("invalid runtime evidence accepted: %+v", offer)
		}
	}
}

func TestMessageChunkSequenceIsStrictlyMonotonic(t *testing.T) {
	tracker := NewChunkTracker()
	for _, sequence := range []uint64{1, 2, 3} {
		if err := tracker.Accept(OutputChunk{RequestID: "request-1", Sequence: sequence, Data: "x"}); err != nil {
			t.Fatalf("sequence %d: %v", sequence, err)
		}
	}
	if err := tracker.Accept(OutputChunk{RequestID: "request-1", Sequence: 3, Data: "duplicate"}); !errors.Is(err, ErrChunkSequence) {
		t.Fatalf("expected ErrChunkSequence for duplicate, got %v", err)
	}
	if err := tracker.Accept(OutputChunk{RequestID: "request-1", Sequence: 5, Data: "gap"}); !errors.Is(err, ErrChunkSequence) {
		t.Fatalf("expected ErrChunkSequence for gap, got %v", err)
	}
}
