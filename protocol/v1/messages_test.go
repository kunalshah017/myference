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
		{"job offer", MessageJobOffer, &JobOffer{RequestID: "request-1", Model: "qwen", OfferID: "offer-1", PriceVersion: 1, MaximumSpend: 10, LeaseExpiresAt: lease, Prompt: "hello"}, func() Validatable { return &JobOffer{} }},
		{"job accept", MessageJobAccept, &JobAccept{RequestID: "request-1"}, func() Validatable { return &JobAccept{} }},
		{"output chunk", MessageOutputChunk, &OutputChunk{RequestID: "request-1", Sequence: 1, Data: "hello"}, func() Validatable { return &OutputChunk{} }},
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
