package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type MarketModel struct {
	Model                   string `json:"model"`
	AvailableProviders      uint64 `json:"available_providers"`
	TotalCapacity           uint64 `json:"total_capacity"`
	MinimumInputPerMillion  string `json:"minimum_input_per_million_wei"`
	MinimumOutputPerMillion string `json:"minimum_output_per_million_wei"`
	MinimumComputePerSecond string `json:"minimum_compute_per_second_wei"`
	Stale                   bool   `json:"stale"`
}

type MarketOffer struct {
	MachineID           string    `json:"machine_id"`
	ProviderAddress     string    `json:"provider_address"`
	OfferID             string    `json:"offer_id"`
	Model               string    `json:"model"`
	Capabilities        []string  `json:"capabilities"`
	PriceVersion        uint64    `json:"price_version"`
	InputPerMillion     string    `json:"input_per_million_wei"`
	OutputPerMillion    string    `json:"output_per_million_wei"`
	ComputePerSecond    string    `json:"compute_per_second_wei"`
	Capacity            uint32    `json:"capacity"`
	LatencyMilliseconds uint64    `json:"latency_milliseconds"`
	SuccessBasisPoints  uint16    `json:"success_basis_points"`
	Reputation          uint64    `json:"reputation"`
	Available           bool      `json:"available"`
	Stale               bool      `json:"stale"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type MarketModelDetail struct {
	Model  string        `json:"model"`
	Offers []MarketOffer `json:"offers"`
}

type AccountActivity struct {
	RequestID       string    `json:"request_id"`
	SessionID       string    `json:"session_id"`
	AccountID       string    `json:"account_id"`
	State           string    `json:"state"`
	MachineID       string    `json:"machine_id"`
	OfferID         string    `json:"offer_id"`
	Model           string    `json:"model"`
	PriceVersion    uint64    `json:"price_version"`
	UpdatedAt       time.Time `json:"updated_at"`
	TransactionHash string    `json:"transaction_hash,omitempty"`
}

func (s *Store) MarketplaceModels(ctx context.Context, staleAfter time.Duration) ([]MarketModel, error) {
	cutoff := time.Now().Add(-staleAfter)
	rows, err := s.db.QueryContext(ctx, `SELECT prs.model,
		count(*) FILTER (WHERE prs.healthy AND prs.capacity>0 AND prs.updated_at >= $1),
		COALESCE(sum(prs.capacity) FILTER (WHERE prs.healthy AND prs.updated_at >= $1),0),
		min(o.input_per_million)::text,min(o.output_per_million)::text,min(o.compute_per_second)::text,
		bool_and(prs.updated_at < $1)
		FROM provider_routing_state prs JOIN offers o ON o.id=prs.offer_id AND o.version=prs.price_version
		WHERE prs.confirmed_bond GROUP BY prs.model ORDER BY prs.model`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	models := []MarketModel{}
	for rows.Next() {
		var model MarketModel
		if err := rows.Scan(&model.Model, &model.AvailableProviders, &model.TotalCapacity, &model.MinimumInputPerMillion, &model.MinimumOutputPerMillion, &model.MinimumComputePerSecond, &model.Stale); err != nil {
			return nil, err
		}
		models = append(models, model)
	}
	return models, rows.Err()
}

func (s *Store) MarketplaceModel(ctx context.Context, modelName string, staleAfter time.Duration) (MarketModelDetail, error) {
	cutoff := time.Now().Add(-staleAfter)
	rows, err := s.db.QueryContext(ctx, `SELECT prs.machine_id,a.wallet_address,prs.offer_id,prs.model,array_to_json(prs.capabilities),prs.price_version,
		o.input_per_million::text,o.output_per_million::text,o.compute_per_second::text,prs.capacity,
		prs.latency_milliseconds,prs.success_basis_points,prs.reputation,
		(prs.healthy AND prs.capacity>0 AND prs.updated_at >= $2),prs.updated_at < $2,prs.updated_at
		FROM provider_routing_state prs
		JOIN offers o ON o.id=prs.offer_id AND o.version=prs.price_version
		JOIN machines m ON m.id=prs.machine_id JOIN accounts a ON a.id=m.account_id
		WHERE prs.model=$1 AND prs.confirmed_bond ORDER BY o.input_per_million,o.output_per_million,prs.machine_id`, modelName, cutoff)
	if err != nil {
		return MarketModelDetail{}, err
	}
	defer rows.Close()
	detail := MarketModelDetail{Model: modelName, Offers: []MarketOffer{}}
	for rows.Next() {
		var offer MarketOffer
		var capabilities string
		if err := rows.Scan(&offer.MachineID, &offer.ProviderAddress, &offer.OfferID, &offer.Model, &capabilities, &offer.PriceVersion, &offer.InputPerMillion, &offer.OutputPerMillion, &offer.ComputePerSecond, &offer.Capacity, &offer.LatencyMilliseconds, &offer.SuccessBasisPoints, &offer.Reputation, &offer.Available, &offer.Stale, &offer.UpdatedAt); err != nil {
			return MarketModelDetail{}, err
		}
		if err := json.Unmarshal([]byte(capabilities), &offer.Capabilities); err != nil {
			return MarketModelDetail{}, err
		}
		detail.Offers = append(detail.Offers, offer)
	}
	if err := rows.Err(); err != nil {
		return MarketModelDetail{}, err
	}
	if len(detail.Offers) == 0 {
		return MarketModelDetail{}, sql.ErrNoRows
	}
	return detail, nil
}

func (s *Store) AccountActivity(ctx context.Context, accountID string, limit int) ([]AccountActivity, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT r.id,r.session_id,s.account_id,r.state,COALESCE(r.machine_id,''),COALESCE(r.offer_id,''),
		COALESCE(rp.model,''),COALESCE(r.price_version,0),r.updated_at,COALESCE(cs.transaction_hash,'')
		FROM requests r JOIN sessions s ON s.id=r.session_id
		LEFT JOIN receipt_proposals rp ON rp.request_id=r.id
		LEFT JOIN chain_settlements cs ON cs.request_id=r.id
		WHERE s.account_id=$1 ORDER BY r.updated_at DESC LIMIT $2`, accountID, limit)
	if err != nil {
		// chain_settlements is introduced by the chain migration; keep the API explicit if deployment migrations are incomplete.
		return nil, err
	}
	defer rows.Close()
	activity := []AccountActivity{}
	for rows.Next() {
		var item AccountActivity
		if err := rows.Scan(&item.RequestID, &item.SessionID, &item.AccountID, &item.State, &item.MachineID, &item.OfferID, &item.Model, &item.PriceVersion, &item.UpdatedAt, &item.TransactionHash); err != nil {
			return nil, err
		}
		activity = append(activity, item)
	}
	return activity, rows.Err()
}
