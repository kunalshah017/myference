//go:build integration

package chain

import (
	"context"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	v1 "github.com/kunalshah017/myference/protocol/v1"
	"github.com/kunalshah017/myference/server/internal/store"
)

const (
	anvilOwnerKey    = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	anvilProviderKey = "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
)

func TestClientDeploysAndSettlesActualMyferenceContract(t *testing.T) {
	rpcURL := os.Getenv("MYFERENCE_TEST_RPC_URL")
	databaseURL := os.Getenv("MYFERENCE_TEST_DATABASE_URL")
	if rpcURL == "" || databaseURL == "" {
		t.Fatal("MYFERENCE_TEST_RPC_URL and MYFERENCE_TEST_DATABASE_URL are required")
	}
	ctx := context.Background()
	owner, err := Dial(ctx, rpcURL, anvilOwnerKey)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	provider, err := Dial(ctx, rpcURL, anvilProviderKey)
	if err != nil {
		t.Fatal(err)
	}
	defer provider.Close()
	beforeDeployment, err := owner.Header(ctx)
	if err != nil {
		t.Fatal(err)
	}
	address, err := owner.Deploy(ctx, owner.Address(), owner.Address(), owner.Address(), big.NewInt(100), 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := provider.Bind(address); err != nil {
		t.Fatal(err)
	}
	terms, err := owner.ReceiptTerms(ctx)
	if err != nil || terms.ChainID != 31337 || terms.Contract != address || terms.SettlementSigner != owner.Address() || terms.FeeBasisPoints != 500 || terms.FeeVersion != 1 {
		t.Fatalf("receipt terms=%+v err=%v", terms, err)
	}
	if err := owner.Deposit(ctx, big.NewInt(1_000)); err != nil {
		t.Fatal(err)
	}
	if err := provider.DepositBond(ctx, big.NewInt(100)); err != nil {
		t.Fatal(err)
	}
	if err := provider.SetProviderSigner(ctx, owner.Address(), true); err != nil {
		t.Fatal(err)
	}
	offerID := crypto.Keccak256Hash([]byte("offer"))
	modelHash := crypto.Keccak256Hash([]byte("qwen"))
	capabilityHash := crypto.Keccak256Hash([]byte("text-stream"))
	if err := provider.PublishOffer(ctx, offerID, modelHash, capabilityHash, big.NewInt(1_000_000), big.NewInt(1_000_000), big.NewInt(0)); err != nil {
		t.Fatal(err)
	}
	sessionID := crypto.Keccak256Hash([]byte("session"))
	header, err := owner.Header(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := owner.OpenSession(ctx, sessionID, big.NewInt(500), header.Time+3600); err != nil {
		t.Fatal(err)
	}
	if err := owner.RequestWithdrawal(ctx, big.NewInt(100)); err != nil {
		t.Fatal(err)
	}
	if err := owner.Claim(ctx); err != nil {
		t.Fatal(err)
	}
	if err := provider.RequestBondExit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := owner.RequestSessionClose(ctx, sessionID); err != nil {
		t.Fatal(err)
	}
	header, _ = owner.Header(ctx)
	receipt := Receipt{
		RequestId: crypto.Keccak256Hash([]byte("request")), SessionId: sessionID,
		Customer: owner.Address(), Provider: provider.Address(), SettlementSigner: owner.Address(),
		OfferId: offerID, PriceVersion: 1, ModelHash: modelHash, CapabilityHash: capabilityHash,
		InputTokens: 10, OutputTokens: 10, MaximumCharge: 100, TotalCharge: 20,
		FeeBasisPoints: 500, FeeVersion: 1, Status: 1, CompletedAt: header.Time,
		InputHash: crypto.Keccak256Hash([]byte("input")), OutputHash: crypto.Keccak256Hash([]byte("output")), Nonce: 1,
	}
	protocolReceipt := v1.Receipt{
		RequestID: v1.Hash(receipt.RequestId), SessionID: v1.Hash(receipt.SessionId), Customer: v1.Address(receipt.Customer), Provider: v1.Address(receipt.Provider), SettlementSigner: v1.Address(receipt.SettlementSigner), OfferID: v1.Hash(receipt.OfferId), PriceVersion: receipt.PriceVersion, ModelHash: v1.Hash(receipt.ModelHash), CapabilityHash: v1.Hash(receipt.CapabilityHash), InputTokens: receipt.InputTokens, OutputTokens: receipt.OutputTokens, ComputeMilliseconds: receipt.ComputeMilliseconds, MaximumCharge: receipt.MaximumCharge, TotalCharge: receipt.TotalCharge, FeeBasisPoints: receipt.FeeBasisPoints, FeeVersion: receipt.FeeVersion, Status: v1.ReceiptStatus(receipt.Status), CompletedAt: receipt.CompletedAt, InputHash: v1.Hash(receipt.InputHash), OutputHash: v1.Hash(receipt.OutputHash), Nonce: receipt.Nonce,
	}
	offlineDigest, err := v1.ReceiptDigest(protocolReceipt, 31337, v1.Address(address))
	if err != nil {
		t.Fatal(err)
	}
	contractDigest, err := owner.HashReceipt(ctx, receipt)
	if err != nil || [32]byte(offlineDigest) != contractDigest {
		t.Fatalf("offline digest=%x contract=%x err=%v", offlineDigest, contractDigest, err)
	}
	providerSignature, err := provider.SignReceipt(ctx, receipt)
	if err != nil {
		t.Fatal(err)
	}
	settlementSignature, err := owner.SignReceipt(ctx, receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := owner.SettleReceipt(ctx, receipt, providerSignature, settlementSignature); err != nil {
		t.Fatal(err)
	}
	settled, claimable, err := owner.SettlementState(ctx, receipt.RequestId, provider.Address())
	if err != nil {
		t.Fatal(err)
	}
	if !settled || claimable.Cmp(big.NewInt(19)) != 0 {
		t.Fatalf("settled=%v claimable=%s contract=%s", settled, claimable, address)
	}
	if address == (common.Address{}) {
		t.Fatal("zero deployment address")
	}
	repository, err := store.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	for _, migration := range []string{"000001_control_plane.sql", "000002_inference.sql", "000003_chain_index.sql", "000006_request_submission.sql", "000007_provider_operations.sql", "000008_machine_signers.sql", "000013_account_analytics.sql"} {
		if err := repository.ApplyMigration(ctx, filepath.Join("..", "..", "..", "migrations", migration)); err != nil {
			t.Fatal(err)
		}
	}
	repository.Close()
	queue, err := OpenSettlementQueue(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer queue.Close()
	if _, err := queue.db.ExecContext(ctx, "TRUNCATE settlement_queue"); err != nil {
		t.Fatal(err)
	}
	suffix := time.Now().Format("150405.000000000")
	receipt.RequestId = crypto.Keccak256Hash([]byte("request-batch-" + suffix))
	receipt.Nonce = 2
	if _, err := queue.db.ExecContext(ctx, `INSERT INTO accounts(id,wallet_address) VALUES ($1,$2)`, "settle-account-"+suffix, "settle-wallet-"+suffix); err != nil {
		t.Fatal(err)
	}
	chainAccountID := "chain-customer-" + suffix
	if err := queue.db.QueryRowContext(ctx, `INSERT INTO accounts(id,wallet_address) VALUES ($1,$2) ON CONFLICT(wallet_address) DO UPDATE SET wallet_address=EXCLUDED.wallet_address RETURNING id`, chainAccountID, owner.Address().Hex()).Scan(&chainAccountID); err != nil {
		t.Fatal(err)
	}
	if _, err := queue.db.ExecContext(ctx, `INSERT INTO sessions(id,account_id,state) VALUES ($1,$2,'open')`, "settle-session-"+suffix, "settle-account-"+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := queue.db.ExecContext(ctx, `INSERT INTO requests(id,session_id,state) VALUES ($1,$2,'completed')`, hashHex(receipt.RequestId), "settle-session-"+suffix); err != nil {
		t.Fatal(err)
	}
	providerSignature, _ = provider.SignReceipt(ctx, receipt)
	settlementSignature, _ = owner.SignReceipt(ctx, receipt)
	if err := queue.Enqueue(ctx, SignedReceipt{Receipt: receipt, ProviderSignature: providerSignature, SettlementSignature: settlementSignature}); err != nil {
		t.Fatal(err)
	}
	var requestState string
	if err := queue.db.QueryRowContext(ctx, `SELECT state FROM requests WHERE id=$1`, hashHex(receipt.RequestId)).Scan(&requestState); err != nil || requestState != "signed" {
		t.Fatalf("request after co-sign=%q err=%v", requestState, err)
	}
	transactionHash, err := queue.SettleBatch(ctx, owner, 10)
	if err != nil || transactionHash == "" {
		t.Fatalf("transaction=%q err=%v", transactionHash, err)
	}
	var state string
	if err := queue.db.QueryRowContext(ctx, `SELECT state FROM settlement_queue WHERE request_id=$1`, hashHex(receipt.RequestId)).Scan(&state); err != nil || state != "broadcasting" {
		t.Fatalf("queue state=%q err=%v", state, err)
	}
	if err := queue.db.QueryRowContext(ctx, `SELECT state FROM requests WHERE id=$1`, hashHex(receipt.RequestId)).Scan(&requestState); err != nil || requestState != "submitted" {
		t.Fatalf("request after broadcast=%q err=%v", requestState, err)
	}
	batchRequestID := receipt.RequestId
	indexer, err := OpenIndexer(ctx, IndexerConfig{RPCURL: rpcURL, DatabaseURL: databaseURL, Contract: address, StartBlock: beforeDeployment.Number.Uint64() + 1})
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()
	if err := indexer.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	var operationalAccount, operationalState, operationalBalance string
	if err := queue.db.QueryRowContext(ctx, `SELECT account_id,state,confirmed_balance_wei::text FROM sessions WHERE id=$1`, sessionID.Hex()).Scan(&operationalAccount, &operationalState, &operationalBalance); err != nil {
		t.Fatal(err)
	}
	if operationalAccount != chainAccountID || operationalState != "closing" || operationalBalance != "460" {
		t.Fatalf("operational session account=%q state=%q balance=%q", operationalAccount, operationalState, operationalBalance)
	}
	var signerAllowed bool
	if err := queue.db.QueryRowContext(ctx, `SELECT allowed FROM chain_provider_signers WHERE chain_id=$1 AND contract_address=$2 AND lower(provider)=lower($3) AND lower(signer)=lower($4)`, indexer.chainID.String(), address.Hex(), provider.Address().Hex(), owner.Address().Hex()).Scan(&signerAllowed); err != nil || !signerAllowed {
		t.Fatalf("indexed signer allowed=%v err=%v", signerAllowed, err)
	}
	if err := queue.db.QueryRowContext(ctx, `SELECT state FROM settlement_queue WHERE request_id=$1`, hashHex(batchRequestID)).Scan(&state); err != nil || state != "settled" {
		t.Fatalf("queue after confirmation=%q err=%v", state, err)
	}
	if err := queue.db.QueryRowContext(ctx, `SELECT state FROM requests WHERE id=$1`, hashHex(batchRequestID)).Scan(&requestState); err != nil || requestState != "settled" {
		t.Fatalf("request after confirmation=%q err=%v", requestState, err)
	}
	receipt.RequestId = crypto.Keccak256Hash([]byte("request-recovery"))
	receipt.Nonce = 3
	providerSignature, _ = provider.SignReceipt(ctx, receipt)
	settlementSignature, _ = owner.SignReceipt(ctx, receipt)
	if err := queue.Enqueue(ctx, SignedReceipt{Receipt: receipt, ProviderSignature: providerSignature, SettlementSignature: settlementSignature}); err != nil {
		t.Fatal(err)
	}
	prepared, err := owner.PrepareSettlement(ctx, []Receipt{receipt}, [][]byte{providerSignature}, [][]byte{settlementSignature})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := prepared.MarshalBinary()
	if _, err := queue.db.ExecContext(ctx, `UPDATE settlement_queue SET state='broadcasting',transaction_hash=$2,raw_transaction=$3 WHERE request_id=$1`, hashHex(receipt.RequestId), prepared.Hash().Hex(), raw); err != nil {
		t.Fatal(err)
	}
	if recoveredHash, err := queue.SettleBatch(ctx, owner, 10); err != nil || recoveredHash != prepared.Hash().Hex() {
		t.Fatalf("recovered=%q want=%q err=%v", recoveredHash, prepared.Hash().Hex(), err)
	}
	if err := indexer.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	if err := queue.db.QueryRowContext(ctx, `SELECT state FROM settlement_queue WHERE request_id=$1`, hashHex(receipt.RequestId)).Scan(&state); err != nil || state != "settled" {
		t.Fatalf("queue after confirmation=%q err=%v", state, err)
	}
	for table, minimum := range map[string]int{"chain_accounts": 2, "chain_offers": 1, "chain_sessions": 1, "chain_settlements": 3} {
		var count int
		query := "SELECT count(*) FROM " + table + " WHERE chain_id=$1 AND contract_address=$2" // fixed internal table names
		if err := indexer.db.QueryRowContext(ctx, query, indexer.chainID.String(), address.Hex()).Scan(&count); err != nil || count < minimum {
			t.Fatalf("%s count=%d err=%v", table, count, err)
		}
	}
	var customerBalance, ownerClaimable, providerBond, providerClaimable, spent string
	var bondExit, closeAvailable uint64
	if err := indexer.db.QueryRowContext(ctx, `SELECT customer_balance::text,claimable::text FROM chain_accounts WHERE chain_id=$1 AND contract_address=$2 AND address=$3`, indexer.chainID.String(), address.Hex(), owner.Address().Hex()).Scan(&customerBalance, &ownerClaimable); err != nil {
		t.Fatal(err)
	}
	if err := indexer.db.QueryRowContext(ctx, `SELECT provider_bond::text,claimable::text,bond_exit_available_at FROM chain_accounts WHERE chain_id=$1 AND contract_address=$2 AND address=$3`, indexer.chainID.String(), address.Hex(), provider.Address().Hex()).Scan(&providerBond, &providerClaimable, &bondExit); err != nil {
		t.Fatal(err)
	}
	if err := indexer.db.QueryRowContext(ctx, `SELECT spent::text,close_available_at FROM chain_sessions WHERE chain_id=$1 AND contract_address=$2 AND session_id=$3`, indexer.chainID.String(), address.Hex(), sessionID.Hex()).Scan(&spent, &closeAvailable); err != nil {
		t.Fatal(err)
	}
	if customerBalance != "400" || ownerClaimable != "3" || providerBond != "100" || providerClaimable != "57" || spent != "60" || bondExit == 0 || closeAvailable == 0 {
		t.Fatalf("bad economic projection customer=%s ownerClaimable=%s bond=%s providerClaimable=%s spent=%s bondExit=%d close=%d", customerBalance, ownerClaimable, providerBond, providerClaimable, spent, bondExit, closeAvailable)
	}
}
