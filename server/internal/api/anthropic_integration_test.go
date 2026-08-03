package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	v1 "github.com/kunalshah017/myference/protocol/v1"
	"github.com/kunalshah017/myference/server/internal/relay"
	"github.com/kunalshah017/myference/server/internal/router"
)

func TestAnthropicMessagesStreamsNativeEventsThroughRealRelay(t *testing.T) {
	hub := relay.NewHub(func(context.Context, string) (string, error) { return "machine-1", nil }, relay.Options{HeartbeatTimeout: time.Second})
	relayServer := httptest.NewTLSServer(hub)
	defer relayServer.Close()
	client := relayServer.Client()
	client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true}
	provider, _, err := websocket.Dial(context.Background(), "wss"+strings.TrimPrefix(relayServer.URL, "https"), &websocket.DialOptions{HTTPClient: client, HTTPHeader: http.Header{"Authorization": {"Bearer token"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer provider.Close(websocket.StatusNormalClosure, "")
	requireRelayConnection(t, hub, "machine-1")
	writeProvider(t, provider, "hello", v1.MessageHello, &v1.Hello{MachineID: "machine-1"})
	persisted := make(chan Proposal, 1)
	workspaces := make(chan []v1.WorkspaceFile, 1)
	openAI := NewOpenAI(Dependencies{Hub: hub, Authorize: func(_ context.Context, token, model, endpoint string, maximum uint64) (Principal, error) {
		if token != "anthropic-key" || model != "qwen" || endpoint != "/v1/messages" || maximum != 100 {
			t.Fatalf("authorization=%q %q %q %d", token, model, endpoint, maximum)
		}
		return Principal{AccountID: "account", SessionID: "session", SessionBalance: 100}, nil
	}, Candidates: func(context.Context, string) ([]router.Candidate, error) {
		return []router.Candidate{{MachineID: "machine-1", OfferID: "offer", Model: "qwen", Capabilities: []string{"text", "stream", "workspace"}, ConfirmedBond: true, Healthy: true, Capacity: 1, MaximumCost: 80, PriceVersion: 1}}, nil
	}, Reserve: func(context.Context, Reservation) error { return nil }, Transition: func(context.Context, string, string) error { return nil }, Abort: func(context.Context, string, string) error { return nil }, Persist: func(_ context.Context, p Proposal) error { persisted <- p; return nil }})
	server := httptest.NewServer(NewAnthropic(openAI))
	defer server.Close()
	go func() {
		_, payload, _ := provider.Read(context.Background())
		var envelope v1.Envelope
		_ = json.Unmarshal(payload, &envelope)
		var offer v1.JobOffer
		_ = envelope.DecodeBody(&offer)
		workspaces <- offer.Workspace
		writeProvider(t, provider, "accept", v1.MessageJobAccept, &v1.JobAccept{RequestID: offer.RequestID})
		writeProvider(t, provider, "chunk", v1.MessageOutputChunk, &v1.OutputChunk{RequestID: offer.RequestID, Sequence: 1, Data: "hello"})
		writeProvider(t, provider, "done", v1.MessageOutputChunk, &v1.OutputChunk{RequestID: offer.RequestID, Sequence: 2, Done: true, InputTokens: 4, OutputTokens: 1})
	}()
	body := `{"model":"qwen","max_tokens":64,"stream":true,"messages":[{"role":"user","content":"say hello"}],"myference_workspace":[{"path":"README.md","content_base64":"aGVsbG8="}]}`
	request, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/messages", strings.NewReader(body))
	request.Header.Set("x-api-key", "anthropic-key")
	request.Header.Set("anthropic-version", "2023-06-01")
	request.Header.Set("content-type", "application/json")
	request.Header.Set("x-myference-max-spend", "100")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	text := string(raw)
	for _, expected := range []string{"event: message_start", "event: content_block_start", "event: content_block_delta", `"text":"hello"`, "event: message_delta", "event: message_stop"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("missing %q in %q", expected, text)
		}
	}
	if response.StatusCode != http.StatusOK || response.Header.Get("x-request-id") == "" {
		t.Fatalf("status=%d headers=%v body=%q", response.StatusCode, response.Header, text)
	}
	workspace := <-workspaces
	if len(workspace) != 1 || workspace[0].Path != "README.md" {
		t.Fatalf("workspace not relayed: %+v", workspace)
	}
	select {
	case proposal := <-persisted:
		if proposal.InputTokens != 4 || proposal.OutputTokens != 1 {
			t.Fatalf("proposal=%+v", proposal)
		}
	case <-time.After(time.Second):
		t.Fatal("proposal not persisted")
	}
}

func TestAnthropicMessagesRequiresVersionAndStreaming(t *testing.T) {
	handler := NewAnthropic(http.NotFoundHandler())
	request := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"qwen","stream":false,"max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`))
	request.Header.Set("content-type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
}
