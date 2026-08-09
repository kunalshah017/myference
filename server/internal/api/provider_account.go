package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kunalshah017/myference/server/internal/store"
)

type ProviderAccountRepository interface {
	ProviderAccount(context.Context, string, store.ProviderAccountConfig) (store.ProviderAccount, error)
	MachineOfferVersions(context.Context, string, string, uint64, string) (map[string]uint64, error)
}

type MachineRequestAuth func(*http.Request) (machineID, accountID string, err error)

type providerAccountHandler struct {
	repository  ProviderAccountRepository
	machineAuth MachineRequestAuth
	accountAuth AccountFromRequest
	config      store.ProviderAccountConfig
}

func NewProviderAccount(repository ProviderAccountRepository, machineAuth MachineRequestAuth, accountAuth AccountFromRequest, config store.ProviderAccountConfig) http.Handler {
	h := &providerAccountHandler{repository: repository, machineAuth: machineAuth, accountAuth: accountAuth, config: config}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/provider/account", h.account)
	mux.HandleFunc("GET /api/provider/machines/{machine}/offer-versions", h.versions)
	return mux
}

func (h *providerAccountHandler) account(w http.ResponseWriter, r *http.Request) {
	var accountID string
	var err error
	if strings.TrimSpace(r.Header.Get("Authorization")) != "" {
		_, accountID, err = h.machineAuth(r)
	} else {
		accountID, err = h.accountAuth(r)
	}
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	account, err := h.repository.ProviderAccount(r.Context(), accountID, h.config)
	if err != nil {
		http.Error(w, "provider account unavailable", http.StatusServiceUnavailable)
		return
	}
	writeProviderAccountJSON(w, account)
}

func (h *providerAccountHandler) versions(w http.ResponseWriter, r *http.Request) {
	machineID, accountID, err := h.machineAuth(r)
	if err != nil {
		http.Error(w, "machine authentication required", http.StatusUnauthorized)
		return
	}
	if machineID != r.PathValue("machine") {
		http.NotFound(w, r)
		return
	}
	versions, err := h.repository.MachineOfferVersions(r.Context(), machineID, accountID, h.config.ChainID, h.config.ContractAddress)
	if err != nil {
		http.Error(w, "offer versions unavailable", http.StatusServiceUnavailable)
		return
	}
	writeProviderAccountJSON(w, versions)
}

func writeProviderAccountJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
