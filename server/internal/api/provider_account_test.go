package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kunalshah017/myference/server/internal/store"
)

type providerAccountRepositoryStub struct{}

func (providerAccountRepositoryStub) ProviderAccount(_ context.Context, accountID string, config store.ProviderAccountConfig) (store.ProviderAccount, error) {
	return store.ProviderAccount{WalletAddress: accountID, MinimumBondWei: config.MinimumBondWei, Offers: []store.EditableOffer{}}, nil
}
func (providerAccountRepositoryStub) MachineOfferVersions(_ context.Context, machineID, accountID string, _ uint64, _ string) (map[string]uint64, error) {
	return map[string]uint64{machineID + ":" + accountID: 3}, nil
}

func TestProviderAccountAPIUsesBrowserOrExactMachineIdentity(t *testing.T) {
	handler := NewProviderAccount(providerAccountRepositoryStub{},
		func(r *http.Request) (string, string, error) {
			if r.Header.Get("Authorization") == "" {
				return "", "", errors.New("missing")
			}
			return "machine-1", "account-1", nil
		},
		func(*http.Request) (string, error) { return "account-browser", nil },
		store.ProviderAccountConfig{ChainID: 10143, ContractAddress: "0xcontract", MinimumBondWei: "5"},
	)
	browser := httptest.NewRecorder()
	handler.ServeHTTP(browser, httptest.NewRequest(http.MethodGet, "/api/provider/account", nil))
	if browser.Code != http.StatusOK || !containsBody(browser, "account-browser") {
		t.Fatalf("browser=%d %s", browser.Code, browser.Body.String())
	}
	machineRequest := httptest.NewRequest(http.MethodGet, "/api/provider/machines/machine-1/offer-versions", nil)
	machineRequest.Header.Set("Authorization", "Bearer token")
	machine := httptest.NewRecorder()
	handler.ServeHTTP(machine, machineRequest)
	if machine.Code != http.StatusOK || !containsBody(machine, "machine-1:account-1") {
		t.Fatalf("machine=%d %s", machine.Code, machine.Body.String())
	}
	wrongRequest := httptest.NewRequest(http.MethodGet, "/api/provider/machines/machine-2/offer-versions", nil)
	wrongRequest.Header.Set("Authorization", "Bearer token")
	wrong := httptest.NewRecorder()
	handler.ServeHTTP(wrong, wrongRequest)
	if wrong.Code != http.StatusNotFound {
		t.Fatalf("wrong machine status=%d", wrong.Code)
	}
}

func containsBody(response *httptest.ResponseRecorder, value string) bool {
	return strings.Contains(response.Body.String(), value)
}
