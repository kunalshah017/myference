//go:build integration

package settlement

import (
	"bytes"
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	_ "github.com/jackc/pgx/v5/stdlib"
	v1 "github.com/kunalshah017/myference/protocol/v1"
	"github.com/kunalshah017/myference/server/internal/chain"
	"github.com/kunalshah017/myference/server/internal/relay"
	"github.com/kunalshah017/myference/server/internal/store"
)

const (
	testOwnerKey    = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	testProviderKey = "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
	testMachineKey  = "5de4111afa1c4b3daadb435b6b1e20f0a32caa4eac926d28c916424f529cf3d0"
)

func TestCoordinatorObtainsMachineSignatureAndSettlesActualReceipt(t *testing.T) {
	rpcURL, databaseURL := os.Getenv("MYFERENCE_TEST_RPC_URL"), os.Getenv("MYFERENCE_TEST_DATABASE_URL")
	if rpcURL == "" || databaseURL == "" {
		t.Fatal("MYFERENCE_TEST_RPC_URL and MYFERENCE_TEST_DATABASE_URL are required")
	}
	ctx := context.Background()
	owner, err := chain.Dial(ctx, rpcURL, testOwnerKey)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close()
	provider, err := chain.Dial(ctx, rpcURL, testProviderKey)
	if err != nil {
		t.Fatal(err)
	}
	defer provider.Close()
	before, err := owner.Header(ctx)
	if err != nil {
		t.Fatal(err)
	}
	contract, err := owner.Deploy(ctx, owner.Address(), owner.Address(), owner.Address(), big.NewInt(100), 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := provider.Bind(contract); err != nil {
		t.Fatal(err)
	}
	machineKey, err := crypto.HexToECDSA(testMachineKey)
	if err != nil {
		t.Fatal(err)
	}
	machineSigner := crypto.PubkeyToAddress(machineKey.PublicKey)
	if err := owner.Deposit(ctx, big.NewInt(1_000)); err != nil {
		t.Fatal(err)
	}
	if err := provider.DepositBond(ctx, big.NewInt(100)); err != nil {
		t.Fatal(err)
	}
	if err := provider.SetProviderSigner(ctx, machineSigner, true); err != nil {
		t.Fatal(err)
	}
	offerID, modelHash, capabilityHash := crypto.Keccak256Hash([]byte("local-qwen")), crypto.Keccak256Hash([]byte("qwen")), crypto.Keccak256Hash([]byte("stream,text"))
	if err := provider.PublishOffer(ctx, offerID, modelHash, capabilityHash, big.NewInt(100_000_000), big.NewInt(0), big.NewInt(0)); err != nil {
		t.Fatal(err)
	}
	sessionID, requestID := crypto.Keccak256Hash([]byte("coordinator-session")), crypto.Keccak256Hash([]byte("coordinator-request"))
	header, _ := owner.Header(ctx)
	if err := owner.OpenSession(ctx, sessionID, big.NewInt(500), header.Time+3600); err != nil {
		t.Fatal(err)
	}

	repository, err := store.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	for _, name := range []string{"000001_control_plane.sql", "000002_inference.sql", "000003_chain_index.sql", "000006_request_submission.sql", "000007_provider_operations.sql", "000008_machine_signers.sql", "000009_receipt_coordination.sql"} {
		if err := repository.ApplyMigration(ctx, filepath.Join("..", "..", "..", "migrations", name)); err != nil {
			t.Fatal(err)
		}
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `TRUNCATE settlement_queue,receipt_proposals,receipt_nonces,inference_reservations,provider_routing_state,outbox,requests,sessions,offers,backends,machines,accounts,chain_settlements,chain_sessions,chain_offers,chain_provider_signers,chain_accounts,chain_logs,chain_blocks,chain_cursors CASCADE`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO accounts(id,wallet_address) VALUES ('customer',$1),('provider',$2)`, owner.Address().Hex(), provider.Address().Hex()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO machines(id,account_id,name,signer_address) VALUES ('machine','provider','headless',$1)`, machineSigner.Hex()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO backends(id,machine_id,kind,model) VALUES ('backend','machine','ollama','qwen')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO offers(id,backend_id,version,input_per_million,output_per_million,compute_per_second) VALUES ('local-qwen','backend',1,100000000,0,0)`); err != nil {
		t.Fatal(err)
	}
	indexer, err := chain.OpenIndexer(ctx, chain.IndexerConfig{RPCURL: rpcURL, DatabaseURL: databaseURL, Contract: contract, StartBlock: before.Number.Uint64() + 1})
	if err != nil {
		t.Fatal(err)
	}
	defer indexer.Close()
	if err := indexer.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	if err := repository.UpsertRoutingState(ctx, store.RoutingState{MachineID: "machine", OfferID: "local-qwen", Model: "qwen", BackendKind: "ollama", Capabilities: []string{"stream", "text"}, PriceVersion: 1, ConfirmedBond: true, Healthy: true, Capacity: 1, MaximumCost: 100, InputPerMillion: 100_000_000}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO requests(id,session_id,state,machine_id,offer_id,price_version,maximum_spend,maximum_input_tokens,maximum_output_tokens,maximum_compute_milliseconds,offer_hash,model_hash,capability_hash) VALUES ($1,$2,'streaming','machine','local-qwen',1,100,1,1,1,$3,$4,$5)`, requestID.Hex(), sessionID.Hex(), offerID.Hex(), modelHash.Hex(), capabilityHash.Hex()); err != nil {
		t.Fatal(err)
	}

	hub := relay.NewHub(func(context.Context, string) (string, error) { return "machine", nil }, relay.Options{})
	relayServer := httptest.NewTLSServer(hub)
	defer relayServer.Close()
	client := relayServer.Client()
	client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true}
	connection, _, err := websocket.Dial(ctx, "wss"+strings.TrimPrefix(relayServer.URL, "https"), &websocket.DialOptions{HTTPClient: client, HTTPHeader: http.Header{"Authorization": {"Bearer token"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(websocket.StatusNormalClosure, "")
	hello, _ := v1.NewEnvelope("hello", v1.MessageHello, &v1.Hello{MachineID: "machine"})
	raw, _ := json.Marshal(hello)
	if err := connection.Write(ctx, websocket.MessageText, raw); err != nil {
		t.Fatal(err)
	}
	signed := make(chan error, 1)
	go func() {
		_, payload, readErr := connection.Read(ctx)
		if readErr != nil {
			signed <- readErr
			return
		}
		envelope, decodeErr := v1.DecodeEnvelope(bytes.NewReader(payload), 1<<20)
		if decodeErr != nil {
			signed <- decodeErr
			return
		}
		var proposal v1.ReceiptProposal
		if envelope.Type != v1.MessageReceiptProposal {
			signed <- v1.ErrInvalidMessage
			return
		}
		if err := envelope.DecodeBody(&proposal); err != nil {
			signed <- err
			return
		}
		signature, signErr := v1.SignReceipt(proposal.Receipt, proposal.ChainID, proposal.Contract, machineKey)
		if signErr != nil {
			signed <- signErr
			return
		}
		message, _ := v1.NewEnvelope("signature", v1.MessageReceiptSignature, &v1.ReceiptSignature{RequestID: proposal.RequestID, Signer: v1.Address(machineSigner), Signature: signature})
		encoded, _ := json.Marshal(message)
		signed <- connection.Write(ctx, websocket.MessageText, encoded)
	}()
	queue, err := chain.OpenSettlementQueue(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer queue.Close()
	coordinator := NewCoordinator(Config{SignatureTimeout: 5 * time.Second}, repository, hub, queue, owner)
	completed := store.ReceiptProposal{RequestID: requestID.Hex(), SessionID: sessionID.Hex(), MachineID: "machine", OfferID: "local-qwen", Model: "qwen", PriceVersion: 1, InputTokens: 1, InputHash: [32]byte{1}, OutputHash: [32]byte{2}, CompletedAt: time.Unix(int64(header.Time), 0)}
	if err := coordinator.Complete(ctx, completed); err != nil {
		t.Fatal(err)
	}
	if err := <-signed; err != nil {
		t.Fatal(err)
	}
	if _, err := queue.SettleBatch(ctx, owner, 10); err != nil {
		t.Fatal(err)
	}
	if err := indexer.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	var state, claimable string
	if err := db.QueryRowContext(ctx, `SELECT state FROM requests WHERE id=$1`, requestID.Hex()).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT claimable::text FROM chain_accounts WHERE chain_id=31337 AND contract_address=$1 AND lower(address)=lower($2)`, contract.Hex(), provider.Address().Hex()).Scan(&claimable); err != nil {
		t.Fatal(err)
	}
	if state != "settled" || claimable != "95" {
		t.Fatalf("state=%q provider claimable=%q", state, claimable)
	}
	if contract == (common.Address{}) {
		t.Fatal("zero contract")
	}
}
