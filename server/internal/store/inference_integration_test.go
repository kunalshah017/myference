package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	v1 "github.com/kunalshah017/myference/protocol/v1"
)

func TestInferenceReservationRoutingAndReceiptAreDurableAndAtomic(t *testing.T) {
	databaseURL := os.Getenv("MYFERENCE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("MYFERENCE_TEST_DATABASE_URL is required for PostgreSQL integration")
	}
	ctx := context.Background()
	s, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	for _, name := range []string{"000001_control_plane.sql", "000002_inference.sql"} {
		if err := s.ApplyMigration(ctx, filepath.Join("..", "..", "..", "migrations", name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.db.ExecContext(ctx, "TRUNCATE receipt_proposals, inference_reservations, provider_routing_state, outbox, requests, sessions, offers, backends, machines, accounts CASCADE"); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateAccount(ctx, Account{ID: "customer", WalletAddress: "0x1111111111111111111111111111111111111111"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateAccount(ctx, Account{ID: "provider", WalletAddress: "0x2222222222222222222222222222222222222222"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSession(ctx, Session{ID: "session-1", AccountID: "customer", State: "open"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, "UPDATE sessions SET confirmed_balance_wei = 100 WHERE id = 'session-1'"); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateMachine(ctx, Machine{ID: "machine-1", AccountID: "provider", Name: "windows"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateBackend(ctx, Backend{ID: "backend-1", MachineID: "machine-1", Kind: "ollama", Model: "qwen"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateOffer(ctx, Offer{ID: "offer-1", BackendID: "backend-1", Version: 3, InputPerMillion: 1, OutputPerMillion: 1}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertRoutingState(ctx, RoutingState{MachineID: "machine-1", OfferID: "offer-1", Model: "qwen", Capabilities: []string{"text", "stream"}, PriceVersion: 3, ConfirmedBond: true, Healthy: true, Capacity: 1, MaximumCost: 60, LatencyMilliseconds: 20, SuccessBasisPoints: 9900, Reputation: 80}); err != nil {
		t.Fatal(err)
	}
	candidates, err := s.RoutingCandidates(ctx, "qwen")
	if err != nil || len(candidates) != 1 || !candidates[0].ConfirmedBond {
		t.Fatalf("candidates=%+v err=%v", candidates, err)
	}
	reservation := InferenceReservation{RequestID: "request-1", SessionID: "session-1", AccountID: "customer", MachineID: "machine-1", OfferID: "offer-1", PriceVersion: 3, MaximumSpend: 60}
	if err := s.ReserveInference(ctx, reservation); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateProviderCapacity(ctx, "machine-1", v1.Capacity{Available: 1, Offers: []v1.OfferCapacity{{OfferID: "offer-1", Model: "qwen", PriceVersion: 3}}}); err != nil {
		t.Fatal(err)
	}
	candidates, err = s.RoutingCandidates(ctx, "qwen")
	if err != nil || candidates[0].Capacity != 0 {
		t.Fatalf("heartbeat reopened reserved capacity: %+v err=%v", candidates, err)
	}
	reservation.RequestID = "request-2"
	reservation.MaximumSpend = 50
	if err := s.ReserveInference(ctx, reservation); !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("expected balance rejection, got %v", err)
	}
	if err := s.TransitionRequest(ctx, "request-1", "accepted"); err != nil {
		t.Fatal(err)
	}
	if err := s.TransitionRequest(ctx, "request-1", "streaming"); err != nil {
		t.Fatal(err)
	}
	completed := time.Now().UTC().Truncate(time.Microsecond)
	if err := s.CompleteInference(ctx, ReceiptProposal{RequestID: "request-1", SessionID: "session-1", MachineID: "machine-1", OfferID: "offer-1", Model: "qwen", PriceVersion: 3, InputTokens: 4, OutputTokens: 1, ComputeMilliseconds: 12, InputHash: [32]byte{1}, OutputHash: [32]byte{2}, CompletedAt: completed}); err != nil {
		t.Fatal(err)
	}
	var state string
	var proposals, activeReservations int
	if err := s.db.QueryRowContext(ctx, "SELECT state FROM requests WHERE id = 'request-1'").Scan(&state); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM receipt_proposals WHERE request_id = 'request-1'").Scan(&proposals); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM inference_reservations WHERE request_id = 'request-1' AND released_at IS NULL").Scan(&activeReservations); err != nil {
		t.Fatal(err)
	}
	if state != "completed" || proposals != 1 || activeReservations != 0 {
		t.Fatalf("state=%s proposals=%d active=%d", state, proposals, activeReservations)
	}
	if err := s.UpdateProviderCapacity(ctx, "machine-1", v1.Capacity{}); err != nil {
		t.Fatal(err)
	}
	candidates, err = s.RoutingCandidates(ctx, "qwen")
	if err != nil || len(candidates) != 1 || candidates[0].Capacity != 0 {
		t.Fatalf("offline capacity was not persisted: %+v err=%v", candidates, err)
	}
}
