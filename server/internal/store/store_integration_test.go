package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestControlPlanePersistsGraphAndTransactionalOutbox(t *testing.T) {
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

	migration := filepath.Join("..", "..", "..", "migrations", "000001_control_plane.sql")
	if err := s.ApplyMigration(ctx, migration); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, "TRUNCATE outbox, requests, sessions, offers, backends, machines, accounts CASCADE"); err != nil {
		t.Fatal(err)
	}

	if err := s.CreateAccount(ctx, Account{ID: "acct-1", WalletAddress: "0x1111111111111111111111111111111111111111"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateMachine(ctx, Machine{ID: "machine-1", AccountID: "acct-1", Name: "windows-lab"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateBackend(ctx, Backend{ID: "backend-1", MachineID: "machine-1", Kind: "ollama", Model: "qwen3:8b"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateOffer(ctx, Offer{ID: "offer-1", BackendID: "backend-1", Version: 1, InputPerMillion: 10, OutputPerMillion: 20, ComputePerSecond: 30}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSession(ctx, Session{ID: "session-1", AccountID: "acct-1", State: "open"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateRequest(ctx, Request{ID: "request-1", SessionID: "session-1", State: "created"}); err != nil {
		t.Fatal(err)
	}

	if err := s.TransitionRequest(ctx, "request-1", "reserved"); err != nil {
		t.Fatal(err)
	}
	if err := s.TransitionRequest(ctx, "request-1", "failed"); err != nil {
		t.Fatal(err)
	}
	if err := s.TransitionRequest(ctx, "request-1", "streaming"); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}

	event, err := s.ClaimOutbox(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if event.AggregateID != "request-1" || event.EventType != "request.reserved" {
		t.Fatalf("unexpected outbox event: %+v", event)
	}
}
