package realtime

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/kunalshah017/myference/server/internal/pg"
)

type Authorize func(context.Context, string) (string, error)

type Events struct {
	db        *sql.DB
	authorize Authorize
	poll      time.Duration
}

func Open(ctx context.Context, databaseURL string, authorize Authorize) (*Events, error) {
	if authorize == nil {
		return nil, errors.New("realtime authorization is required")
	}
	db, err := pg.Open(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return &Events{db: db, authorize: authorize, poll: time.Second}, nil
}

func (e *Events) Close() error { return e.db.Close() }

func (e *Events) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	token := request.URL.Query().Get("ticket")
	accountID, err := e.authorize(request.Context(), token)
	if token == "" || err != nil || accountID == "" {
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
	response.Header().Set("Referrer-Policy", "no-referrer")
	response.WriteHeader(http.StatusOK)
	flusher.Flush()
	ticker := time.NewTicker(e.poll)
	defer ticker.Stop()
	for {
		next, err := e.writeAvailable(request.Context(), response, cursor, accountID)
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

func (e *Events) writeAvailable(ctx context.Context, writer http.ResponseWriter, cursor int64, accountID string) (int64, error) {
	rows, err := e.db.QueryContext(ctx, `SELECT o.id,o.event_type,o.payload FROM outbox o JOIN requests r ON o.aggregate_type='request' AND r.id=o.aggregate_id JOIN sessions s ON s.id=r.session_id WHERE o.id>$1 AND s.account_id=$2 ORDER BY o.id LIMIT 100`, cursor, accountID)
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
