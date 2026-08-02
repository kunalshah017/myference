package provider

import (
	"bytes"
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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/kunalshah017/myference/cli/internal/backend"
	v1 "github.com/kunalshah017/myference/protocol/v1"
)

func TestDaemonSignsOnlyPinnedReceiptDomain(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	var contract v1.Address
	contract[19] = 9
	receipt := validDaemonReceipt(v1.Address(crypto.PubkeyToAddress(key.PublicKey)))
	daemon := NewDaemon(Config{SignerKey: key, ChainID: 10143, Contract: contract}, nil)
	signed, err := daemon.signProposal(v1.ReceiptProposal{RequestID: "request-1", ChainID: 10143, Contract: contract, Receipt: receipt})
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := v1.RecoverReceiptSigner(receipt, 10143, contract, signed.Signature)
	if err != nil || recovered != signed.Signer {
		t.Fatalf("recovered=%x signed=%x err=%v", recovered, signed.Signer, err)
	}
	other := contract
	other[19]++
	if _, err := daemon.signProposal(v1.ReceiptProposal{RequestID: "request-1", ChainID: 10143, Contract: other, Receipt: receipt}); err == nil {
		t.Fatal("accepted unpinned receipt domain")
	}
}

func validDaemonReceipt(provider v1.Address) v1.Receipt {
	hash := func(seed byte) v1.Hash { var value v1.Hash; value[0] = seed; return value }
	address := func(seed byte) v1.Address { var value v1.Address; value[19] = seed; return value }
	return v1.Receipt{RequestID: hash(1), SessionID: hash(2), Customer: address(1), Provider: provider, SettlementSigner: address(2), OfferID: hash(3), PriceVersion: 1, ModelHash: hash(4), CapabilityHash: hash(5), InputTokens: 1, OutputTokens: 1, MaximumCharge: 10, TotalCharge: 2, FeeBasisPoints: 500, FeeVersion: 1, Status: v1.ReceiptStatusCompleted, CompletedAt: 1, InputHash: hash(6), OutputHash: hash(7), Nonce: 1}
}

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

func TestDaemonConnectsToRealTLSRelayAndStreamsBackend(t *testing.T) {
	events := make(chan v1.Envelope, 4)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		connection, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer connection.Close(websocket.StatusNormalClosure, "test complete")
		for range 3 {
			_, payload, err := connection.Read(r.Context())
			if err != nil {
				return
			}
			envelope, err := v1.DecodeEnvelope(bytes.NewReader(payload), 1<<20)
			if err != nil {
				t.Errorf("decode handshake: %v", err)
				return
			}
			events <- envelope
		}
		offer, err := v1.NewEnvelope("offer", v1.MessageJobOffer, &v1.JobOffer{RequestID: "request-1", Model: "model", OfferID: "local", PriceVersion: 1, MaximumSpend: 10, LeaseExpiresAt: time.Now().Add(time.Minute), Prompt: "real prompt"})
		if err != nil {
			t.Errorf("create offer: %v", err)
			return
		}
		payload, _ := json.Marshal(offer)
		if err := connection.Write(r.Context(), websocket.MessageText, payload); err != nil {
			return
		}
		for range 3 {
			_, payload, err := connection.Read(r.Context())
			if err != nil {
				return
			}
			envelope, err := v1.DecodeEnvelope(bytes.NewReader(payload), 1<<20)
			if err != nil {
				t.Errorf("decode event: %v", err)
				return
			}
			events <- envelope
		}
	}))
	defer server.Close()
	client := server.Client()
	client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true} // test certificate
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	daemon := NewDaemon(Config{
		RelayURL:          "wss" + strings.TrimPrefix(server.URL, "https"),
		Token:             "token",
		MachineID:         "machine-1",
		Offers:            []v1.OfferCapacity{{OfferID: "local", Model: "model", PriceVersion: 1}},
		HTTPClient:        client,
		HeartbeatInterval: 10 * time.Millisecond,
	}, map[string]backend.Backend{"local": testBackend{}})
	done := make(chan error, 1)
	go func() { done <- daemon.Serve(ctx) }()
	seen := map[string]int{}
	for seen[v1.MessageOutputChunk] < 2 || seen[v1.MessageJobAccept] < 1 {
		select {
		case event := <-events:
			seen[event.Type]++
		case <-time.After(time.Second):
			t.Fatalf("timed out, events=%v", seen)
		}
	}
	if seen[v1.MessageJobAccept] != 1 {
		t.Fatalf("accept count=%d", seen[v1.MessageJobAccept])
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("serve returned %v", err)
	}
}

type testBackend struct{}

func (testBackend) Models(context.Context) ([]backend.Model, error) {
	return []backend.Model{{Name: "model"}}, nil
}
func (testBackend) Generate(_ context.Context, request backend.Request, emit func(string) error) (backend.Usage, error) {
	if request.Prompt != "real prompt" {
		return backend.Usage{}, errors.New("prompt mismatch")
	}
	if err := emit("hello"); err != nil {
		return backend.Usage{}, err
	}
	return backend.Usage{InputTokens: 1, OutputTokens: 1, ComputeMilliseconds: 1}, nil
}
