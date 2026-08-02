package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/kunalshah017/myference/server/internal/store"
)

type MarketplaceRepository interface {
	MarketplaceModels(context.Context, time.Duration) ([]store.MarketModel, error)
	MarketplaceModel(context.Context, string, time.Duration) (store.MarketModelDetail, error)
	AccountActivity(context.Context, string, int) ([]store.AccountActivity, error)
}

type AccountFromRequest func(*http.Request) (string, error)

type marketplaceHandler struct {
	repository MarketplaceRepository
	account    AccountFromRequest
	staleAfter time.Duration
}

func NewMarketplace(repository MarketplaceRepository, account AccountFromRequest, staleAfter time.Duration) http.Handler {
	if staleAfter <= 0 {
		staleAfter = 30 * time.Second
	}
	h := &marketplaceHandler{repository: repository, account: account, staleAfter: staleAfter}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/models", h.models)
	mux.HandleFunc("GET /api/models/{model}", h.model)
	mux.HandleFunc("GET /api/activity", h.activity)
	return mux
}

func (h *marketplaceHandler) models(w http.ResponseWriter, r *http.Request) {
	models, err := h.repository.MarketplaceModels(r.Context(), h.staleAfter)
	marketJSON(w, models, err)
}

func (h *marketplaceHandler) model(w http.ResponseWriter, r *http.Request) {
	model := strings.TrimSpace(r.PathValue("model"))
	if model == "" {
		http.Error(w, "model is required", http.StatusBadRequest)
		return
	}
	detail, err := h.repository.MarketplaceModel(r.Context(), model, h.staleAfter)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "model not found", http.StatusNotFound)
		return
	}
	marketJSON(w, detail, err)
}

func (h *marketplaceHandler) activity(w http.ResponseWriter, r *http.Request) {
	accountID, err := h.account(r)
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	activity, err := h.repository.AccountActivity(r.Context(), accountID, 100)
	marketJSON(w, activity, err)
}

func marketJSON(w http.ResponseWriter, value any, err error) {
	if err != nil {
		http.Error(w, "marketplace unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
