package store

import (
	"context"
	"database/sql"
	"time"
)

type AnalyticsTotals struct {
	SettledRequests     uint64 `json:"settled_requests"`
	InputTokens         uint64 `json:"input_tokens"`
	OutputTokens        uint64 `json:"output_tokens"`
	ComputeMilliseconds uint64 `json:"compute_milliseconds"`
	ProviderChargesWei  string `json:"provider_charges_wei"`
	ProtocolFeesWei     string `json:"protocol_fees_wei"`
	TotalSpentWei       string `json:"total_spent_wei"`
	GrossRevenueWei     string `json:"gross_revenue_wei"`
	TotalSlashedWei     string `json:"total_slashed_wei"`
}

type AnalyticsDay struct {
	Date               string `json:"date"`
	CustomerRequests   uint64 `json:"customer_requests"`
	CustomerSpentWei   string `json:"customer_spent_wei"`
	ProviderRequests   uint64 `json:"provider_requests"`
	ProviderRevenueWei string `json:"provider_revenue_wei"`
}

type UsageRecord struct {
	RequestID           string    `json:"request_id"`
	Model               string    `json:"model"`
	ProviderAmountWei   string    `json:"provider_amount_wei"`
	FeeAmountWei        string    `json:"fee_amount_wei"`
	TotalChargeWei      string    `json:"total_charge_wei"`
	TransactionHash     string    `json:"transaction_hash"`
	InputTokens         uint64    `json:"input_tokens"`
	OutputTokens        uint64    `json:"output_tokens"`
	ComputeMilliseconds uint64    `json:"compute_milliseconds"`
	CompletedAt         time.Time `json:"completed_at"`
}

type ProviderSettlement struct {
	RequestID           string    `json:"request_id"`
	Model               string    `json:"model"`
	RevenueWei          string    `json:"revenue_wei"`
	TransactionHash     string    `json:"transaction_hash"`
	InputTokens         uint64    `json:"input_tokens"`
	OutputTokens        uint64    `json:"output_tokens"`
	ComputeMilliseconds uint64    `json:"compute_milliseconds"`
	CompletedAt         time.Time `json:"completed_at"`
}

type SlashRecord struct {
	RequestID       string    `json:"request_id"`
	AmountWei       string    `json:"amount_wei"`
	TransactionHash string    `json:"transaction_hash"`
	BlockNumber     uint64    `json:"block_number"`
	IndexedAt       time.Time `json:"indexed_at"`
}

type AccountAnalytics struct {
	Customer    AnalyticsTotals      `json:"customer"`
	Provider    AnalyticsTotals      `json:"provider"`
	Daily       []AnalyticsDay       `json:"daily"`
	Usage       []UsageRecord        `json:"usage"`
	Settlements []ProviderSettlement `json:"settlements"`
	Slashes     []SlashRecord        `json:"slashes"`
}

func emptyAnalytics() AccountAnalytics {
	zero := AnalyticsTotals{ProviderChargesWei: "0", ProtocolFeesWei: "0", TotalSpentWei: "0", GrossRevenueWei: "0", TotalSlashedWei: "0"}
	return AccountAnalytics{Customer: zero, Provider: zero, Daily: []AnalyticsDay{}, Usage: []UsageRecord{}, Settlements: []ProviderSettlement{}, Slashes: []SlashRecord{}}
}

