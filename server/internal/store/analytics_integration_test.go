package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestAccountAnalyticsAggregatesConfirmedUsageRevenueAndSlashes(t *testing.T) {
	databaseURL := os.Getenv("MYFERENCE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("MYFERENCE_TEST_DATABASE_URL is required")
	}
	ctx := context.Background()
	repository, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	for _, name := range []string{"000001_control_plane.sql", "000002_inference.sql", "000003_chain_index.sql", "000007_provider_operations.sql", "000013_account_analytics.sql"} {
		if err := repository.ApplyMigration(ctx, filepath.Join("..", "..", "..", "migrations", name)); err != nil {
			t.Fatal(err)
		}
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	suffix := time.Now().Format("150405000000000")
	customerID, providerID := "analytics-customer-"+suffix, "analytics-provider-"+suffix
	walletSeed := time.Now().UnixNano()
	customerWallet, providerWallet := fmt.Sprintf("0x%040x", walletSeed), fmt.Sprintf("0x%040x", walletSeed+1)
	contract, sessionID, requestID := "0x3333333333333333333333333333333333333333", "session-"+suffix, "request-"+suffix
	machineID, backendID, offerID := "machine-"+suffix, "backend-"+suffix, "offer-"+suffix
	for _, id := range []string{customerID, providerID} {
		_, _ = db.ExecContext(ctx, `DELETE FROM accounts WHERE id=$1`, id)
	}
	if err := repository.CreateAccount(ctx, Account{ID: customerID, WalletAddress: customerWallet}); err != nil {
		t.Fatal(err)
	}
	if err := repository.CreateAccount(ctx, Account{ID: providerID, WalletAddress: providerWallet}); err != nil {
		t.Fatal(err)
	}
	if err := repository.CreateMachine(ctx, Machine{ID: machineID, AccountID: providerID, Name: "analytics-node"}); err != nil {
		t.Fatal(err)
	}
	if err := repository.CreateBackend(ctx, Backend{ID: backendID, MachineID: machineID, Kind: "ollama", Model: "qwen"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO offers(id,backend_id,version,input_per_million,output_per_million,compute_per_second) VALUES($1,$2,1,1,1,1)`, offerID, backendID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO sessions(id,account_id,state,confirmed_balance_wei) VALUES($1,$2,'open',100)`, sessionID, customerID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO requests(id,session_id,state,machine_id,offer_id,price_version,maximum_spend) VALUES($1,$2,'settled',$3,$4,1,100)`, requestID, sessionID, machineID, offerID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO receipt_proposals(request_id,session_id,machine_id,offer_id,model,price_version,input_tokens,output_tokens,compute_milliseconds,input_hash,output_hash,completed_at) VALUES($1,$2,$3,$4,'qwen',1,11,7,90,decode(repeat('00',32),'hex'),decode(repeat('11',32),'hex'),now())`, requestID, sessionID, machineID, offerID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO chain_settlements(chain_id,contract_address,request_id,session_id,provider,provider_amount,fee_amount,transaction_hash,block_number,indexed_at) VALUES(10143,$1,$2,$3,$4,19,1,'0xsettled',12,now())`, contract, requestID, sessionID, providerWallet); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO chain_slashes(chain_id,contract_address,request_id,provider,amount,block_number,transaction_hash,indexed_at) VALUES(10143,$1,$2,$3,5,13,'0xslashed',now())`, contract, requestID, providerWallet); err != nil {
		t.Fatal(err)
	}

	customer, err := repository.AccountAnalytics(ctx, customerID, 10143, contract)
	if err != nil {
		t.Fatal(err)
	}
	if customer.Customer.SettledRequests != 1 || customer.Customer.InputTokens != 11 || customer.Customer.TotalSpentWei != "20" {
		t.Fatalf("customer analytics=%+v", customer.Customer)
	}
	provider, err := repository.AccountAnalytics(ctx, providerID, 10143, contract)
	if err != nil {
		t.Fatal(err)
	}
	if provider.Provider.SettledRequests != 1 || provider.Provider.GrossRevenueWei != "19" || provider.Provider.TotalSlashedWei != "5" || len(provider.Slashes) != 1 {
		t.Fatalf("provider analytics=%+v slashes=%+v", provider.Provider, provider.Slashes)
	}
}
