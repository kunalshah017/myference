package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"slices"
	"strings"
	"sync"
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
	ActionSourceMachine          = "machine"
	ActionSourceBrowser          = "browser"
	providerActionTTL            = 15 * time.Minute
	maxProviderActions           = 1024
	maxProviderActionOffers      = 64
)

var ErrProviderActionNotFound = errors.New("provider action not found")

type ProviderActionOffer struct {
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
	Kind      string                `json:"kind"`
	AmountWei string                `json:"amount_wei,omitempty"`
	Offers    []ProviderActionOffer `json:"offers,omitempty"`
}

type ProviderActionBaseline struct {
	BondWei         string
	ClaimableWei    string
	ExitAvailableAt uint64
	Versions        map[string]uint64
}

type ProviderAction struct {
	ID                string                `json:"id"`
	Kind              string                `json:"kind"`
	Status            string                `json:"status"`
	WalletAddress     string                `json:"wallet_address"`
	AmountWei         string                `json:"amount_wei,omitempty"`
	Offers            []ProviderActionOffer `json:"offers,omitempty"`
	TransactionHashes []string              `json:"transaction_hashes,omitempty"`
	Versions          map[string]uint64     `json:"versions,omitempty"`
	ExpiresAt         time.Time             `json:"expires_at"`

	MachineIDValue string                 `json:"-"`
	AccountIDValue string                 `json:"-"`
	BaselineState  ProviderActionBaseline `json:"-"`
}

func (action ProviderAction) AccountID() string                { return action.AccountIDValue }
func (action ProviderAction) Baseline() ProviderActionBaseline { return action.BaselineState }

type ProviderActionStore struct {
	mu    sync.Mutex
	now   func() time.Time
	items map[string]ProviderAction
}

func NewProviderActionStore(now func() time.Time) *ProviderActionStore {
	if now == nil {
		now = time.Now
	}
	return &ProviderActionStore{now: now, items: make(map[string]ProviderAction)}
}

func (s *ProviderActionStore) Create(machineID, accountID, wallet string, input ProviderActionInput, baseline ProviderActionBaseline) (ProviderAction, error) {
	if strings.TrimSpace(accountID) == "" || strings.TrimSpace(wallet) == "" {
		return ProviderAction{}, errors.New("provider account is required")
	}
	if err := validateProviderActionInput(&input); err != nil {
		return ProviderAction{}, err
	}
	id, err := providerActionID()
	if err != nil {
		return ProviderAction{}, err
	}
	action := ProviderAction{ID: id, Kind: input.Kind, Status: ActionPendingWallet, WalletAddress: wallet, AmountWei: input.AmountWei, Offers: append([]ProviderActionOffer(nil), input.Offers...), ExpiresAt: s.now().Add(providerActionTTL), MachineIDValue: machineID, AccountIDValue: accountID, BaselineState: baseline}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeLocked()
	if len(s.items) >= maxProviderActions {
		return ProviderAction{}, errors.New("too many provider actions")
	}
	s.items[id] = action
	return action, nil
}

func (s *ProviderActionStore) GetForAccount(id, accountID string) (ProviderAction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	action, ok := s.items[id]
	if !ok || !s.now().Before(action.ExpiresAt) {
		delete(s.items, id)
		return ProviderAction{}, ErrProviderActionNotFound
	}
	if action.AccountIDValue != accountID {
		return ProviderAction{}, ErrProviderActionNotFound
	}
	return action, nil
}

