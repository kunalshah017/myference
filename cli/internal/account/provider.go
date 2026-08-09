package account

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

const (
	ActionPublishOffer           = "publish_offer"
	ActionDepositCollateral      = "deposit_collateral"
	ActionRequestCollateralExit  = "request_collateral_exit"
	ActionFinalizeCollateralExit = "finalize_collateral_exit"
	ActionPendingWallet          = "pending_wallet"
	ActionPendingChain           = "pending_chain"
	ActionConfirmed              = "confirmed"
)

type ProviderOffer struct {
	OfferID             string   `json:"offer_id"`
	Model               string   `json:"model"`
	Kind                string   `json:"kind"`
	Capabilities        []string `json:"capabilities"`
	MeteringMode        string   `json:"metering_mode"`
	InputPerMillionWei  string   `json:"input_per_million_wei"`
	OutputPerMillionWei string   `json:"output_per_million_wei"`
	ComputePerSecondWei string   `json:"compute_per_second_wei"`
}

type ProviderActionInput struct {
	Kind      string          `json:"kind"`
	AmountWei string          `json:"amount_wei,omitempty"`
	Offers    []ProviderOffer `json:"offers,omitempty"`
}

type ProviderAction struct {
	ID                string            `json:"id"`
	Kind              string            `json:"kind"`
	Status            string            `json:"status"`
	WalletAddress     string            `json:"wallet_address"`
	AmountWei         string            `json:"amount_wei,omitempty"`
	Offers            []ProviderOffer   `json:"offers,omitempty"`
	TransactionHashes []string          `json:"transaction_hashes,omitempty"`
	Versions          map[string]uint64 `json:"versions,omitempty"`
	ExpiresAt         time.Time         `json:"expires_at"`
}

type EditableOffer struct {
	OfferID             string   `json:"offer_id"`
	Model               string   `json:"model"`
	BackendKind         string   `json:"backend_kind"`
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

func (c *Client) ProviderAccount(ctx context.Context, token string) (ProviderAccount, error) {
	var result ProviderAccount
	err := c.authorizedRequest(ctx, http.MethodGet, "/api/provider/account", token, nil, &result)
	return result, err
}

func (c *Client) CreateProviderAction(ctx context.Context, token string, input ProviderActionInput) (ProviderAction, error) {
	var result ProviderAction
	err := c.authorizedRequest(ctx, http.MethodPost, "/api/provider/actions", token, input, &result)
	return result, err
}

func (c *Client) ProviderAction(ctx context.Context, token, id string) (ProviderAction, error) {
	var result ProviderAction
	err := c.authorizedRequest(ctx, http.MethodGet, "/api/provider/actions/"+url.PathEscape(id), token, nil, &result)
	return result, err
}

func (c *Client) MachineOfferVersions(ctx context.Context, token, machineID string) (map[string]uint64, error) {
	result := make(map[string]uint64)
	err := c.authorizedRequest(ctx, http.MethodGet, "/api/provider/machines/"+url.PathEscape(machineID)+"/offer-versions", token, nil, &result)
	return result, err
}
