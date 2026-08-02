//go:build integration

package api

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	providerclient "github.com/kunalshah017/myference/cli/provider"
	v1 "github.com/kunalshah017/myference/protocol/v1"
	"github.com/kunalshah017/myference/server/internal/relay"
	"github.com/kunalshah017/myference/server/internal/router"
)

func TestOpenAIStreamsFromActualProviderDaemonAndRealOllama(t *testing.T) {
	model := os.Getenv("MYFERENCE_TEST_OLLAMA_MODEL")
	if model == "" {
		t.Fatal("MYFERENCE_TEST_OLLAMA_MODEL is required")
	}
	hub := relay.NewHub(func(context.Context, string) (string, error) { return "machine-real", nil }, relay.Options{HeartbeatTimeout: time.Second})
	relayServer := httptest.NewTLSServer(hub)
	defer relayServer.Close()
	client := relayServer.Client()
	client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true} // test certificate
	offer := v1.OfferCapacity{OfferID: "real-ollama", Model: model, PriceVersion: 1}
	daemon, err := providerclient.NewOllama(providerclient.Config{RelayURL: "wss" + strings.TrimPrefix(relayServer.URL, "https"), Token: "machine-token", MachineID: "machine-real", Offers: []v1.OfferCapacity{offer}, HTTPClient: client}, map[string]string{"real-ollama": "http://127.0.0.1:11434"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go daemon.Serve(ctx)
	deadline := time.Now().Add(time.Second)
	for !hub.Connected("machine-real") && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	persisted := make(chan Proposal, 1)
	handler := NewOpenAI(Dependencies{
		Hub: hub,
		Authorize: func(context.Context, string, string, string, uint64) (Principal, error) {
			return Principal{AccountID: "account", SessionID: "session", SessionBalance: 100}, nil
		},
		Candidates: func(context.Context, string) ([]router.Candidate, error) {
			return []router.Candidate{{MachineID: "machine-real", OfferID: offer.OfferID, Model: model, Capabilities: []string{"text", "stream"}, ConfirmedBond: true, Healthy: true, Capacity: 1, MaximumCost: 10, PriceVersion: 1}}, nil
		},
		Reserve:    func(context.Context, Reservation) error { return nil },
		Transition: func(context.Context, string, string) error { return nil },
		Abort:      func(context.Context, string, string) error { return nil },
		Persist:    func(_ context.Context, proposal Proposal) error { persisted <- proposal; return nil },
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	request, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/chat/completions", strings.NewReader(`{"model":"`+model+`","stream":true,"messages":[{"role":"user","content":"Reply with exactly OK"}]}`))
	request.Header.Set("Authorization", "Bearer real")
	request.Header.Set("X-Myference-Max-Spend", "100")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(string(data), "data: [DONE]") {
		t.Fatalf("status=%d body=%q", response.StatusCode, data)
	}
	select {
	case proposal := <-persisted:
		if proposal.OutputTokens == 0 || proposal.ComputeMilliseconds == 0 {
			t.Fatalf("real usage missing: %+v", proposal)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("real receipt proposal missing")
	}
}
