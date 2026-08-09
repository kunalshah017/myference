package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestProviderAccountProjectionIsTenantIsolatedAndReturnsCompatibleVersions(t *testing.T) {
	databaseURL := os.Getenv("MYFERENCE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("MYFERENCE_TEST_DATABASE_URL is required")
	}
	ctx := context.Background()
	repository, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repository.Close() })
	for _, name := range []string{"000001_control_plane.sql", "000002_inference.sql", "000003_chain_index.sql", "000015_runtime_model_evidence.sql"} {
		if err := repository.ApplyMigration(ctx, filepath.Join("..", "..", "..", "migrations", name)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := repository.db.ExecContext(ctx, "TRUNCATE provider_routing_state,offers,backends,machines,accounts,chain_offers,chain_accounts CASCADE"); err != nil {
		t.Fatal(err)
	}
	contract := "0x4444444444444444444444444444444444444444"
	offerHash := crypto.Keccak256Hash([]byte("local-qwen")).Hex()
	modelHash := crypto.Keccak256Hash([]byte("qwen")).Hex()
	capabilityHash := crypto.Keccak256Hash([]byte("stream,text")).Hex()
	for _, fixture := range []struct {
		account, machine, backend, wallet string
		rate, version                     int
	}{
		{"account-one", "machine-one", "backend-one", "0x1111111111111111111111111111111111111111", 10, 2},
		{"account-two", "machine-two", "backend-two", "0x2222222222222222222222222222222222222222", 99, 4},
	} {
		if err := repository.CreateAccount(ctx, Account{ID: fixture.account, WalletAddress: fixture.wallet}); err != nil {
			t.Fatal(err)
		}
		if err := repository.CreateMachine(ctx, Machine{ID: fixture.machine, AccountID: fixture.account, Name: fixture.machine}); err != nil {
			t.Fatal(err)
		}
		if err := repository.CreateBackend(ctx, Backend{ID: fixture.backend, MachineID: fixture.machine, Kind: "ollama", Model: "qwen"}); err != nil {
			t.Fatal(err)
		}
		if err := repository.CreateOffer(ctx, Offer{ID: "local-qwen", BackendID: fixture.backend, Version: int64(fixture.version), InputPerMillion: uint64(fixture.rate), OutputPerMillion: uint64(fixture.rate), ComputePerSecond: uint64(fixture.rate)}); err != nil {
			t.Fatal(err)
		}
		if err := repository.UpsertRoutingState(ctx, RoutingState{MachineID: fixture.machine, OfferID: "local-qwen", Model: "qwen", BackendKind: "ollama", Capabilities: []string{"stream", "text"}, MeteringMode: "tokens_and_compute", EvidenceKind: "ollama_digest", EvidenceDigest: "sha256:same", PriceVersion: uint64(fixture.version), ConfirmedBond: true, Healthy: true, Capacity: 1, MaximumCost: 1}); err != nil {
			t.Fatal(err)
		}
		if _, err := repository.db.ExecContext(ctx, `INSERT INTO chain_accounts(chain_id,contract_address,address,provider_bond,claimable) VALUES(10143,$1,$2,100,7)`, contract, fixture.wallet); err != nil {
			t.Fatal(err)
		}
		if _, err := repository.db.ExecContext(ctx, `INSERT INTO chain_offers(chain_id,contract_address,provider,offer_id,version,model_hash,capability_hash,input_per_million,output_per_million,compute_per_second) VALUES(10143,$1,$2,$3,$4,$5,$6,$7,$7,$7)`, contract, fixture.wallet, offerHash, fixture.version, modelHash, capabilityHash, fixture.rate); err != nil {
			t.Fatal(err)
		}
	}

	account, err := repository.ProviderAccount(ctx, "account-one", ProviderAccountConfig{ChainID: 10143, ContractAddress: contract, MinimumBondWei: "5"})
	if err != nil || account.WalletAddress != "0x1111111111111111111111111111111111111111" || len(account.Offers) != 1 || account.Offers[0].OfferID != "local-qwen" || account.Offers[0].Version != 2 || account.Offers[0].InputPerMillionWei != "10" {
		t.Fatalf("account=%+v err=%v", account, err)
	}
	versions, err := repository.MachineOfferVersions(ctx, "machine-one", "account-one", 10143, contract)
	if err != nil || len(versions) != 1 || versions["local-qwen"] != 2 {
		t.Fatalf("versions=%v err=%v", versions, err)
	}
	if other, err := repository.MachineOfferVersions(ctx, "machine-two", "account-one", 10143, contract); err != nil || len(other) != 0 {
		t.Fatalf("cross-account versions=%v err=%v", other, err)
	}
	evidence, err := repository.ProviderActionState(ctx, "account-one", ProviderAccountConfig{ChainID: 10143, ContractAddress: contract}, []ProviderOfferQuery{{OfferID: "local-qwen", OfferHash: offerHash, ModelHash: modelHash, CapabilityHash: capabilityHash, InputPerMillionWei: "10", OutputPerMillionWei: "10", ComputePerSecondWei: "10"}})
	if err != nil || evidence.BondWei != "100" || evidence.LatestVersions["local-qwen"] != 2 || evidence.MatchingVersions["local-qwen"] != 2 {
		t.Fatalf("evidence=%+v err=%v", evidence, err)
	}
}
