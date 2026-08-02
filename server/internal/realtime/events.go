package realtime

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Authorize func(context.Context, string) error

type Events struct {
	db        *sql.DB
	authorize Authorize
	poll      time.Duration
}

func Open(ctx context.Context, databaseURL string, authorize Authorize) (*Events, error) {
	if authorize == nil {
		return nil, errors.New("realtime authorization is required")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return &Events{db: db, authorize: authorize, poll: time.Second}, nil
}

func (e *Events) Close() error { return e.db.Close() }

func (e *Events) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	token := strings.TrimSpace(strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer "))
	if token == "" || e.authorize(request.Context(), token) != nil {
		http.Error(response, "unauthorized", http.StatusUnauthorized)
		return
	}
	flusher, ok := response.(http.Flusher)
	if !ok {
		http.Error(response, "streaming unavailable", http.StatusInternalServerError)
		return
	}
	cursor, _ := strconv.ParseInt(request.Header.Get("Last-Event-ID"), 10, 64)
	response.Header().Set("Content-Type", "text/event-stream")
	response.Header().Set("Cache-Control", "no-cache")
	response.WriteHeader(http.StatusOK)
	flusher.Flush()
	ticker := time.NewTicker(e.poll)
	defer ticker.Stop()
	for {
		next, err := e.writeAvailable(request.Context(), response, cursor)
		if err != nil {
			return
		}
		if next > cursor {
			cursor = next
			flusher.Flush()
			continue
		}
		select {
		case <-request.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

func (e *Events) writeAvailable(ctx context.Context, writer http.ResponseWriter, cursor int64) (int64, error) {
	rows, err := e.db.QueryContext(ctx, `SELECT id,event_type,payload FROM outbox WHERE id>$1 ORDER BY id LIMIT 100`, cursor)
	if err != nil {
		return cursor, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var eventType string
		var payload json.RawMessage
		if err := rows.Scan(&id, &eventType, &payload); err != nil {
			return cursor, err
		}
		if _, err := fmt.Fprintf(writer, "id: %d\nevent: %s\ndata: %s\n\n", id, eventType, payload); err != nil {
			return cursor, err
		}
		cursor = id
	}
	return cursor, rows.Err()
}
