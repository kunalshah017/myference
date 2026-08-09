package providerops

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kunalshah017/myference/cli/internal/account"
	"github.com/kunalshah017/myference/cli/internal/config"
)

func TestParseMONIsExactAndRejectsAmbiguousValues(t *testing.T) {
	valid := map[string]string{"1": "1000000000000000000", "0.000000000000000001": "1", "5.25": "5250000000000000000"}
	for input, want := range valid {
		got, err := ParseMON(input)
		if err != nil || got != want {
			t.Fatalf("ParseMON(%q)=%q err=%v", input, got, err)
		}
	}
	for _, input := range []string{"", "-1", "1e3", ".5", "1.0000000000000000001", "115792089237316195423570985008687907853269984665640564039458"} {
		if _, err := ParseMON(input); err == nil {
			t.Fatalf("ParseMON(%q) succeeded", input)
		}
	}
}

func TestPublishCreatesExactActionOpensApprovalAndAppliesConfirmedVersion(t *testing.T) {
	api := &providerAPIStub{actions: []account.ProviderAction{
		{ID: "action-1", Status: account.ActionPendingWallet, ExpiresAt: time.Now().Add(time.Minute)},
		{ID: "action-1", Status: account.ActionConfirmed, Versions: map[string]uint64{"local-qwen": 7}},
	}}
	cfg := config.Config{MachineID: "machine-1", Backends: []config.Backend{{Name: "local-qwen", Kind: "ollama", Model: "qwen2.5", Enabled: true}}}
	opened := ""
	saved := config.Config{}
	service := Service{API: api, Token: "machine-token", WebURL: "https://myference.test/app", LoadConfig: func() (config.Config, error) { return cfg, nil }, SaveConfig: func(value config.Config) error { saved = value; return nil }, OpenURL: func(value string) error { opened = value; return nil }, Wait: noWait}
	err := service.Publish(context.Background(), cfg.Backends[0], Rates{InputPerMillionMON: "0.1", OutputPerMillionMON: "0.2", ComputePerSecondMON: "0.0001"})
	if err != nil {
		t.Fatal(err)
	}
	if opened != "https://myference.test/app/provider/approve?action=action-1" {
		t.Fatalf("opened=%q", opened)
	}
	if saved.Backends[0].PriceVersion != 7 {
		t.Fatalf("saved=%+v", saved.Backends)
	}
	input := api.created
	if input.Kind != account.ActionPublishOffer || input.Offers[0].InputPerMillionWei != "100000000000000000" || input.Offers[0].OutputPerMillionWei != "200000000000000000" || input.Offers[0].ComputePerSecondWei != "100000000000000" {
		t.Fatalf("input=%+v", input)
	}
}

func TestDepositParsesAmountAndBrowserFailureReturnsCopyableURL(t *testing.T) {
	api := &providerAPIStub{actions: []account.ProviderAction{{ID: "deposit", Status: account.ActionPendingWallet, ExpiresAt: time.Now().Add(time.Minute)}}}
	service := Service{API: api, Token: "token", WebURL: "https://myference.test", OpenURL: func(string) error { return errors.New("no browser") }}
	err := service.Deposit(context.Background(), "5.25")
	var approval *ApprovalRequiredError
	if !errors.As(err, &approval) || approval.URL != "https://myference.test/provider/approve?action=deposit" {
		t.Fatalf("error=%v", err)
	}
	if api.created.AmountWei != "5250000000000000000" {
		t.Fatalf("input=%+v", api.created)
	}
}

type providerAPIStub struct {
	created  account.ProviderActionInput
	actions  []account.ProviderAction
	versions map[string]uint64
}

func (s *providerAPIStub) ProviderAccount(context.Context, string) (account.ProviderAccount, error) {
	return account.ProviderAccount{}, nil
}
func (s *providerAPIStub) CreateProviderAction(_ context.Context, _ string, input account.ProviderActionInput) (account.ProviderAction, error) {
	s.created = input
	return s.actions[0], nil
}
func (s *providerAPIStub) ProviderAction(context.Context, string, string) (account.ProviderAction, error) {
	if len(s.actions) < 2 {
		return s.actions[0], nil
	}
	action := s.actions[1]
	s.actions = s.actions[1:]
	return action, nil
}
func (s *providerAPIStub) MachineOfferVersions(context.Context, string, string) (map[string]uint64, error) {
	return s.versions, nil
}
func noWait(context.Context, time.Duration) error { return nil }
