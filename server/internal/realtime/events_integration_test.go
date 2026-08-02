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
	events, err := Open(ctx, databaseURL, func(_ context.Context, token string) error {
		if token != "stream-ticket" {
			return errors.New("invalid ticket")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer events.Close()
	if _, err := events.db.ExecContext(ctx, `TRUNCATE outbox RESTART IDENTITY`); err != nil {
		t.Fatal(err)
	}
	if _, err := events.db.ExecContext(ctx, `INSERT INTO outbox (aggregate_type,aggregate_id,event_type,payload) VALUES ('request','request-1','request.completed','{"state":"completed"}')`); err != nil {
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
	request, _ := http.NewRequestWithContext(streamCtx, http.MethodGet, server.URL, nil)
	request.Header.Set("Authorization", "Bearer stream-ticket")
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
	if text := frame.String(); !strings.Contains(text, "id: 1") || !strings.Contains(text, "event: request.completed") || !strings.Contains(text, `data: {"state": "completed"}`) && !strings.Contains(text, `data: {"state":"completed"}`) {
		t.Fatalf("frame=%q", text)
	}
}
