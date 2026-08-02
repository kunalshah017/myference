package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRequestSettlementLifecycleIncludesSubmittedBeforeConfirmed(t *testing.T) {
	databaseURL := os.Getenv("MYFERENCE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("MYFERENCE_TEST_DATABASE_URL is required")
	}
	ctx := context.Background()
	repository, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repository.Close() })
	for _, migration := range []string{"000001_control_plane.sql", "000006_request_submission.sql"} {
		if err := repository.ApplyMigration(ctx, filepath.Join("..", "..", "..", "migrations", migration)); err != nil {
			t.Fatal(err)
		}
	}
	suffix := time.Now().Format("150405.000000000")
	accountID, sessionID, requestID := "activity-account-"+suffix, "activity-session-"+suffix, "activity-request-"+suffix
	if err := repository.CreateAccount(ctx, Account{ID: accountID, WalletAddress: "activity-wallet-" + suffix}); err != nil {
		t.Fatal(err)
	}
	if err := repository.CreateSession(ctx, Session{ID: sessionID, AccountID: accountID, State: "open"}); err != nil {
		t.Fatal(err)
	}
	if err := repository.CreateRequest(ctx, Request{ID: requestID, SessionID: sessionID, State: "completed"}); err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{"signed", "submitted", "settled"} {
		if err := repository.TransitionRequest(ctx, requestID, state); err != nil {
			t.Fatalf("transition to %s: %v", state, err)
		}
	}
	var states []string
	rows, err := repository.db.QueryContext(ctx, `SELECT event_type FROM outbox WHERE aggregate_id=$1 ORDER BY id`, requestID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var state string
		if err := rows.Scan(&state); err != nil {
			t.Fatal(err)
		}
		states = append(states, state)
	}
	if len(states) != 3 || states[0] != "request.signed" || states[1] != "request.submitted" || states[2] != "request.settled" {
		t.Fatalf("unexpected lifecycle events: %v", states)
	}
}
