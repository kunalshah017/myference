package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
)

type ProviderAccountConfig struct {
	ChainID         uint64
	ContractAddress string
	ExplorerURL     string
	Confirmations   uint64
	MinimumBondWei  string
}

type EditableOffer struct {
	OfferID             string   `json:"offer_id"`
	Model               string   `json:"model"`
	Capabilities        []string `json:"capabilities"`
	MeteringMode        string   `json:"metering_mode"`
	Version             uint64   `json:"version"`
	InputPerMillionWei  string   `json:"input_per_million_wei"`
	OutputPerMillionWei string   `json:"output_per_million_wei"`
	ComputePerSecondWei string   `json:"compute_per_second_wei"`
}

type ProviderAccount struct {
	ChainID             uint64          `json:"chain_id"`
	ContractAddress     string          `json:"contract_address"`
	ExplorerURL         string          `json:"explorer_url"`
	Confirmations       uint64          `json:"confirmations"`
	WalletAddress       string          `json:"wallet_address"`
	MinimumBondWei      string          `json:"minimum_bond_wei"`
	ProviderBondWei     string          `json:"provider_bond_wei"`
	ClaimableWei        string          `json:"claimable_wei"`
	ProviderEarningsWei string          `json:"provider_earnings_wei"`
	BondExitAvailableAt uint64          `json:"bond_exit_available_at"`
	Offers              []EditableOffer `json:"offers"`
}

func (s *Store) ProviderAccount(ctx context.Context, accountID string, config ProviderAccountConfig) (ProviderAccount, error) {
	view := ProviderAccount{
		ChainID: config.ChainID, ContractAddress: strings.ToLower(config.ContractAddress), ExplorerURL: config.ExplorerURL,
		Confirmations: config.Confirmations, MinimumBondWei: config.MinimumBondWei,
		ProviderBondWei: "0", ClaimableWei: "0", ProviderEarningsWei: "0", Offers: []EditableOffer{},
	}
	if err := s.db.QueryRowContext(ctx, `SELECT wallet_address FROM accounts WHERE id=$1`, accountID).Scan(&view.WalletAddress); err != nil {
		return ProviderAccount{}, err
	}
	err := s.db.QueryRowContext(ctx, `SELECT provider_bond::text,claimable::text,bond_exit_available_at FROM chain_accounts WHERE chain_id=$1 AND lower(contract_address)=lower($2) AND lower(address)=lower($3)`, config.ChainID, config.ContractAddress, view.WalletAddress).Scan(&view.ProviderBondWei, &view.ClaimableWei, &view.BondExitAvailableAt)
	if err != nil && err != sql.ErrNoRows {
		return ProviderAccount{}, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(sum(provider_amount),0)::text FROM chain_settlements WHERE chain_id=$1 AND lower(contract_address)=lower($2) AND lower(provider)=lower($3)`, config.ChainID, config.ContractAddress, view.WalletAddress).Scan(&view.ProviderEarningsWei); err != nil {
		return ProviderAccount{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT DISTINCT ON (prs.offer_id) prs.offer_id,prs.model,array_to_json(prs.capabilities),prs.metering_mode,co.version,co.input_per_million::text,co.output_per_million::text,co.compute_per_second::text
		FROM provider_routing_state prs
		JOIN machines m ON m.id=prs.machine_id AND m.account_id=$1
		JOIN chain_offers co ON co.chain_id=$2 AND lower(co.contract_address)=lower($3) AND lower(co.provider)=lower($4) AND co.offer_id=prs.offer_hash AND co.model_hash=prs.model_hash AND co.capability_hash=prs.capability_hash
		ORDER BY prs.offer_id,co.version DESC`, accountID, config.ChainID, config.ContractAddress, view.WalletAddress)
	if err != nil {
		return ProviderAccount{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var offer EditableOffer
		var capabilities string
		if err := rows.Scan(&offer.OfferID, &offer.Model, &capabilities, &offer.MeteringMode, &offer.Version, &offer.InputPerMillionWei, &offer.OutputPerMillionWei, &offer.ComputePerSecondWei); err != nil {
			return ProviderAccount{}, err
		}
		if err := json.Unmarshal([]byte(capabilities), &offer.Capabilities); err != nil {
			return ProviderAccount{}, err
		}
		view.Offers = append(view.Offers, offer)
	}
	return view, rows.Err()
}

func (s *Store) MachineOfferVersions(ctx context.Context, machineID, accountID string, chainID uint64, contractAddress string) (map[string]uint64, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT prs.offer_id,max(co.version)
		FROM provider_routing_state prs
		JOIN machines m ON m.id=prs.machine_id AND m.id=$1 AND m.account_id=$2
		JOIN accounts a ON a.id=m.account_id
		JOIN chain_offers co ON co.chain_id=$3 AND lower(co.contract_address)=lower($4) AND lower(co.provider)=lower(a.wallet_address) AND co.offer_id=prs.offer_hash AND co.model_hash=prs.model_hash AND co.capability_hash=prs.capability_hash
		GROUP BY prs.offer_id`, machineID, accountID, chainID, contractAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]uint64)
	for rows.Next() {
		var offerID string
		var version uint64
		if err := rows.Scan(&offerID, &version); err != nil {
			return nil, err
		}
		result[offerID] = version
	}
	return result, rows.Err()
}
