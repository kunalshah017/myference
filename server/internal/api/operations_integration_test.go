package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/kunalshah017/myference/server/internal/store"
)

func TestOperationsReturnsIndexedEconomicsMachinesOffersAndEarnings(t *testing.T) {
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
	for _, migration := range []string{"000001_control_plane.sql", "000002_inference.sql", "000003_chain_index.sql", "000007_provider_operations.sql"} {
		if err := repository.ApplyMigration(ctx, filepath.Join("..", "..", "..", "migrations", migration)); err != nil {
			t.Fatal(err)
		}
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	suffix := time.Now().Format("150405.000000000")
	accountID, machineID, backendID := "ops-account-"+suffix, "ops-machine-"+suffix, "ops-backend-"+suffix
	wallet := "0x3333333333333333333333333333333333333333"
	contract := "0x4444444444444444444444444444444444444444"
	if _, err := db.ExecContext(ctx, `DELETE FROM accounts WHERE wallet_address=$1`, wallet); err != nil {
		t.Fatal(err)
	}
	if err := repository.CreateAccount(ctx, store.Account{ID: accountID, WalletAddress: wallet}); err != nil {
		t.Fatal(err)
	}
	if err := repository.CreateMachine(ctx, store.Machine{ID: machineID, AccountID: accountID, Name: "studio-node"}); err != nil {
		t.Fatal(err)
	}
	if err := repository.CreateBackend(ctx, store.Backend{ID: backendID, MachineID: machineID, Kind: "ollama", Model: "qwen"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO chain_accounts(chain_id,contract_address,address,customer_balance,provider_bond,claimable) VALUES (10143,$1,$2,1000,2000,3000) ON CONFLICT (chain_id,contract_address,address) DO UPDATE SET customer_balance=1000,provider_bond=2000,claimable=3000`, contract, wallet); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO chain_sessions(chain_id,contract_address,session_id,customer,allowance,spent,expires_at,close_available_at,finalized) VALUES (10143,$1,$2,$3,500,20,2000000000,0,false) ON CONFLICT DO NOTHING`, contract, "0x"+suffix, wallet); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO chain_offers(chain_id,contract_address,provider,offer_id,version,model_hash,capability_hash,input_per_million,output_per_million,compute_per_second) VALUES (10143,$1,$2,$3,2,$4,$5,10,20,30) ON CONFLICT DO NOTHING`, contract, wallet, "0xoffer"+suffix, "0xmodel"+suffix, "0xcap"+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO chain_settlements(chain_id,contract_address,request_id,session_id,provider,provider_amount,fee_amount,transaction_hash) VALUES (10143,$1,$2,$3,$4,95,5,$5) ON CONFLICT DO NOTHING`, contract, "0xrequest"+suffix, "0xsession"+suffix, wallet, "0xtx"+suffix); err != nil {
		t.Fatal(err)
	}

	handler := NewOperations(repository, func(*http.Request) (string, error) { return accountID, nil }, OperationsConfig{ChainID: 10143, ContractAddress: contract, ExplorerURL: "https://testnet.monadexplorer.com", Confirmations: 2})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	response, err := http.Get(server.URL + "/api/account/operations")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var view store.AccountOperations
	if err := json.NewDecoder(response.Body).Decode(&view); err != nil {
		t.Fatal(err)
	}
	if view.CustomerBalanceWei != "1000" || view.ProviderBondWei != "2000" || view.ClaimableWei != "3000" || view.ProviderEarningsWei != "95" {
		t.Fatalf("unexpected economics: %+v", view)
	}
	if view.ContractAddress != contract || view.ChainID != 10143 || len(view.Machines) != 1 || view.Machines[0].Name != "studio-node" || len(view.Sessions) != 1 || len(view.Offers) != 1 {
		t.Fatalf("unexpected operations view: %+v", view)
	}
}
