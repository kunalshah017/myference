package auth

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"time"
)

type StreamTicket struct {
	Token     string    `json:"ticket"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (s *Service) CreateStreamTicket(ctx context.Context, accountID string, lifetime time.Duration) (StreamTicket, error) {
	if accountID == "" || lifetime <= 0 {
		return StreamTicket{}, errors.New("account and positive lifetime are required")
	}
	id, err := randomID("stream")
	if err != nil {
		return StreamTicket{}, err
	}
	secret, err := randomToken(24)
	if err != nil {
		return StreamTicket{}, err
	}
	ticket := StreamTicket{Token: id + "." + secret, ExpiresAt: s.now().UTC().Truncate(time.Microsecond).Add(lifetime)}
	_, err = s.db.ExecContext(ctx, `INSERT INTO stream_tickets (id,account_id,token_hash,expires_at) VALUES ($1,$2,$3,$4)`, id, accountID, digest(secret), ticket.ExpiresAt)
	return ticket, err
}

func (s *Service) ConsumeStreamTicket(ctx context.Context, token string) (string, error) {
	id, secret, ok := splitCredential(token)
	if !ok {
		return "", ErrInvalidCredential
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var accountID string
	var stored []byte
	var expiresAt time.Time
	var consumed sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT account_id,token_hash,expires_at,consumed_at FROM stream_tickets WHERE id=$1 FOR UPDATE`, id).Scan(&accountID, &stored, &expiresAt, &consumed)
	if errors.Is(err, sql.ErrNoRows) || err == nil && subtle.ConstantTimeCompare(stored, digest(secret)) != 1 {
		return "", ErrInvalidCredential
	}
	if err != nil {
		return "", err
	}
	if consumed.Valid {
		return "", ErrAuthorizationConsumed
	}
	if !s.now().Before(expiresAt) {
		return "", ErrAuthorizationExpired
	}
	if _, err := tx.ExecContext(ctx, `UPDATE stream_tickets SET consumed_at=$2 WHERE id=$1`, id, s.now()); err != nil {
		return "", err
	}
	return accountID, tx.Commit()
}