func (s *Store) AccountAnalytics(ctx context.Context, accountID string, chainID uint64, contract string) (AccountAnalytics, error) {
	view := emptyAnalytics()
	var wallet string
	if err := s.db.QueryRowContext(ctx, `SELECT wallet_address FROM accounts WHERE id=$1`, accountID).Scan(&wallet); err != nil {
		return AccountAnalytics{}, err
	}
	customerBase := ` FROM chain_settlements cs JOIN receipt_proposals rp ON rp.request_id=cs.request_id JOIN sessions se ON se.id=rp.session_id WHERE cs.chain_id=$1 AND lower(cs.contract_address)=lower($2) AND se.account_id=$3`
	if err := scanTotals(s.db.QueryRowContext(ctx, `SELECT count(*),COALESCE(sum(rp.input_tokens),0),COALESCE(sum(rp.output_tokens),0),COALESCE(sum(rp.compute_milliseconds),0),COALESCE(sum(cs.provider_amount),0)::text,COALESCE(sum(cs.fee_amount),0)::text,COALESCE(sum(cs.provider_amount+cs.fee_amount),0)::text`+customerBase, chainID, contract, accountID), &view.Customer); err != nil {
		return AccountAnalytics{}, err
	}
	providerBase := ` FROM chain_settlements cs JOIN receipt_proposals rp ON rp.request_id=cs.request_id JOIN machines m ON m.id=rp.machine_id WHERE cs.chain_id=$1 AND lower(cs.contract_address)=lower($2) AND m.account_id=$3`
	if err := scanProviderTotals(s.db.QueryRowContext(ctx, `SELECT count(*),COALESCE(sum(rp.input_tokens),0),COALESCE(sum(rp.output_tokens),0),COALESCE(sum(rp.compute_milliseconds),0),COALESCE(sum(cs.provider_amount),0)::text`+providerBase, chainID, contract, accountID), &view.Provider); err != nil {
		return AccountAnalytics{}, err
	}
	_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(sum(amount),0)::text FROM chain_slashes WHERE chain_id=$1 AND lower(contract_address)=lower($2) AND lower(provider)=lower($3)`, chainID, contract, wallet).Scan(&view.Provider.TotalSlashedWei)
	if err := s.loadDailyAnalytics(ctx, &view, accountID, chainID, contract); err != nil {
		return AccountAnalytics{}, err
	}
	if err := s.loadRecentAnalytics(ctx, &view, accountID, wallet, chainID, contract); err != nil {
		return AccountAnalytics{}, err
	}
	return view, nil
}

func scanTotals(row *sql.Row, totals *AnalyticsTotals) error {
	return row.Scan(&totals.SettledRequests, &totals.InputTokens, &totals.OutputTokens, &totals.ComputeMilliseconds, &totals.ProviderChargesWei, &totals.ProtocolFeesWei, &totals.TotalSpentWei)
}
func scanProviderTotals(row *sql.Row, totals *AnalyticsTotals) error {
	return row.Scan(&totals.SettledRequests, &totals.InputTokens, &totals.OutputTokens, &totals.ComputeMilliseconds, &totals.GrossRevenueWei)
}

func (s *Store) loadDailyAnalytics(ctx context.Context, view *AccountAnalytics, accountID string, chainID uint64, contract string) error {
	points := map[string]*AnalyticsDay{}
	for offset := 29; offset >= 0; offset-- {
		date := time.Now().UTC().AddDate(0, 0, -offset).Format("2006-01-02")
		points[date] = &AnalyticsDay{Date: date, CustomerSpentWei: "0", ProviderRevenueWei: "0"}
	}
	rows, err := s.db.QueryContext(ctx, `SELECT (rp.completed_at AT TIME ZONE 'UTC')::date::text,count(*),sum(cs.provider_amount+cs.fee_amount)::text FROM chain_settlements cs JOIN receipt_proposals rp ON rp.request_id=cs.request_id JOIN sessions se ON se.id=rp.session_id WHERE cs.chain_id=$1 AND lower(cs.contract_address)=lower($2) AND se.account_id=$3 AND rp.completed_at>=now()-interval '30 days' GROUP BY 1`, chainID, contract, accountID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var date, spent string
		var count uint64
		if err := rows.Scan(&date, &count, &spent); err != nil {
			rows.Close()
			return err
		}
		if point := points[date]; point != nil {
			point.CustomerRequests = count
			point.CustomerSpentWei = spent
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	rows, err = s.db.QueryContext(ctx, `SELECT (rp.completed_at AT TIME ZONE 'UTC')::date::text,count(*),sum(cs.provider_amount)::text FROM chain_settlements cs JOIN receipt_proposals rp ON rp.request_id=cs.request_id JOIN machines m ON m.id=rp.machine_id WHERE cs.chain_id=$1 AND lower(cs.contract_address)=lower($2) AND m.account_id=$3 AND rp.completed_at>=now()-interval '30 days' GROUP BY 1`, chainID, contract, accountID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var date, revenue string
		var count uint64
		if err := rows.Scan(&date, &count, &revenue); err != nil {
			rows.Close()
			return err
		}
		if point := points[date]; point != nil {
			point.ProviderRequests = count
			point.ProviderRevenueWei = revenue
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for offset := 29; offset >= 0; offset-- {
		date := time.Now().UTC().AddDate(0, 0, -offset).Format("2006-01-02")
		view.Daily = append(view.Daily, *points[date])
	}
	return nil
}

func (s *Store) loadRecentAnalytics(ctx context.Context, view *AccountAnalytics, accountID, wallet string, chainID uint64, contract string) error {
	rows, err := s.db.QueryContext(ctx, `SELECT rp.request_id,rp.model,rp.input_tokens,rp.output_tokens,rp.compute_milliseconds,cs.provider_amount::text,cs.fee_amount::text,(cs.provider_amount+cs.fee_amount)::text,cs.transaction_hash,rp.completed_at FROM chain_settlements cs JOIN receipt_proposals rp ON rp.request_id=cs.request_id JOIN sessions se ON se.id=rp.session_id WHERE cs.chain_id=$1 AND lower(cs.contract_address)=lower($2) AND se.account_id=$3 ORDER BY rp.completed_at DESC LIMIT 100`, chainID, contract, accountID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var item UsageRecord
		if err := rows.Scan(&item.RequestID, &item.Model, &item.InputTokens, &item.OutputTokens, &item.ComputeMilliseconds, &item.ProviderAmountWei, &item.FeeAmountWei, &item.TotalChargeWei, &item.TransactionHash, &item.CompletedAt); err != nil {
			rows.Close()
			return err
		}
		view.Usage = append(view.Usage, item)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	rows, err = s.db.QueryContext(ctx, `SELECT rp.request_id,rp.model,rp.input_tokens,rp.output_tokens,rp.compute_milliseconds,cs.provider_amount::text,cs.transaction_hash,rp.completed_at FROM chain_settlements cs JOIN receipt_proposals rp ON rp.request_id=cs.request_id JOIN machines m ON m.id=rp.machine_id WHERE cs.chain_id=$1 AND lower(cs.contract_address)=lower($2) AND m.account_id=$3 ORDER BY rp.completed_at DESC LIMIT 100`, chainID, contract, accountID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var item ProviderSettlement
		if err := rows.Scan(&item.RequestID, &item.Model, &item.InputTokens, &item.OutputTokens, &item.ComputeMilliseconds, &item.RevenueWei, &item.TransactionHash, &item.CompletedAt); err != nil {
			rows.Close()
			return err
		}
		view.Settlements = append(view.Settlements, item)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	rows, err = s.db.QueryContext(ctx, `SELECT request_id,amount::text,block_number,transaction_hash,indexed_at FROM chain_slashes WHERE chain_id=$1 AND lower(contract_address)=lower($2) AND lower(provider)=lower($3) ORDER BY indexed_at DESC LIMIT 100`, chainID, contract, wallet)
	if err != nil {
		return err
	}
	for rows.Next() {
		var item SlashRecord
		if err := rows.Scan(&item.RequestID, &item.AmountWei, &item.BlockNumber, &item.TransactionHash, &item.IndexedAt); err != nil {
			rows.Close()
			return err
		}
		view.Slashes = append(view.Slashes, item)
	}
	return rows.Close()
}
