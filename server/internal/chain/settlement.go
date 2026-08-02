package chain

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/ethereum/go-ethereum/core/types"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var ErrNoSignedReceipts = errors.New("no co-signed receipts")

type SignedReceipt struct {
	Receipt             Receipt
	ProviderSignature   []byte
	SettlementSignature []byte
}

type SettlementQueue struct{ db *sql.DB }

func OpenSettlementQueue(ctx context.Context, databaseURL string) (*SettlementQueue, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return &SettlementQueue{db: db}, nil
}

func (q *SettlementQueue) Close() error { return q.db.Close() }

func (q *SettlementQueue) Enqueue(ctx context.Context, signed SignedReceipt) error {
	payload, err := json.Marshal(signed.Receipt)
	if err != nil {
		return err
	}
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	requestID := hashHex(signed.Receipt.RequestId)
	result, err := tx.ExecContext(ctx, `INSERT INTO settlement_queue (request_id,receipt_json,provider_signature,settlement_signature,state) VALUES ($1,$2,$3,$4,'cosigned') ON CONFLICT (request_id) DO NOTHING`, requestID, payload, signed.ProviderSignature, signed.SettlementSignature)
	if err != nil {
		return err
	}
	if inserted, _ := result.RowsAffected(); inserted == 1 {
		if err := transitionSettlementRequest(ctx, tx, requestID, "completed", "signed"); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (q *SettlementQueue) SettleBatch(ctx context.Context, client *Client, limit int) (string, error) {
	if limit <= 0 {
		return "", ErrNoSignedReceipts
	}
	if hash, transaction, err := q.pendingBroadcast(ctx); err != nil {
		return "", err
	} else if transaction != nil {
		if err := client.Broadcast(ctx, transaction); err != nil {
			return hash, err
		}
		return hash, nil
	}
	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT request_id,receipt_json,provider_signature,settlement_signature FROM settlement_queue WHERE state='cosigned' ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT $1`, limit)
	if err != nil {
		return "", err
	}
	var ids []string
	var receipts []Receipt
	var providerSignatures, settlementSignatures [][]byte
	for rows.Next() {
		var id string
		var payload, providerSignature, settlementSignature []byte
		if err := rows.Scan(&id, &payload, &providerSignature, &settlementSignature); err != nil {
			rows.Close()
			return "", err
		}
		var receipt Receipt
		if err := json.Unmarshal(payload, &receipt); err != nil {
			rows.Close()
			return "", err
		}
		ids, receipts = append(ids, id), append(receipts, receipt)
		providerSignatures, settlementSignatures = append(providerSignatures, providerSignature), append(settlementSignatures, settlementSignature)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", err
	}
	if err := rows.Close(); err != nil {
		return "", err
	}
	if len(receipts) == 0 {
		return "", ErrNoSignedReceipts
	}
	transaction, err := client.PrepareSettlement(ctx, receipts, providerSignatures, settlementSignatures)
	if err != nil {
		return "", err
	}
	hash := transaction.Hash().Hex()
	rawTransaction, err := transaction.MarshalBinary()
	if err != nil {
		return "", err
	}
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, `UPDATE settlement_queue SET state='broadcasting',transaction_hash=$2,raw_transaction=$3,updated_at=now() WHERE request_id=$1`, id, hash, rawTransaction); err != nil {
			return "", err
		}
		if err := transitionSettlementRequest(ctx, tx, id, "signed", "submitted"); err != nil {
			return "", err
		}
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	if err := client.Broadcast(ctx, transaction); err != nil {
		return hash, err
	}
	return hash, nil
}

func transitionSettlementRequest(ctx context.Context, tx *sql.Tx, requestID, from, to string) error {
	result, err := tx.ExecContext(ctx, `UPDATE requests SET state=$3,updated_at=now() WHERE id=$1 AND state=$2`, requestID, from, to)
	if err != nil {
		return err
	}
	if changed, _ := result.RowsAffected(); changed == 1 {
		_, err = tx.ExecContext(ctx, `INSERT INTO outbox (aggregate_type,aggregate_id,event_type,payload) VALUES ('request',$1,$2,jsonb_build_object('from',$3::text,'to',$4::text))`, requestID, "request."+to, from, to)
	}
	return err
}

func (q *SettlementQueue) pendingBroadcast(ctx context.Context) (string, *types.Transaction, error) {
	var hash string
	var raw []byte
	err := q.db.QueryRowContext(ctx, `SELECT transaction_hash,raw_transaction FROM settlement_queue WHERE state='broadcasting' AND raw_transaction IS NOT NULL ORDER BY updated_at LIMIT 1`).Scan(&hash, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, err
	}
	var transaction types.Transaction
	if err := transaction.UnmarshalBinary(raw); err != nil {
		return "", nil, err
	}
	return hash, &transaction, nil
}

func hashHex(value [32]byte) string {
	const digits = "0123456789abcdef"
	result := make([]byte, 66)
	result[0], result[1] = '0', 'x'
	for index, item := range value {
		result[2+index*2], result[3+index*2] = digits[item>>4], digits[item&15]
	}
	return string(result)
}
