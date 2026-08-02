package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kunalshah017/myference/server/internal/store"
)

type analyticsRepositoryStub struct{ view store.AccountAnalytics }

func (s analyticsRepositoryStub) AccountAnalytics(context.Context, string, uint64, string) (store.AccountAnalytics, error) {
	return s.view, nil
}

func TestAnalyticsRequiresBrowserAuthenticationAndReturnsWeiStrings(t *testing.T) {
	config := AnalyticsConfig{ChainID: 10143, ContractAddress: "0xmarket"}
	unauthorized := NewAnalytics(analyticsRepositoryStub{}, func(*http.Request) (string, error) { return "", errors.New("no session") }, config)
	response := httptest.NewRecorder()
	unauthorized.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/account/analytics", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", response.Code)
	}
	view := store.AccountAnalytics{Customer: store.AnalyticsTotals{TotalSpentWei: "20"}, Provider: store.AnalyticsTotals{GrossRevenueWei: "19"}, Daily: []store.AnalyticsDay{}, Usage: []store.UsageRecord{}, Settlements: []store.ProviderSettlement{}, Slashes: []store.SlashRecord{}}
	handler := NewAnalytics(analyticsRepositoryStub{view: view}, func(*http.Request) (string, error) { return "account", nil }, config)
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/account/analytics", nil))
	var decoded store.AccountAnalytics
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || decoded.Customer.TotalSpentWei != "20" || decoded.Provider.GrossRevenueWei != "19" {
		t.Fatalf("status=%d view=%+v", response.Code, decoded)
	}
}
