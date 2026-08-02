package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/kunalshah017/myference/server/internal/store"
)

type AnalyticsConfig struct {
	ChainID         uint64
	ContractAddress string
}

type AnalyticsRepository interface {
	AccountAnalytics(context.Context, string, uint64, string) (store.AccountAnalytics, error)
}

func NewAnalytics(repository AnalyticsRepository, account AccountFromRequest, config AnalyticsConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/account/analytics" {
			http.NotFound(w, r)
			return
		}
		accountID, err := account(r)
		if err != nil {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		view, err := repository.AccountAnalytics(r.Context(), accountID, config.ChainID, config.ContractAddress)
		if err != nil {
			http.Error(w, "account analytics unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(view)
	})
}
