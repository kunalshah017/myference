package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kunalshah017/myference/server/internal/store"
)

func TestMarketplaceServesOnlyPersistedOffersAndAccountActivity(t *testing.T) {
	databaseURL := os.Getenv("MYFERENCE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("MYFERENCE_TEST_DATABASE_URL is required")
	}
	ctx := context.Background()
	repository, err := store.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repository.Close() })
	for _, migration := range []string{"000001_control_plane.sql", "000002_inference.sql", "000003_chain_index.sql"} {
		if err := repository.ApplyMigration(ctx, filepath.Join("..", "..", "..", "migrations", migration)); err != nil {
			t.Fatal(err)
		}
	}
	suffix := time.Now().Format("150405.000000000")
	accountID, machineID, backendID, offerID := "market-acct-"+suffix, "market-machine-"+suffix, "market-backend-"+suffix, "market-offer-"+suffix
	model := "org/qwen-market-" + suffix
	if err := repository.CreateAccount(ctx, store.Account{ID: accountID, WalletAddress: "0x" + time.Now().Format("0102150405000000000000000000000000000000")}); err != nil {
		t.Fatal(err)
	}
	if err := repository.CreateMachine(ctx, store.Machine{ID: machineID, AccountID: accountID, Name: "real-provider"}); err != nil {
		t.Fatal(err)
	}
	if err := repository.CreateBackend(ctx, store.Backend{ID: backendID, MachineID: machineID, Kind: "ollama", Model: model}); err != nil {
		t.Fatal(err)
	}
	if err := repository.CreateOffer(ctx, store.Offer{ID: offerID, BackendID: backendID, Version: 1, InputPerMillion: 10, OutputPerMillion: 20, ComputePerSecond: 30}); err != nil {
		t.Fatal(err)
	}
	if err := repository.UpsertRoutingState(ctx, store.RoutingState{MachineID: machineID, OfferID: offerID, Model: model, BackendKind: "ollama", Capabilities: []string{"chat", "stream"}, PriceVersion: 1, MaximumCost: 100, InputPerMillion: 10, OutputPerMillion: 20, ComputePerSecond: 30, ConfirmedBond: true, Healthy: true, Capacity: 2, SuccessBasisPoints: 9900}); err != nil {
		t.Fatal(err)
	}
	if err := repository.CreateSession(ctx, store.Session{ID: "market-session-" + suffix, AccountID: accountID, State: "open"}); err != nil {
		t.Fatal(err)
	}
	if err := repository.CreateRequest(ctx, store.Request{ID: "market-request-" + suffix, SessionID: "market-session-" + suffix, State: "created"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.MarketplaceModel(ctx, model, 30*time.Second); err != nil {
		t.Fatalf("query persisted model: %v", err)
	}

	handler := NewMarketplace(repository, func(*http.Request) (string, error) { return accountID, nil }, 30*time.Second)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	response, err := http.Get(server.URL + "/api/models")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var models []store.MarketModel
	if err := json.NewDecoder(response.Body).Decode(&models); err != nil {
		t.Fatal(err)
	}
	var found *store.MarketModel
	for i := range models {
		if models[i].Model == model {
			found = &models[i]
		}
	}
	if found == nil || found.AvailableProviders != 1 || found.TotalCapacity != 2 || found.MinimumInputPerMillion != "10" {
		t.Fatalf("persisted model missing or altered: %+v", found)
	}
	detailResponse, err := http.Get(server.URL + "/api/models/" + url.PathEscape(model))
	if err != nil {
		t.Fatal(err)
	}
	defer detailResponse.Body.Close()
	if detailResponse.StatusCode != http.StatusOK {
		var problem map[string]any
		_ = json.NewDecoder(detailResponse.Body).Decode(&problem)
		t.Fatalf("detail status=%d body=%v", detailResponse.StatusCode, problem)
	}
	var detail store.MarketModelDetail
	if err := json.NewDecoder(detailResponse.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if len(detail.Offers) != 1 || detail.Offers[0].MachineID != machineID || detail.Offers[0].InputPerMillion != "10" || detail.Offers[0].PriceVersion != 1 {
		t.Fatalf("unexpected detail: %+v", detail)
	}
	activityResponse, err := http.Get(server.URL + "/api/activity")
	if err != nil {
		t.Fatal(err)
	}
	defer activityResponse.Body.Close()
	var activity []store.AccountActivity
	if err := json.NewDecoder(activityResponse.Body).Decode(&activity); err != nil {
		t.Fatal(err)
	}
	if len(activity) == 0 || activity[0].AccountID != accountID {
		t.Fatalf("account activity missing: %+v", activity)
	}
}
