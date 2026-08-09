package main

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/kunalshah017/myference/server/internal/api"
	"github.com/kunalshah017/myference/server/internal/store"
)

func TestProviderOfferQueriesHashImmutableIdentityAndVerifyNewVersion(t *testing.T) {
	offer := api.ProviderActionOffer{OfferID: "local-qwen", Model: "qwen", Kind: "ollama", Capabilities: []string{"stream", "text"}, MeteringMode: "tokens_and_compute", InputPerMillionWei: "10", OutputPerMillionWei: "20", ComputePerSecondWei: "30"}
	queries := providerOfferQueries([]api.ProviderActionOffer{offer})
	if len(queries) != 1 || queries[0].OfferHash != crypto.Keccak256Hash([]byte("local-qwen")).Hex() || queries[0].CapabilityHash != crypto.Keccak256Hash([]byte("stream,text")).Hex() {
		t.Fatalf("queries=%+v", queries)
	}
	action := api.ProviderAction{Kind: api.ActionPublishOffer, Offers: []api.ProviderActionOffer{offer}, BaselineState: api.ProviderActionBaseline{Versions: map[string]uint64{"local-qwen": 2}}}
	versions, confirmed := providerActionConfirmed(action, store.ProviderActionState{MatchingVersions: map[string]uint64{"local-qwen": 3}})
	if !confirmed || versions["local-qwen"] != 3 {
		t.Fatalf("versions=%v confirmed=%v", versions, confirmed)
	}
}

func TestProviderActionConfirmationUsesCollateralBaselines(t *testing.T) {
	deposit := api.ProviderAction{Kind: api.ActionDepositCollateral, BaselineState: api.ProviderActionBaseline{BondWei: "100"}, AmountWei: "25"}
	if _, ok := providerActionConfirmed(deposit, store.ProviderActionState{BondWei: "124"}); ok {
		t.Fatal("confirmed insufficient deposit")
	}
	if _, ok := providerActionConfirmed(deposit, store.ProviderActionState{BondWei: "125"}); !ok {
		t.Fatal("did not confirm deposit")
	}
	request := api.ProviderAction{Kind: api.ActionRequestCollateralExit, BaselineState: api.ProviderActionBaseline{ExitAvailableAt: 0}}
	if _, ok := providerActionConfirmed(request, store.ProviderActionState{ExitAvailableAt: 200}); !ok {
		t.Fatal("did not confirm exit request")
	}
	finalize := api.ProviderAction{Kind: api.ActionFinalizeCollateralExit, BaselineState: api.ProviderActionBaseline{BondWei: "100", ExitAvailableAt: 200}}
	if _, ok := providerActionConfirmed(finalize, store.ProviderActionState{BondWei: "0", ExitAvailableAt: 0}); !ok {
		t.Fatal("did not confirm exit finalization")
	}
}