func (s *ProviderActionStore) Submitted(id, accountID string, hashes []string) (ProviderAction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	action, ok := s.items[id]
	if !ok || !s.now().Before(action.ExpiresAt) {
		delete(s.items, id)
		return ProviderAction{}, ErrProviderActionNotFound
	}
	if action.AccountIDValue != accountID {
		return ProviderAction{}, ErrProviderActionNotFound
	}
	if action.Status == ActionConfirmed {
		return action, nil
	}
	if len(hashes) == 0 || len(hashes) > max(1, len(action.Offers)) {
		return ProviderAction{}, errors.New("transaction hashes do not match this action")
	}
	for _, hash := range hashes {
		if !validTransactionHash(hash) {
			return ProviderAction{}, errors.New("invalid transaction hash")
		}
	}
	action.TransactionHashes = append([]string(nil), hashes...)
	action.Status = ActionPendingChain
	s.items[id] = action
	return action, nil
}

func (s *ProviderActionStore) Confirm(id, accountID string, versions map[string]uint64) (ProviderAction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	action, ok := s.items[id]
	if !ok || action.AccountIDValue != accountID {
		return ProviderAction{}, ErrProviderActionNotFound
	}
	action.Status = ActionConfirmed
	action.Versions = cloneActionVersions(versions)
	s.items[id] = action
	return action, nil
}

func (s *ProviderActionStore) purgeLocked() {
	now := s.now()
	for id, action := range s.items {
		if !now.Before(action.ExpiresAt) {
			delete(s.items, id)
		}
	}
}

type ProviderActionDependencies struct {
	MachineAuth func(*http.Request) (machineID, accountID string, err error)
	AccountAuth AccountFromRequest
	Prepare     func(ctx context.Context, source, machineID, accountID string, input ProviderActionInput) (wallet string, baseline ProviderActionBaseline, err error)
	Verify      func(ctx context.Context, action ProviderAction) (versions map[string]uint64, confirmed bool, err error)
}

type providerActionHandler struct {
	store *ProviderActionStore
	deps  ProviderActionDependencies
}

func NewProviderActions(store *ProviderActionStore, deps ProviderActionDependencies) http.Handler {
	h := &providerActionHandler{store: store, deps: deps}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/provider/actions", h.create)
	mux.HandleFunc("GET /api/provider/actions/{id}", h.get)
	mux.HandleFunc("POST /api/provider/actions/{id}/submitted", h.submitted)
	return mux
}

