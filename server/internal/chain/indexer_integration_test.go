//go:build integration

package chain

import (
	"context"
	"database/sql"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/kunalshah017/myference/server/internal/chain/bindings"
	"github.com/kunalshah017/myference/server/internal/store"
)

func TestIndexerRestartsIdempotentlyAndRewindsAnvilReorg(t *testing.T) {
	rpcURL, databaseURL := os.Getenv("MYFERENCE_TEST_RPC_URL"), os.Getenv("MYFERENCE_TEST_DATABASE_URL")
	if rpcURL == "" || databaseURL == "" {
		t.Fatal("MYFERENCE_TEST_RPC_URL and MYFERENCE_TEST_DATABASE_URL are required")
	}
	ctx := context.Background()
	repository, err := store.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"000001_control_plane.sql", "000002_inference.sql", "000003_chain_index.sql", "000007_provider_operations.sql", "000008_machine_signers.sql", "000013_account_analytics.sql"} {
		if err := repository.ApplyMigration(ctx, filepath.Join("..", "..", "..", "migrations", name)); err != nil {
			t.Fatal(err)
		}
	}
	repository.Close()
	owner, err := Dial(ctx, rpcURL, anvilOwnerKey)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	before, _ := owner.Header(ctx)
	contract, err := owner.Deploy(ctx, owner.Address(), owner.Address(), owner.Address(), big.NewInt(100), 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	rpcClient, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		t.Fatal(err)
	}
	defer rpcClient.Close()
	var snapshot string
	if err := rpcClient.CallContext(ctx, &snapshot, "evm_snapshot"); err != nil {
		t.Fatal(err)
	}
	if err := owner.Deposit(ctx, big.NewInt(100)); err != nil {
		t.Fatal(err)
	}
	indexer, err := OpenIndexer(ctx, IndexerConfig{RPCURL: rpcURL, DatabaseURL: databaseURL, Contract: contract, StartBlock: before.Number.Uint64() + 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := indexer.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	logsBefore := indexer.LogCount(ctx)
	indexer.Close()
	restarted, err := OpenIndexer(ctx, IndexerConfig{RPCURL: rpcURL, DatabaseURL: databaseURL, Contract: contract, StartBlock: before.Number.Uint64() + 1})
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	if err := restarted.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	if logsAfter := restarted.LogCount(ctx); logsAfter != logsBefore || logsAfter < 2 {
		t.Fatalf("idempotency logs before=%d after=%d", logsBefore, logsAfter)
	}
	var balance string
	if err := restarted.db.QueryRowContext(ctx, `SELECT customer_balance::text FROM chain_accounts WHERE chain_id=$1 AND contract_address=$2 AND address=$3`, restarted.chainID.String(), contract.Hex(), owner.Address().Hex()).Scan(&balance); err != nil || balance != "100" {
		t.Fatalf("balance=%q err=%v", balance, err)
	}
	var reverted bool
	if err := rpcClient.CallContext(ctx, &reverted, "evm_revert", snapshot); err != nil || !reverted {
		t.Fatalf("revert=%v err=%v", reverted, err)
	}
	if err := owner.Deposit(ctx, big.NewInt(40)); err != nil {
		t.Fatal(err)
	}
	if err := restarted.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	if err := restarted.db.QueryRowContext(ctx, `SELECT customer_balance::text FROM chain_accounts WHERE chain_id=$1 AND contract_address=$2 AND address=$3`, restarted.chainID.String(), contract.Hex(), owner.Address().Hex()).Scan(&balance); err != nil || balance != "40" {
		t.Fatalf("reorg balance=%q err=%v", balance, err)
	}
}

func TestIndexerPersistsProviderSlashedEvent(t *testing.T) {
	databaseURL := os.Getenv("MYFERENCE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("MYFERENCE_TEST_DATABASE_URL is required")
	}
	ctx := context.Background()
	repository, err := store.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"000001_control_plane.sql", "000002_inference.sql", "000003_chain_index.sql", "000007_provider_operations.sql", "000008_machine_signers.sql", "000013_account_analytics.sql"} {
		if err := repository.ApplyMigration(ctx, filepath.Join("..", "..", "..", "migrations", name)); err != nil {
			t.Fatal(err)
		}
	}
	repository.Close()
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	contract := common.HexToAddress("0x4444444444444444444444444444444444444444")
	provider := common.HexToAddress("0x5555555555555555555555555555555555555555")
	fee := common.HexToAddress("0x6666666666666666666666666666666666666666")
	requestID := common.HexToHash("0x1234")
	_, _ = db.ExecContext(ctx, `DELETE FROM chain_slashes WHERE chain_id=10143 AND contract_address=$1`, contract.Hex())
	_, err = db.ExecContext(ctx, `INSERT INTO chain_accounts(chain_id,contract_address,address,provider_bond) VALUES(10143,$1,$2,10) ON CONFLICT(chain_id,contract_address,address) DO UPDATE SET provider_bond=10`, contract.Hex(), provider.Hex())
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := bindings.MyferenceMarketMetaData.GetAbi()
	if err != nil {
		t.Fatal(err)
	}
	event := parsed.Events["ProviderSlashed"]
	data, err := event.Inputs.NonIndexed().Pack(big.NewInt(5))
	if err != nil {
		t.Fatal(err)
	}
	binding, err := bindings.NewMyferenceMarket(contract, nil)
	if err != nil {
		t.Fatal(err)
	}
	indexer := &Indexer{db: db, contract: contract, binding: binding, chainID: big.NewInt(10143), feeRecipient: fee}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	log := types.Log{Address: contract, Topics: []common.Hash{event.ID, common.BytesToHash(provider.Bytes()), requestID}, Data: data, BlockNumber: 42, TxHash: common.HexToHash("0x9876")}
	if err := indexer.applyLog(ctx, tx, log); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	var amount string
	var block uint64
	if err := db.QueryRowContext(ctx, `SELECT amount::text,block_number FROM chain_slashes WHERE chain_id=10143 AND contract_address=$1 AND request_id=$2`, contract.Hex(), requestID.Hex()).Scan(&amount, &block); err != nil || amount != "5" || block != 42 {
		t.Fatalf("amount=%q block=%d err=%v", amount, block, err)
	}
}
