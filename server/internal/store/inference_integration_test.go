package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	v1 "github.com/kunalshah017/myference/protocol/v1"
)

func TestInferenceReservationRoutingAndReceiptAreDurableAndAtomic(t *testing.T) {
	databaseURL := os.Getenv("MYFERENCE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("MYFERENCE_TEST_DATABASE_URL is required for PostgreSQL integration")
	}
	ctx := context.Background()
	s, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	for _, name := range []string{"000001_control_plane.sql", "000002_inference.sql", "000014_reservation_finality.sql"} {
		if err := s.ApplyMigration(ctx, filepath.Join("..", "..", "..", "migrations", name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.db.ExecContext(ctx, "TRUNCATE receipt_proposals, inference_reservations, provider_routing_state, outbox, requests, sessions, offers, backends, machines, accounts CASCADE"); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateAccount(ctx, Account{ID: "customer", WalletAddress: "0x1111111111111111111111111111111111111111"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateAccount(ctx, Account{ID: "provider", WalletAddress: "0x2222222222222222222222222222222222222222"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSession(ctx, Session{ID: "session-1", AccountID: "customer", State: "open"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, "UPDATE sessions SET confirmed_balance_wei = 70000000000000000000 WHERE id = 'session-1'"); err != nil {
		t.Fatal(err)
	}
	if id, balance, err := s.OpenSession(ctx, "customer"); err != nil || id != "session-1" || balance != ^uint64(0) {
		t.Fatalf("large on-chain session id=%q balance=%d err=%v", id, balance, err)
	}
	if err := s.CreateMachine(ctx, Machine{ID: "machine-1", AccountID: "provider", Name: "windows"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateBackend(ctx, Backend{ID: "backend-1", MachineID: "machine-1", Kind: "ollama", Model: "qwen"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateOffer(ctx, Offer{ID: "offer-1", BackendID: "backend-1", Version: 3, InputPerMillion: 1, OutputPerMillion: 1}); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertRoutingState(ctx, RoutingState{MachineID: "machine-1", OfferID: "offer-1", Model: "qwen", BackendKind: "ollama", Capabilities: []string{"text", "stream"}, PriceVersion: 3, ConfirmedBond: true, Healthy: true, Capacity: 1, MaximumCost: 60, InputPerMillion: 1, OutputPerMillion: 1, LatencyMilliseconds: 20, SuccessBasisPoints: 9900, Reputation: 80}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		offerID, requestID  string
		inputRate, maxInput uint64
	}{
		{offerID: "offer-zero", requestID: "request-zero", inputRate: 0, maxInput: 4},
		{offerID: "offer-overflow", requestID: "request-overflow", inputRate: ^uint64(0), maxInput: 4_000_000},
	} {
		if err := s.CreateOffer(ctx, Offer{ID: tc.offerID, BackendID: "backend-1", Version: 3, InputPerMillion: tc.inputRate}); err != nil {
			t.Fatal(err)
		}
		if err := s.UpsertRoutingState(ctx, RoutingState{MachineID: "machine-1", OfferID: tc.offerID, Model: "qwen", BackendKind: "ollama", Capabilities: []string{"text", "stream"}, PriceVersion: 3, ConfirmedBond: true, Healthy: true, Capacity: 1, MaximumCost: 1, InputPerMillion: tc.inputRate}); err != nil {
			t.Fatal(err)
		}
		err := s.ReserveInference(ctx, InferenceReservation{RequestID: tc.requestID, SessionID: "session-1", AccountID: "customer", MachineID: "machine-1", OfferID: tc.offerID, PriceVersion: 3, MaximumSpend: ^uint64(0), MaximumInputTokens: tc.maxInput, MaximumOutputTokens: 1, MaximumComputeMilliseconds: 1})
		if !errors.Is(err, ErrIneligibleRoute) {
			t.Fatalf("offer %s expected ineligible route, got %v", tc.offerID, err)
		}
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE sessions SET confirmed_balance_wei=3 WHERE id='session-1'; UPDATE provider_routing_state SET capacity=2 WHERE machine_id='machine-1' AND offer_id='offer-1'`); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	type reserveResult struct {
		requestID string
		err       error
	}
	results := make(chan reserveResult, 2)
	var workers sync.WaitGroup
	for _, requestID := range []string{"request-concurrent-a", "request-concurrent-b"} {
		workers.Add(1)
		go func(id string) {
			defer workers.Done()
			<-start
			err := s.ReserveInference(ctx, InferenceReservation{RequestID: id, SessionID: "session-1", AccountID: "customer", MachineID: "machine-1", OfferID: "offer-1", PriceVersion: 3, MaximumSpend: 60, MaximumInputTokens: 4, MaximumOutputTokens: 1, MaximumComputeMilliseconds: 12})
			results <- reserveResult{requestID: id, err: err}
		}(requestID)
	}
	close(start)
	workers.Wait()
	close(results)
	var succeeded string
	insufficient := 0
	for result := range results {
		if result.err == nil {
			succeeded = result.requestID
		} else if errors.Is(result.err, ErrInsufficientBalance) {
			insufficient++
		} else {
			t.Fatalf("concurrent reserve %s: %v", result.requestID, result.err)
		}
	}
	if succeeded == "" || insufficient != 1 {
		t.Fatalf("concurrent admission succeeded=%q insufficient=%d", succeeded, insufficient)
	}
	if err := s.AbortInference(ctx, succeeded, "cancelled"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE sessions SET confirmed_balance_wei=70000000000000000000 WHERE id='session-1'; UPDATE provider_routing_state SET capacity=1 WHERE machine_id='machine-1' AND offer_id='offer-1'`); err != nil {
		t.Fatal(err)
	}
	candidates, err := s.RoutingCandidates(ctx, "qwen")
	if err != nil || len(candidates) != 3 {
		t.Fatalf("candidates=%+v err=%v", candidates, err)
	}
	reservation := InferenceReservation{RequestID: "request-1", SessionID: "session-1", AccountID: "customer", MachineID: "machine-1", OfferID: "offer-1", PriceVersion: 3, MaximumSpend: 60, MaximumInputTokens: 4, MaximumOutputTokens: 1, MaximumComputeMilliseconds: 12}
	if err := s.ReserveInference(ctx, reservation); err != nil {
		t.Fatal(err)
	}
	var held, requestMaximum string
	if err := s.db.QueryRowContext(ctx, `SELECT r.amount::text,q.maximum_spend::text FROM inference_reservations r JOIN requests q ON q.id=r.request_id WHERE r.request_id='request-1'`).Scan(&held, &requestMaximum); err != nil {
		t.Fatal(err)
	}
	if held != "2" || requestMaximum != "2" {
		t.Fatalf("held=%s maximum=%s want computed worst-case 2", held, requestMaximum)
	}
	if err := s.UpdateProviderCapacity(ctx, "machine-1", v1.Capacity{Available: 1, Offers: []v1.OfferCapacity{{OfferID: "offer-1", Model: "qwen", PriceVersion: 3}}}); err != nil {
		t.Fatal(err)
	}
	candidates, err = s.RoutingCandidates(ctx, "qwen")
	var offerCapacity uint32
	for _, candidate := range candidates {
		if candidate.OfferID == "offer-1" {
			offerCapacity = candidate.Capacity
		}
	}
	if err != nil || offerCapacity != 0 {
		t.Fatalf("heartbeat reopened reserved capacity: %+v err=%v", candidates, err)
	}
	if err := s.TransitionRequest(ctx, "request-1", "accepted"); err != nil {
		t.Fatal(err)
	}
	if err := s.TransitionRequest(ctx, "request-1", "streaming"); err != nil {
		t.Fatal(err)
	}
	completed := time.Now().UTC().Truncate(time.Microsecond)
	if err := s.CompleteInference(ctx, ReceiptProposal{RequestID: "request-1", SessionID: "session-1", MachineID: "machine-1", OfferID: "offer-1", Model: "qwen", PriceVersion: 3, InputTokens: 4, OutputTokens: 2, ComputeMilliseconds: 12, InputHash: [32]byte{1}, OutputHash: [32]byte{2}, CompletedAt: completed}); !errors.Is(err, ErrUsageLimitExceeded) {
		t.Fatalf("expected untrusted provider usage rejection, got %v", err)
	}
	if err := s.CompleteInference(ctx, ReceiptProposal{RequestID: "request-1", SessionID: "session-1", MachineID: "machine-1", OfferID: "offer-1", Model: "qwen", PriceVersion: 3, InputTokens: 4, OutputTokens: 1, ComputeMilliseconds: 12, InputHash: [32]byte{1}, OutputHash: [32]byte{2}, CompletedAt: completed}); err != nil {
		t.Fatal(err)
	}
	var state string
	var proposals, activeReservations int
	if err := s.db.QueryRowContext(ctx, "SELECT state FROM requests WHERE id = 'request-1'").Scan(&state); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM receipt_proposals WHERE request_id = 'request-1'").Scan(&proposals); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM inference_reservations WHERE request_id = 'request-1' AND released_at IS NULL").Scan(&activeReservations); err != nil {
		t.Fatal(err)
	}
	if state != "completed" || proposals != 1 || activeReservations != 1 {
		t.Fatalf("state=%s proposals=%d active=%d", state, proposals, activeReservations)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE sessions SET confirmed_balance_wei=3 WHERE id='session-1'`); err != nil {
		t.Fatal(err)
	}
	reservation.RequestID = "request-2"
	reservation.MaximumSpend = 50
	if err := s.ReserveInference(ctx, reservation); !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("expected pending-completion hold to prevent overspend, got %v", err)
	}
	if err := s.AbortInference(ctx, "request-1", "failed"); err != nil {
		t.Fatal(err)
	}
	var capacity uint32
	if err := s.db.QueryRowContext(ctx, `SELECT capacity FROM provider_routing_state WHERE machine_id='machine-1' AND offer_id='offer-1'`).Scan(&capacity); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM inference_reservations WHERE request_id='request-1' AND released_at IS NULL`).Scan(&activeReservations); err != nil {
		t.Fatal(err)
	}
	if capacity != 1 || activeReservations != 0 {
		t.Fatalf("post-completion abort capacity=%d active=%d", capacity, activeReservations)
	}
	if err := s.UpdateProviderCapacity(ctx, "machine-1", v1.Capacity{}); err != nil {
		t.Fatal(err)
	}
	candidates, err = s.RoutingCandidates(ctx, "qwen")
	allOffline := len(candidates) == 3
	for _, candidate := range candidates {
		allOffline = allOffline && candidate.Capacity == 0
	}
	if err != nil || !allOffline {
		t.Fatalf("offline capacity was not persisted: %+v err=%v", candidates, err)
	}
}

func TestCapacityReconcilesOnlyIndexedBondedMonadOffer(t *testing.T) {
	databaseURL := os.Getenv("MYFERENCE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("MYFERENCE_TEST_DATABASE_URL is required")
	}
	ctx := context.Background()
	s, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	for _, name := range []string{"000001_control_plane.sql", "000002_inference.sql", "000003_chain_index.sql", "000007_provider_operations.sql", "000008_machine_signers.sql", "000009_receipt_coordination.sql", "000014_reservation_finality.sql", "000015_runtime_model_evidence.sql"} {
		if err := s.ApplyMigration(ctx, filepath.Join("..", "..", "..", "migrations", name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.db.ExecContext(ctx, "TRUNCATE receipt_proposals,inference_reservations,provider_routing_state,outbox,requests,sessions,offers,backends,machines,accounts,chain_offers,chain_provider_signers,chain_accounts CASCADE"); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateAccount(ctx, Account{ID: "reconcile-account", WalletAddress: "0x1111111111111111111111111111111111111111"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateMachine(ctx, Machine{ID: "reconcile-machine", AccountID: "reconcile-account", Name: "unused-mac"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE machines SET signer_address='0x3333333333333333333333333333333333333333' WHERE id='reconcile-machine'`); err != nil {
		t.Fatal(err)
	}
	contract := "0x4444444444444444444444444444444444444444"
	offerHash := crypto.Keccak256Hash([]byte("local-qwen")).Hex()
	modelHash := crypto.Keccak256Hash([]byte("qwen")).Hex()
	capabilityHash := crypto.Keccak256Hash([]byte("stream,text")).Hex()
	if _, err := s.db.ExecContext(ctx, `INSERT INTO chain_accounts(chain_id,contract_address,address,provider_bond) VALUES (10143,$1,'0x1111111111111111111111111111111111111111',100)`, contract); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO chain_offers(chain_id,contract_address,provider,offer_id,version,model_hash,capability_hash,input_per_million,output_per_million,compute_per_second) VALUES (10143,$1,'0x1111111111111111111111111111111111111111',$2,1,$3,$4,10,20,30)`, contract, offerHash, modelHash, capabilityHash); err != nil {
		t.Fatal(err)
	}
	capacity := v1.Capacity{Available: 1, Offers: []v1.OfferCapacity{{OfferID: "local-qwen", Model: "qwen", PriceVersion: 1, BackendKind: "ollama", OfferHash: offerHash, ModelHash: modelHash, CapabilityHash: capabilityHash, Capabilities: []string{"stream", "text"}, EvidenceKind: "ollama_digest", EvidenceDigest: "sha256:first", MeteringMode: "tokens_and_compute"}}}
	if err := s.ReconcileProviderCapacity(ctx, "reconcile-machine", capacity, 10143, contract); !errors.Is(err, ErrIneligibleRoute) {
		t.Fatalf("expected unauthorized signer rejection, got %v", err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO chain_provider_signers(chain_id,contract_address,provider,signer,allowed) VALUES (10143,$1,'0x1111111111111111111111111111111111111111','0x3333333333333333333333333333333333333333',true)`, contract); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM chain_offers WHERE chain_id=10143 AND contract_address=$1`, contract); err != nil {
		t.Fatal(err)
	}
	if err := s.ReconcileProviderCapacity(ctx, "reconcile-machine", capacity, 10143, contract); err != nil {
		t.Fatalf("pre-activation discovery failed: %v", err)
	}
	var discovered int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM backends WHERE machine_id='reconcile-machine' AND model='qwen'`).Scan(&discovered); err != nil || discovered != 1 {
		t.Fatalf("discovered backends=%d err=%v", discovered, err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO chain_offers(chain_id,contract_address,provider,offer_id,version,model_hash,capability_hash,input_per_million,output_per_million,compute_per_second) VALUES (10143,$1,'0x1111111111111111111111111111111111111111',$2,1,$3,$4,10,20,30)`, contract, offerHash, modelHash, capabilityHash); err != nil {
		t.Fatal(err)
	}
	tampered := capacity
	tampered.Offers = append([]v1.OfferCapacity(nil), capacity.Offers...)
	tampered.Offers[0].OfferHash = crypto.Keccak256Hash([]byte("different-offer")).Hex()
	if err := s.ReconcileProviderCapacity(ctx, "reconcile-machine", tampered, 10143, contract); !errors.Is(err, ErrIneligibleRoute) {
		t.Fatalf("expected label/hash mismatch rejection, got %v", err)
	}
	if err := s.ReconcileProviderCapacity(ctx, "reconcile-machine", capacity, 10143, contract); err != nil {
		t.Fatal(err)
	}
	candidates, err := s.RoutingCandidates(ctx, "qwen")
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || !candidates[0].ConfirmedBond || candidates[0].MaximumCost != 60 || candidates[0].Capacity != 1 {
		t.Fatalf("unexpected reconciled route: %+v", candidates)
	}
	drifted := capacity
	drifted.Offers = append([]v1.OfferCapacity(nil), capacity.Offers...)
	drifted.Offers[0].EvidenceDigest = "sha256:changed"
	if err := s.ReconcileProviderCapacity(ctx, "reconcile-machine", drifted, 10143, contract); err != nil {
		t.Fatal(err)
	}
	candidates, err = s.RoutingCandidates(ctx, "qwen")
	if err != nil || len(candidates) != 1 || candidates[0].Capacity != 0 {
		t.Fatalf("digest drift was not quarantined: candidates=%+v err=%v", candidates, err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO chain_offers(chain_id,contract_address,provider,offer_id,version,model_hash,capability_hash,input_per_million,output_per_million,compute_per_second) VALUES (10143,$1,'0x1111111111111111111111111111111111111111',$2,2,$3,$4,10,20,30)`, contract, offerHash, modelHash, capabilityHash); err != nil {
		t.Fatal(err)
	}
	drifted.Offers[0].PriceVersion = 2
	if err := s.ReconcileProviderCapacity(ctx, "reconcile-machine", drifted, 10143, contract); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateAccount(ctx, Account{ID: "second-provider", WalletAddress: "0x7777777777777777777777777777777777777777"}); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateMachine(ctx, Machine{ID: "second-machine", AccountID: "second-provider", Name: "second-provider"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE machines SET signer_address='0x8888888888888888888888888888888888888888' WHERE id='second-machine'`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO chain_accounts(chain_id,contract_address,address,provider_bond) VALUES (10143,$1,'0x7777777777777777777777777777777777777777',100)`, contract); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO chain_provider_signers(chain_id,contract_address,provider,signer,allowed) VALUES (10143,$1,'0x7777777777777777777777777777777777777777','0x8888888888888888888888888888888888888888',true)`, contract); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO chain_offers(chain_id,contract_address,provider,offer_id,version,model_hash,capability_hash,input_per_million,output_per_million,compute_per_second) VALUES (10143,$1,'0x7777777777777777777777777777777777777777',$2,1,$3,$4,100,200,300)`, contract, offerHash, modelHash, capabilityHash); err != nil {
		t.Fatal(err)
	}
	if err := s.ReconcileProviderCapacity(ctx, "second-machine", capacity, 10143, contract); err != nil {
		t.Fatal(err)
	}
	candidates, err = s.RoutingCandidates(ctx, "qwen")
	if err != nil || len(candidates) != 2 {
		t.Fatalf("colliding offer candidates=%+v err=%v", candidates, err)
	}
	for _, candidate := range candidates {
		if candidate.MachineID == "reconcile-machine" && (candidate.InputPerMillion != 10 || candidate.OutputPerMillion != 20 || candidate.ComputePerSecond != 30) {
			t.Fatalf("first provider rates were overwritten: %+v", candidate)
		}
		if candidate.MachineID == "second-machine" && (candidate.InputPerMillion != 100 || candidate.OutputPerMillion != 200 || candidate.ComputePerSecond != 300) {
			t.Fatalf("second provider inherited colliding rates: %+v", candidate)
		}
	}
	detail, err := s.MarketplaceModel(ctx, "qwen", time.Hour)
	if err != nil || len(detail.Offers) != 2 {
		t.Fatalf("colliding marketplace offers=%+v err=%v", detail.Offers, err)
	}
	for _, item := range detail.Offers {
		if item.MachineID == "reconcile-machine" && (item.InputPerMillion != "10" || item.OutputPerMillion != "20" || item.ComputePerSecond != "30") {
			t.Fatalf("marketplace mixed first provider rates: %+v", item)
		}
		if item.MachineID == "second-machine" && (item.InputPerMillion != "100" || item.OutputPerMillion != "200" || item.ComputePerSecond != "300") {
			t.Fatalf("marketplace mixed second provider rates: %+v", item)
		}
	}
	operations, err := s.AccountOperations(ctx, "second-provider", 10143, contract, "https://testnet.monadexplorer.com", 1)
	if err != nil || len(operations.Machines) != 1 || len(operations.Machines[0].Backends) != 1 || !operations.Machines[0].Backends[0].Healthy || operations.Machines[0].Backends[0].Capacity != 1 {
		t.Fatalf("second provider operations=%+v err=%v", operations.Machines, err)
	}
	requestID := "0x" + strings.Repeat("12", 32)
	sessionID := "0x" + strings.Repeat("34", 32)
	if err := s.CreateAccount(ctx, Account{ID: "receipt-customer", WalletAddress: "0x5555555555555555555555555555555555555555"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO sessions(id,account_id,state,confirmed_balance_wei) VALUES ($1,'receipt-customer','open',60)`, sessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO requests(id,session_id,state,machine_id,offer_id,price_version,maximum_spend,maximum_input_tokens,maximum_output_tokens,maximum_compute_milliseconds,offer_hash,model_hash,capability_hash) SELECT $1,$2,'streaming','reconcile-machine','local-qwen',1,60,1,1,1,offer_hash,model_hash,capability_hash FROM provider_routing_state WHERE machine_id='reconcile-machine' AND offer_id='local-qwen'`, requestID, sessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO inference_reservations(request_id,session_id,amount) VALUES($1,$2,60)`, requestID, sessionID); err != nil {
		t.Fatal(err)
	}
	if err := s.ReconcileProviderCapacity(ctx, "reconcile-machine", capacity, 10143, contract); err != nil {
		t.Fatal(err)
	}
	var reconciledCapacity uint32
	if err := s.db.QueryRowContext(ctx, `SELECT capacity FROM provider_routing_state WHERE machine_id='reconcile-machine' AND offer_id='local-qwen'`).Scan(&reconciledCapacity); err != nil || reconciledCapacity != 0 {
		t.Fatalf("production heartbeat reopened executing slot: capacity=%d err=%v", reconciledCapacity, err)
	}
	if err := s.CompleteInference(ctx, ReceiptProposal{RequestID: requestID, SessionID: sessionID, MachineID: "reconcile-machine", OfferID: "local-qwen", Model: "qwen", PriceVersion: 1, InputTokens: 1, OutputTokens: 1, ComputeMilliseconds: 1, InputHash: [32]byte{1}, OutputHash: [32]byte{2}, CompletedAt: time.Unix(100, 0)}); err != nil {
		t.Fatal(err)
	}
	if err := s.ReconcileProviderCapacity(ctx, "reconcile-machine", capacity, 10143, contract); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT capacity FROM provider_routing_state WHERE machine_id='reconcile-machine' AND offer_id='local-qwen'`).Scan(&reconciledCapacity); err != nil || reconciledCapacity != 1 {
		t.Fatalf("financial hold kept completed execution slot closed: capacity=%d err=%v", reconciledCapacity, err)
	}
	receipt, machineID, signer, err := s.PrepareReceipt(ctx, requestID, ReceiptDomain{ChainID: 10143, ContractAddress: contract, SettlementSigner: "0x6666666666666666666666666666666666666666", FeeBasisPoints: 500, FeeVersion: 1})
	if err != nil {
		t.Fatal(err)
	}
	if machineID != "reconcile-machine" || signer != "0x3333333333333333333333333333333333333333" || receipt.TotalCharge != 3 || receipt.MaximumCharge != 60 || receipt.Nonce != 1 || receipt.OfferID != v1.Hash(crypto.Keccak256Hash([]byte("local-qwen"))) {
		t.Fatalf("prepared receipt=%+v machine=%q signer=%q", receipt, machineID, signer)
	}
	var exactHold string
	var holdReleased bool
	if err := s.db.QueryRowContext(ctx, `SELECT amount::text,released_at IS NOT NULL FROM inference_reservations WHERE request_id=$1`, requestID).Scan(&exactHold, &holdReleased); err != nil {
		t.Fatal(err)
	}
	if exactHold != "3" || holdReleased {
		t.Fatalf("exact hold=%s released=%v", exactHold, holdReleased)
	}
}
