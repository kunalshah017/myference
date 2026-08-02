package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/kunalshah017/myference/server/internal/store"
)

type OperationsConfig struct {
	ChainID                      uint64
	ContractAddress, ExplorerURL string
	Confirmations                uint64
}

type OperationsRepository interface {
	AccountOperations(context.Context, string, uint64, string, string, uint64) (store.AccountOperations, error)
}

func NewOperations(repository OperationsRepository, account AccountFromRequest, config OperationsConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/account/operations" {
			http.NotFound(w, r)
			return
		}
		accountID, err := account(r)
		if err != nil {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		view, err := repository.AccountOperations(r.Context(), accountID, config.ChainID, config.ContractAddress, config.ExplorerURL, config.Confirmations)
		if err != nil {
			http.Error(w, "account operations unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(view)
	})
}
