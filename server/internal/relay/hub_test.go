package relay

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	v1 "github.com/kunalshah017/myference/protocol/v1"
)

func TestHubAuthenticatesRealTLSWebSocketAndTracksCapacityChunksAndCancellation(t *testing.T) {
	hub := NewHub(func(_ context.Context, token string) (string, error) {
		if token != "machine-token" {
			return "", ErrUnauthorized
		}
		return "machine-1", nil
	}, Options{QueueSize: 4, HeartbeatTimeout: time.Second})
	server := httptest.NewTLSServer(hub)
	defer server.Close()
	client := server.Client()
	client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true} // test server certificate
	url := "wss" + strings.TrimPrefix(server.URL, "https")

	if connection, _, err := websocket.Dial(context.Background(), url, &websocket.DialOptions{HTTPClient: client}); err == nil {
		connection.Close(websocket.StatusNormalClosure, "")
		t.Fatal("unauthenticated connection succeeded")
	}
	header := http.Header{"Authorization": {"Bearer machine-token"}}
	connection, _, err := websocket.Dial(context.Background(), url, &websocket.DialOptions{HTTPClient: client, HTTPHeader: header})
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(websocket.StatusNormalClosure, "")

	writeEnvelope(t, connection, "hello-1", v1.MessageHello, &v1.Hello{MachineID: "machine-1"})
	writeEnvelope(t, connection, "capacity-1", v1.MessageCapacity, &v1.Capacity{Available: 1, Offers: []v1.OfferCapacity{{OfferID: "offer-1", Model: "qwen3:8b", PriceVersion: 1}}})
	event := waitEvent(t, hub)
	if event.Type != v1.MessageCapacity || event.MachineID != "machine-1" {
		t.Fatalf("unexpected capacity event: %+v", event)
	}

	writeEnvelope(t, connection, "accept-1", v1.MessageJobAccept, &v1.JobAccept{RequestID: "request-1"})
	writeEnvelope(t, connection, "chunk-1", v1.MessageOutputChunk, &v1.OutputChunk{RequestID: "request-1", Sequence: 1, Data: "hello"})
	writeEnvelope(t, connection, "chunk-2", v1.MessageOutputChunk, &v1.OutputChunk{RequestID: "request-1", Sequence: 2, Done: true})
	for range 3 {
		waitEvent(t, hub)
	}
	if hub.CanRetry("request-1") {
		t.Fatal("request became retryable after output began")
	}

	cancel, err := v1.NewEnvelope("cancel-1", v1.MessageCancel, &v1.Cancel{RequestID: "request-1", Reason: "customer_cancelled"})
	if err != nil {
		t.Fatal(err)
	}
	if err := hub.Send("machine-1", cancel); err != nil {
		t.Fatal(err)
	}
	_, payload, err := connection.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var received v1.Envelope
	if err := json.Unmarshal(payload, &received); err != nil || received.Type != v1.MessageCancel {
		t.Fatalf("received=%+v err=%v", received, err)
	}
}

func TestHubRejectsDuplicateChunksAndExpiresSilentConnections(t *testing.T) {
	hub := NewHub(func(context.Context, string) (string, error) { return "machine-1", nil }, Options{QueueSize: 1, HeartbeatTimeout: 50 * time.Millisecond})
	if err := hub.AcceptInbound("machine-1", envelope(t, "one", v1.MessageOutputChunk, &v1.OutputChunk{RequestID: "request-1", Sequence: 1, Data: "x"})); err != nil {
		t.Fatal(err)
	}
	if err := hub.AcceptInbound("machine-1", envelope(t, "duplicate", v1.MessageOutputChunk, &v1.OutputChunk{RequestID: "request-1", Sequence: 1, Data: "x"})); !errors.Is(err, v1.ErrChunkSequence) {
		t.Fatalf("expected duplicate rejection, got %v", err)
	}
	server := httptest.NewTLSServer(hub)
	defer server.Close()
	client := server.Client()
	client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true} // test server certificate
	connection, _, err := websocket.Dial(context.Background(), "wss"+strings.TrimPrefix(server.URL, "https"), &websocket.DialOptions{HTTPClient: client, HTTPHeader: http.Header{"Authorization": {"Bearer token"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(websocket.StatusNormalClosure, "")
	deadline := time.Now().Add(time.Second)
	for !hub.Connected("machine-1") && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	for hub.Connected("machine-1") && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if hub.Connected("machine-1") {
		t.Fatal("silent connection did not expire")
	}
}

func writeEnvelope(t *testing.T, connection *websocket.Conn, id, messageType string, body v1.Validatable) {
	t.Helper()
	encoded, err := json.Marshal(envelope(t, id, messageType, body))
	if err != nil {
		t.Fatal(err)
	}
	if err := connection.Write(context.Background(), websocket.MessageText, encoded); err != nil {
		t.Fatal(err)
	}
}

func envelope(t *testing.T, id, messageType string, body v1.Validatable) v1.Envelope {
	t.Helper()
	message, err := v1.NewEnvelope(id, messageType, body)
	if err != nil {
		t.Fatal(err)
	}
	return message
}

func waitEvent(t *testing.T, hub *Hub) Event {
	t.Helper()
	select {
	case event := <-hub.Events():
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for relay event")
		return Event{}
	}
}
