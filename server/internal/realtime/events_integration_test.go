//go:build integration

package realtime

import (
	"bufio"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kunalshah017/myference/server/internal/store"
)

func TestEventsAuthenticatesAndStreamsDurableOutboxCursor(t *testing.T) {
	databaseURL := os.Getenv("MYFERENCE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("MYFERENCE_TEST_DATABASE_URL is required")
	}
	ctx := context.Background()
	repository, err := store.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.ApplyMigration(ctx, filepath.Join("..", "..", "..", "migrations", "000001_control_plane.sql")); err != nil {
		t.Fatal(err)
	}
	repository.Close()
	events, err := Open(ctx, databaseURL, func(_ context.Context, token string) (string, error) {
		if token != "stream-ticket" {
			return "", errors.New("invalid ticket")
		}
		return "acct-stream-1", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer events.Close()
	if _, err := events.db.ExecContext(ctx, `TRUNCATE outbox,requests,sessions,accounts CASCADE`); err != nil {
		t.Fatal(err)
	}
	if _, err := events.db.ExecContext(ctx, `INSERT INTO accounts(id,wallet_address) VALUES ('acct-stream-1','0x1111111111111111111111111111111111111111'),('acct-stream-2','0x2222222222222222222222222222222222222222'); INSERT INTO sessions(id,account_id,state) VALUES ('session-stream-1','acct-stream-1','open'),('session-stream-2','acct-stream-2','open'); INSERT INTO requests(id,session_id,state) VALUES ('request-other','session-stream-2','completed'),('request-own','session-stream-1','completed'); INSERT INTO outbox (aggregate_type,aggregate_id,event_type,payload) VALUES ('request','request-other','request.completed','{"request":"other"}'),('request','request-own','request.completed','{"request":"own"}')`); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(events)
	defer server.Close()
	unauthorized, _ := http.Get(server.URL)
	if unauthorized.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized status=%d", unauthorized.StatusCode)
	}
	unauthorized.Body.Close()
	streamCtx, cancel := context.WithCancel(context.Background())
	request, _ := http.NewRequestWithContext(streamCtx, http.MethodGet, server.URL+"?ticket=stream-ticket", nil)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(response.Body)
	var frame strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		frame.WriteString(line)
		if line == "\n" {
			break
		}
	}
	cancel()
	response.Body.Close()
	if text := frame.String(); !strings.Contains(text, "id: ") || !strings.Contains(text, "event: request.completed") || strings.Contains(text, "other") || !strings.Contains(text, "own") {
		t.Fatalf("frame=%q", text)
	}
}
