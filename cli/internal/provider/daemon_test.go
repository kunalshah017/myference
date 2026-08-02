package provider

import (
	"errors"
	"testing"

	v1 "github.com/kunalshah017/myference/protocol/v1"
)

func TestRequestStateAcceptsOneLeaseOrdersChunksAndReconnectsFromCursor(t *testing.T) {
	state := NewRequestState()
	if err := state.Accept("request-1"); err != nil {
		t.Fatal(err)
	}
	if err := state.Accept("request-1"); !errors.Is(err, ErrLeaseAlreadyAccepted) {
		t.Fatalf("expected duplicate lease rejection, got %v", err)
	}
	if err := state.RecordChunk(v1.OutputChunk{RequestID: "request-1", Sequence: 1, Data: "a"}); err != nil {
		t.Fatal(err)
	}
	if state.CanRetry("request-1") {
		t.Fatal("retry allowed after first output chunk")
	}
	cursor := state.Cursor("request-1")
	reconnected := NewRequestState()
	reconnected.Restore("request-1", cursor)
	if err := reconnected.RecordChunk(v1.OutputChunk{RequestID: "request-1", Sequence: 2, Done: true}); err != nil {
		t.Fatal(err)
	}
	if err := reconnected.Cancel("request-1"); !errors.Is(err, ErrRequestTerminal) {
		t.Fatalf("expected terminal cancellation rejection, got %v", err)
	}
}

func TestBoundedQueueAppliesBackpressure(t *testing.T) {
	queue := NewOutboundQueue(1)
	if err := queue.TryPush(v1.Envelope{ID: "one"}); err != nil {
		t.Fatal(err)
	}
	if err := queue.TryPush(v1.Envelope{ID: "two"}); !errors.Is(err, ErrBackpressure) {
		t.Fatalf("expected backpressure, got %v", err)
	}
}
