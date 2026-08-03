package api

import (
	"bufio"
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

func TestRandomRequestIDIsDirectBytes32(t *testing.T) {
	id, err := randomID()
	if err != nil || len(id) != 66 || !strings.HasPrefix(id, "0x") {
		t.Fatalf("id=%q err=%v", id, err)
	}
}

func TestOpenAIStreamingUsesRealRelayAndPersistsProposal(t *testing.T) {
	hub := relay.NewHub(func(context.Context, string) (string, error) { return "machine-1", nil }, relay.Options{HeartbeatTimeout: time.Second})
	relayServer := httptest.NewTLSServer(hub)
	defer relayServer.Close()
	relayClient := relayServer.Client()
	relayClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true} // test certificate
	provider, _, err := websocket.Dial(context.Background(), "wss"+strings.TrimPrefix(relayServer.URL, "https"), &websocket.DialOptions{HTTPClient: relayClient, HTTPHeader: http.Header{"Authorization": {"Bearer machine-token"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer provider.Close(websocket.StatusNormalClosure, "")
	requireRelayConnection(t, hub, "machine-1")
	writeProvider(t, provider, "hello", v1.MessageHello, &v1.Hello{MachineID: "machine-1"})
	writeProvider(t, provider, "capacity", v1.MessageCapacity, &v1.Capacity{Available: 1, Offers: []v1.OfferCapacity{{OfferID: "offer-1", Model: "qwen", PriceVersion: 3}}})

	proposals := make(chan Proposal, 1)
	workspaces := make(chan []v1.WorkspaceFile, 1)
	transitions := make(chan string, 2)
	aborts := make(chan string, 1)
	handler := NewOpenAI(Dependencies{
		Hub: hub,
		Authorize: func(_ context.Context, token, model, endpoint string, maximum uint64) (Principal, error) {
			if token != "api-token" || model != "qwen" || endpoint != "/v1/chat/completions" || maximum != 100 {
				t.Fatalf("unexpected authorization input: %q %q %q %d", token, model, endpoint, maximum)
			}
			return Principal{AccountID: "account-1", SessionID: "session-1", SessionBalance: 100}, nil
		},
		Candidates: func(context.Context, string) ([]router.Candidate, error) {
			return []router.Candidate{{MachineID: "machine-1", OfferID: "offer-1", Model: "qwen", Capabilities: []string{"text", "stream", "workspace"}, ConfirmedBond: true, Healthy: true, Capacity: 1, MaximumCost: 80, PriceVersion: 3, InputPerMillion: 1, OutputPerMillion: 1}}, nil
		},
		Reserve: func(_ context.Context, reservation Reservation) error {
			if reservation.SessionID != "session-1" || reservation.OfferID != "offer-1" || reservation.MaximumSpend != 2 {
				t.Fatalf("reservation=%+v", reservation)
			}
			return nil
		},
		Transition: func(_ context.Context, _ string, state string) error { transitions <- state; return nil },
		Abort:      func(_ context.Context, _ string, state string) error { aborts <- state; return nil },
		Persist:    func(_ context.Context, proposal Proposal) error { proposals <- proposal; return nil },
	})
	apiServer := httptest.NewServer(handler)
	defer apiServer.Close()

	go func() {
		_, payload, readErr := provider.Read(context.Background())
		if readErr != nil {
			return
		}
		var envelope v1.Envelope
		if json.Unmarshal(payload, &envelope) != nil || envelope.Type != v1.MessageJobOffer {
			return
		}
		var offer v1.JobOffer
		if envelope.DecodeBody(&offer) != nil {
			return
		}
		workspaces <- offer.Workspace
		writeProvider(t, provider, "accept", v1.MessageJobAccept, &v1.JobAccept{RequestID: offer.RequestID})
		writeProvider(t, provider, "chunk-1", v1.MessageOutputChunk, &v1.OutputChunk{RequestID: offer.RequestID, Sequence: 1, Data: "hello"})
		writeProvider(t, provider, "chunk-2", v1.MessageOutputChunk, &v1.OutputChunk{RequestID: offer.RequestID, Sequence: 2, Done: true, InputTokens: 4, OutputTokens: 1, ComputeMilliseconds: 12})
	}()

	body := `{"model":"qwen","stream":true,"messages":[{"role":"user","content":"say hello"}],"myference_workspace":[{"path":"src/main.go","content_base64":"cGFja2FnZSBtYWlu"}]}`
	request, _ := http.NewRequest(http.MethodPost, apiServer.URL+"/v1/chat/completions", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer api-token")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Myference-Max-Spend", "100")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, _ := io.ReadAll(response.Body)
	text := string(data)
	if response.StatusCode != http.StatusOK || !strings.Contains(response.Header.Get("Content-Type"), "text/event-stream") || !strings.Contains(text, `"content":"hello"`) || !strings.HasSuffix(text, "data: [DONE]\n\n") {
		t.Fatalf("status=%d headers=%v body=%q", response.StatusCode, response.Header, text)
	}
	requestID := response.Header.Get("X-Request-ID")
	if requestID == "" || !strings.Contains(text, requestID) {
		t.Fatalf("request ID not propagated: %q %q", requestID, text)
	}
	workspace := <-workspaces
	if len(workspace) != 1 || workspace[0].Path != "src/main.go" || workspace[0].ContentBase64 != "cGFja2FnZSBtYWlu" {
		t.Fatalf("workspace not relayed: %+v", workspace)
	}
	select {
	case proposal := <-proposals:
		if proposal.RequestID != requestID || proposal.InputTokens != 4 || proposal.OutputTokens != 1 || proposal.ComputeMilliseconds != 12 {
			t.Fatalf("proposal=%+v", proposal)
		}
	case <-time.After(time.Second):
		t.Fatal("receipt proposal was not persisted")
	}
	if first, second := <-transitions, <-transitions; first != "accepted" || second != "streaming" {
		t.Fatalf("transitions=%q,%q", first, second)
	}

	// SSE frames must remain parseable and ordered.
	scanner := bufio.NewScanner(strings.NewReader(text))
	frames := 0
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "data: ") {
			frames++
		}
	}
	if frames != 3 {
		t.Fatalf("SSE frames=%d body=%q", frames, text)
	}

	cancelSeen := make(chan struct{}, 1)
	go func() {
		_, payload, readErr := provider.Read(context.Background())
		if readErr != nil {
			return
		}
		var envelope v1.Envelope
		if json.Unmarshal(payload, &envelope) != nil {
			return
		}
		var offer v1.JobOffer
		if envelope.DecodeBody(&offer) != nil {
			return
		}
		writeProvider(t, provider, "accept-cancel", v1.MessageJobAccept, &v1.JobAccept{RequestID: offer.RequestID})
		_, payload, readErr = provider.Read(context.Background())
		if readErr == nil && json.Unmarshal(payload, &envelope) == nil && envelope.Type == v1.MessageCancel {
			cancelSeen <- struct{}{}
		}
	}()
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancelRequest, _ := http.NewRequestWithContext(cancelCtx, http.MethodPost, apiServer.URL+"/v1/chat/completions", strings.NewReader(body))
	cancelRequest.Header.Set("Authorization", "Bearer api-token")
	cancelRequest.Header.Set("Content-Type", "application/json")
	cancelRequest.Header.Set("X-Myference-Max-Spend", "100")
	cancelResponse, err := http.DefaultClient.Do(cancelRequest)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	cancelResponse.Body.Close()
	select {
	case <-cancelSeen:
	case <-time.After(time.Second):
		t.Fatal("provider did not receive cancellation")
	}
	select {
	case state := <-aborts:
		if state != "cancelled" {
			t.Fatalf("abort state=%q", state)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled reservation was not released")
	}
}

func writeProvider(t *testing.T, connection *websocket.Conn, id, messageType string, message v1.Validatable) {
	t.Helper()
	envelope, err := v1.NewEnvelope(id, messageType, message)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(envelope)
	if err := connection.Write(context.Background(), websocket.MessageText, payload); err != nil {
		t.Fatal(err)
	}
}

func requireRelayConnection(t *testing.T, hub *relay.Hub, machineID string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for !hub.Connected(machineID) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !hub.Connected(machineID) {
		t.Fatalf("relay did not register machine %q", machineID)
	}
}
