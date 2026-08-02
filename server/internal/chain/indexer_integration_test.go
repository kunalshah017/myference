//go:build integration

package chain

import (
	"context"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
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
	for _, name := range []string{"000001_control_plane.sql", "000002_inference.sql", "000003_chain_index.sql"} {
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
