package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	activationTTL       = 15 * time.Minute
	maxActivations      = 1024
	maxDraftOffers      = 64
	ActivationPending   = "pending"
	ActivationConfirmed = "confirmed"
)

var ErrActivationNotFound = errors.New("provider activation not found")

type ActivationOffer struct {
	OfferID      string   `json:"offer_id"`
	Model        string   `json:"model"`
	Kind         string   `json:"kind"`
	Capabilities []string `json:"capabilities,omitempty"`
	MeteringMode string   `json:"metering_mode,omitempty"`
}

type ProviderActivation struct {
	ID       string            `json:"id"`
	Status   string            `json:"status"`
	Offers   []ActivationOffer `json:"offers"`
	Versions map[string]int    `json:"versions,omitempty"`
	Expires  time.Time         `json:"expires_at"`

	machineID string
	accountID string
}

type ActivationStore struct {
	mu    sync.Mutex
	now   func() time.Time
	items map[string]ProviderActivation
}

func NewActivationStore(now func() time.Time) *ActivationStore {
	if now == nil {
		now = time.Now
	}
	return &ActivationStore{now: now, items: make(map[string]ProviderActivation)}
}

func (s *ActivationStore) Create(machineID, accountID string, offers []ActivationOffer, ttl time.Duration) (ProviderActivation, error) {
	if strings.TrimSpace(machineID) == "" || strings.TrimSpace(accountID) == "" || len(offers) == 0 || len(offers) > maxDraftOffers {
		return ProviderActivation{}, errors.New("invalid provider activation")
	}
	seen := make(map[string]struct{}, len(offers))
	for i := range offers {
		offers[i].OfferID = strings.TrimSpace(offers[i].OfferID)
		offers[i].Model = strings.TrimSpace(offers[i].Model)
		offers[i].Kind = strings.TrimSpace(offers[i].Kind)
		if offers[i].OfferID == "" || offers[i].Model == "" || offers[i].Kind == "" {
			return ProviderActivation{}, errors.New("activation offers require offer_id, model, and kind")
		}
		if _, exists := seen[offers[i].OfferID]; exists {
			return ProviderActivation{}, errors.New("activation offer IDs must be unique")
		}
		seen[offers[i].OfferID] = struct{}{}
	}
	if ttl <= 0 {
		ttl = activationTTL
	}
	id, err := activationID()
	if err != nil {
		return ProviderActivation{}, err
	}
	draft := ProviderActivation{
		ID: id, Status: ActivationPending, Offers: append([]ActivationOffer(nil), offers...),
		Expires: s.now().Add(ttl), machineID: machineID, accountID: accountID,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgeExpiredLocked()
	if len(s.items) >= maxActivations {
		return ProviderActivation{}, errors.New("too many provider activations")
	}
	s.items[id] = draft
	return draft, nil
}

func (s *ActivationStore) Get(id string) (ProviderActivation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	draft, ok := s.items[id]
	if !ok || !s.now().Before(draft.Expires) {
		delete(s.items, id)
		return ProviderActivation{}, ErrActivationNotFound
	}
	return draft, nil
}

func (s *ActivationStore) Complete(id, accountID string, versions map[string]int) (ProviderActivation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	draft, ok := s.items[id]
	if !ok || !s.now().Before(draft.Expires) {
		delete(s.items, id)
		return ProviderActivation{}, ErrActivationNotFound
	}
	if draft.accountID != accountID {
		return ProviderActivation{}, ErrActivationNotFound
	}
	if len(versions) != len(draft.Offers) {
		return ProviderActivation{}, errors.New("a version is required for every offer")
	}
	for _, offer := range draft.Offers {
		if versions[offer.OfferID] <= 0 {
			return ProviderActivation{}, errors.New("a positive version is required for every offer")
		}
	}
	draft.Status = ActivationConfirmed
	draft.Versions = cloneVersions(versions)
	s.items[id] = draft
	return draft, nil
}

func (s *ActivationStore) purgeExpiredLocked() {
	now := s.now()
	for id, draft := range s.items {
		if !now.Before(draft.Expires) {
			delete(s.items, id)
		}
	}
}

func activationID() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func cloneVersions(versions map[string]int) map[string]int {
	result := make(map[string]int, len(versions))
	for id, version := range versions {
		result[id] = version
	}
	return result
}

type MachineActivationAuth func(*http.Request) (machineID, accountID string, err error)

type providerActivationHandler struct {
	store       *ActivationStore
	machineAuth MachineActivationAuth
	accountAuth AccountFromRequest
}

func NewProviderActivation(store *ActivationStore, machineAuth MachineActivationAuth, accountAuth AccountFromRequest) http.Handler {
	h := &providerActivationHandler{store: store, machineAuth: machineAuth, accountAuth: accountAuth}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/provider/activations", h.create)
	mux.HandleFunc("GET /api/provider/activations/{id}", h.get)
	mux.HandleFunc("POST /api/provider/activations/{id}/complete", h.complete)
	return mux
}

func (h *providerActivationHandler) create(w http.ResponseWriter, r *http.Request) {
	machineID, accountID, err := h.machineAuth(r)
	if err != nil {
		http.Error(w, "machine authentication required", http.StatusUnauthorized)
		return
	}
	var input struct {
		Offers []ActivationOffer `json:"offers"`
	}
	if err := decodeActivationJSON(r, &input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	draft, err := h.store.Create(machineID, accountID, input.Offers, activationTTL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	activationJSON(w, http.StatusCreated, draft)
}

func (h *providerActivationHandler) get(w http.ResponseWriter, r *http.Request) {
	draft, err := h.store.Get(r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var accountID string
	if strings.TrimSpace(r.Header.Get("Authorization")) != "" {
		_, accountID, err = h.machineAuth(r)
	} else {
		accountID, err = h.accountAuth(r)
	}
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if accountID != draft.accountID {
		http.NotFound(w, r)
		return
	}
	activationJSON(w, http.StatusOK, draft)
}

func (h *providerActivationHandler) complete(w http.ResponseWriter, r *http.Request) {
	accountID, err := h.accountAuth(r)
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	var input struct {
		Versions map[string]int `json:"versions"`
	}
	if err := decodeActivationJSON(r, &input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	draft, err := h.store.Complete(r.PathValue("id"), accountID, input.Versions)
	if errors.Is(err, ErrActivationNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	activationJSON(w, http.StatusOK, draft)
}

func decodeActivationJSON(r *http.Request, value any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	return decoder.Decode(value)
}

func activationJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
