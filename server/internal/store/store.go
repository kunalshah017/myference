package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/kunalshah017/myference/server/internal/pg"
)

var (
	ErrInvalidTransition = errors.New("invalid request transition")
	ErrNoOutboxEvents    = errors.New("no outbox events")
)

type Store struct{ db *sql.DB }

type Account struct{ ID, WalletAddress string }
type Machine struct{ ID, AccountID, Name string }
type Backend struct{ ID, MachineID, Kind, Model string }
type Offer struct {
	ID, BackendID                                       string
	Version                                             int64
	InputPerMillion, OutputPerMillion, ComputePerSecond uint64
}
type Session struct{ ID, AccountID, State string }
type Request struct{ ID, SessionID, State string }
type OutboxEvent struct {
	ID                                    int64
	AggregateType, AggregateID, EventType string
	Payload                               json.RawMessage
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	db, err := pg.Open(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) ApplyMigration(ctx context.Context, path string) error {
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, string(sqlBytes))
	return err
}

func (s *Store) CreateAccount(ctx context.Context, account Account) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO accounts (id, wallet_address) VALUES ($1, $2)`, account.ID, account.WalletAddress)
	return err
}

func (s *Store) CreateMachine(ctx context.Context, machine Machine) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO machines (id, account_id, name) VALUES ($1, $2, $3)`, machine.ID, machine.AccountID, machine.Name)
	return err
}

func (s *Store) CreateBackend(ctx context.Context, backend Backend) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO backends (id, machine_id, kind, model) VALUES ($1, $2, $3, $4)`, backend.ID, backend.MachineID, backend.Kind, backend.Model)
	return err
}

func (s *Store) CreateOffer(ctx context.Context, offer Offer) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO offers (id, backend_id, version, input_per_million, output_per_million, compute_per_second) VALUES ($1, $2, $3, $4, $5, $6)`, offer.ID, offer.BackendID, offer.Version, offer.InputPerMillion, offer.OutputPerMillion, offer.ComputePerSecond)
	return err
}

func (s *Store) CreateSession(ctx context.Context, session Session) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO sessions (id, account_id, state) VALUES ($1, $2, $3)`, session.ID, session.AccountID, session.State)
	return err
}

func (s *Store) CreateRequest(ctx context.Context, request Request) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO requests (id, session_id, state) VALUES ($1, $2, $3)`, request.ID, request.SessionID, request.State)
	return err
}

var transitions = map[string]map[string]bool{
	"created":   {"reserved": true, "cancelled": true, "failed": true},
	"reserved":  {"offered": true, "cancelled": true, "expired": true, "failed": true},
	"offered":   {"accepted": true, "rejected": true, "expired": true, "cancelled": true},
	"accepted":  {"streaming": true, "cancelled": true, "failed": true},
	"streaming": {"completed": true, "cancelled": true, "failed": true},
	"completed": {"signed": true, "failed": true},
	"signed":    {"submitted": true},
	"submitted": {"settled": true},
}

func (s *Store) TransitionRequest(ctx context.Context, requestID, next string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, requestID); err != nil {
		return err
	}
	var current string
	if err := tx.QueryRowContext(ctx, `SELECT state FROM requests WHERE id = $1 FOR UPDATE`, requestID).Scan(&current); err != nil {
		return err
	}
	if !transitions[current][next] {
		return ErrInvalidTransition
	}
	if _, err := tx.ExecContext(ctx, `UPDATE requests SET state = $2, updated_at = now() WHERE id = $1`, requestID, next); err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]string{"from": current, "to": next})
	if _, err := tx.ExecContext(ctx, `INSERT INTO outbox (aggregate_type, aggregate_id, event_type, payload) VALUES ('request', $1, $2, $3)`, requestID, "request."+next, payload); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ClaimOutbox(ctx context.Context) (OutboxEvent, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return OutboxEvent{}, err
	}
	defer tx.Rollback()
	var event OutboxEvent
	err = tx.QueryRowContext(ctx, `SELECT id, aggregate_type, aggregate_id, event_type, payload FROM outbox WHERE published_at IS NULL ORDER BY id FOR UPDATE SKIP LOCKED LIMIT 1`).Scan(&event.ID, &event.AggregateType, &event.AggregateID, &event.EventType, &event.Payload)
	if errors.Is(err, sql.ErrNoRows) {
		return OutboxEvent{}, ErrNoOutboxEvents
	}
	if err != nil {
		return OutboxEvent{}, fmt.Errorf("claim outbox: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE outbox SET published_at = now() WHERE id = $1`, event.ID); err != nil {
		return OutboxEvent{}, err
	}
	if err := tx.Commit(); err != nil {
		return OutboxEvent{}, err
	}
	return event, nil
}