func (h *providerActionHandler) create(w http.ResponseWriter, r *http.Request) {
	source, machineID, accountID, err := h.identity(r)
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	var input ProviderActionInput
	if err := decodeProviderActionJSON(r, &input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if h.deps.Prepare == nil {
		http.Error(w, "provider actions unavailable", http.StatusServiceUnavailable)
		return
	}
	wallet, baseline, err := h.deps.Prepare(r.Context(), source, machineID, accountID, input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	action, err := h.store.Create(machineID, accountID, wallet, input, baseline)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	providerActionJSON(w, http.StatusCreated, action)
}

func (h *providerActionHandler) get(w http.ResponseWriter, r *http.Request) {
	source, machineID, accountID, err := h.identity(r)
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	action, err := h.store.GetForAccount(r.PathValue("id"), accountID)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if source == ActionSourceMachine && action.MachineIDValue != machineID {
		http.NotFound(w, r)
		return
	}
	if action.Status == ActionPendingChain && h.deps.Verify != nil {
		versions, confirmed, verifyErr := h.deps.Verify(r.Context(), action)
		if verifyErr != nil {
			http.Error(w, "chain confirmation unavailable", http.StatusServiceUnavailable)
			return
		}
		if confirmed {
			action, err = h.store.Confirm(action.ID, accountID, versions)
			if err != nil {
				http.NotFound(w, r)
				return
			}
		}
	}
	providerActionJSON(w, http.StatusOK, action)
}

func (h *providerActionHandler) submitted(w http.ResponseWriter, r *http.Request) {
	source, _, accountID, err := h.identity(r)
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if source != ActionSourceBrowser {
		http.NotFound(w, r)
		return
	}
	var input struct {
		TransactionHashes []string `json:"transaction_hashes"`
	}
	if err := decodeProviderActionJSON(r, &input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	action, err := h.store.Submitted(r.PathValue("id"), accountID, input.TransactionHashes)
	if errors.Is(err, ErrProviderActionNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	providerActionJSON(w, http.StatusOK, action)
}

func (h *providerActionHandler) identity(r *http.Request) (source, machineID, accountID string, err error) {
	if strings.TrimSpace(r.Header.Get("Authorization")) != "" {
		if h.deps.MachineAuth == nil {
			return "", "", "", errors.New("machine authentication unavailable")
		}
		machineID, accountID, err = h.deps.MachineAuth(r)
		return ActionSourceMachine, machineID, accountID, err
	}
	if h.deps.AccountAuth == nil {
		return "", "", "", errors.New("account authentication unavailable")
	}
	accountID, err = h.deps.AccountAuth(r)
	return ActionSourceBrowser, "", accountID, err
}

func validateProviderActionInput(input *ProviderActionInput) error {
	input.Kind = strings.TrimSpace(input.Kind)
	switch input.Kind {
	case ActionDepositCollateral:
		if len(input.Offers) != 0 || !validDecimal(input.AmountWei, true) {
			return errors.New("deposit requires a positive amount_wei")
		}
	case ActionRequestCollateralExit, ActionFinalizeCollateralExit:
		if len(input.Offers) != 0 || strings.TrimSpace(input.AmountWei) != "" {
			return errors.New("collateral exit does not accept offers or an amount")
		}
	case ActionPublishOffer:
		if strings.TrimSpace(input.AmountWei) != "" || len(input.Offers) == 0 || len(input.Offers) > maxProviderActionOffers {
			return errors.New("offer publication requires one to 64 offers")
		}
		seen := make(map[string]struct{}, len(input.Offers))
		for i := range input.Offers {
			offer := &input.Offers[i]
			offer.OfferID, offer.Model, offer.Kind = strings.TrimSpace(offer.OfferID), strings.TrimSpace(offer.Model), strings.TrimSpace(offer.Kind)
			offer.MeteringMode = strings.TrimSpace(offer.MeteringMode)
			slices.Sort(offer.Capabilities)
			offer.Capabilities = slices.Compact(offer.Capabilities)
			if offer.OfferID == "" || offer.Model == "" || offer.Kind == "" || len(offer.Capabilities) == 0 || (offer.MeteringMode != "tokens_and_compute" && offer.MeteringMode != "compute_only") {
				return errors.New("offer identity and metering are required")
			}
			if _, exists := seen[offer.OfferID]; exists {
				return errors.New("offer IDs must be unique")
			}
			seen[offer.OfferID] = struct{}{}
			if !validDecimal(offer.InputPerMillionWei, false) || !validDecimal(offer.OutputPerMillionWei, false) || !validDecimal(offer.ComputePerSecondWei, false) {
				return errors.New("offer rates must be non-negative decimal integers")
			}
			if offer.MeteringMode == "compute_only" && (offer.InputPerMillionWei != "0" || offer.OutputPerMillionWei != "0") {
				return errors.New("compute-only offers require zero token rates")
			}
		}
	default:
		return errors.New("unsupported provider action")
	}
	return nil
}

func validDecimal(value string, positive bool) bool {
	parsed, ok := new(big.Int).SetString(value, 10)
	return ok && parsed.BitLen() <= 256 && ((!positive && parsed.Sign() >= 0) || (positive && parsed.Sign() > 0)) && parsed.String() == value
}

func validTransactionHash(value string) bool {
	if len(value) != 66 || !strings.HasPrefix(value, "0x") {
		return false
	}
	decoded, err := hex.DecodeString(value[2:])
	return err == nil && len(decoded) == 32
}

func providerActionID() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func cloneActionVersions(source map[string]uint64) map[string]uint64 {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]uint64, len(source))
	for id, version := range source {
		result[id] = version
	}
	return result
}

func decodeProviderActionJSON(r *http.Request, value any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	return decoder.Decode(value)
}

func providerActionJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
