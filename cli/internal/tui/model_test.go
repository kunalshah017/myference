package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kunalshah017/myference/cli/internal/account"
	"github.com/kunalshah017/myference/cli/internal/config"
	"github.com/kunalshah017/myference/cli/internal/host"
	"github.com/kunalshah017/myference/cli/internal/provider"
	"github.com/kunalshah017/myference/cli/internal/providerops"
)

func TestHomeExposesProviderOfferCollateralAndStatusWorkflows(t *testing.T) {
	model := NewModel(Dependencies{}, nil)
	view := model.ViewText()
	for _, item := range []string{"Providers", "Offers & Pricing", "Collateral", "Live Status", "Quit"} {
		if !strings.Contains(view, item) {
			t.Fatalf("home=%q missing=%q", view, item)
		}
	}
	model, _ = model.HandleKey("down")
	model, _ = model.HandleKey("enter")
	if model.Screen() != ScreenOffers {
		t.Fatalf("screen=%v", model.Screen())
	}
}

func TestOffersShowPublicationStateAndOpenMeteringAwarePricing(t *testing.T) {
	backends := []config.Backend{{Name: "local-qwen", Kind: "ollama", Model: "qwen", PriceVersion: 0}, {Name: "agent", Kind: "codex", Model: "gpt", Image: "agent", PriceVersion: 3}}
	model := NewModel(Dependencies{Backends: backends}, nil)
	model.screen = ScreenOffers
	view := model.ViewText()
	if !strings.Contains(view, "Not published") || !strings.Contains(view, "v3") {
		t.Fatalf("view=%q", view)
	}
	model, _ = model.HandleKey("enter")
	if model.Screen() != ScreenPricing || !strings.Contains(model.ViewText(), "Input / million tokens") {
		t.Fatalf("screen=%v view=%q", model.Screen(), model.ViewText())
	}
	model.screen, model.cursor = ScreenOffers, 1
	model, _ = model.HandleKey("e")
	if !strings.Contains(model.ViewText(), "Compute / second") || strings.Contains(model.ViewText(), "Input / million tokens") {
		t.Fatalf("compute-only view=%q", model.ViewText())
	}
}

func TestOffersRenderEveryWalletOffer(t *testing.T) {
	backend := config.Backend{Name: "ollama-qwen", OfferID: "local-qwen", Kind: "ollama", Model: "qwen2.5:0.5b", PriceVersion: 1, Enabled: true}
	model := NewModel(Dependencies{Backends: []config.Backend{backend}}, nil)
	model.screen = ScreenOffers
	model.applyAccount(accountMsg{account: account.ProviderAccount{Offers: []account.EditableOffer{
		{OfferID: "local-qwen", Model: backend.Model, BackendKind: backend.Kind, Capabilities: []string{"stream", "text"}, MeteringMode: "tokens_and_compute", Version: 1},
		{OfferID: "gpt-5.6-terra", Model: "gpt-5.6-terra", BackendKind: "openai", Capabilities: []string{"stream", "text"}, MeteringMode: "tokens_and_compute", Version: 1},
	}}})
	view := model.ViewText()
	for _, expected := range []string{"This machine", "Wallet offers", "local-qwen", "Attached here", "gpt-5.6-terra", "No matching local provider"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("view=%q missing=%q", view, expected)
		}
	}
}

func TestAttachedOfferSelectionDoesNotOpenPricing(t *testing.T) {
	backend := config.Backend{Name: "ollama-qwen", OfferID: "local-qwen", Kind: "ollama", Model: "qwen", PriceVersion: 1, Enabled: true}
	offer := account.EditableOffer{OfferID: "local-qwen", Model: backend.Model, BackendKind: backend.Kind, Capabilities: []string{"stream", "text"}, MeteringMode: "tokens_and_compute", Version: 1}
	model := NewModel(Dependencies{Backends: []config.Backend{backend}}, nil)
	model.screen = ScreenOffers
	model.applyAccount(accountMsg{account: account.ProviderAccount{Offers: []account.EditableOffer{offer}}})
	model, command := model.HandleKey("enter")
	if command != nil || model.Screen() != ScreenOffers || !strings.Contains(model.ViewText(), "Already attached") {
		t.Fatalf("screen=%v command=%v view=%q", model.Screen(), command, model.ViewText())
	}
	model, _ = model.HandleKey("down")
	model, command = model.HandleKey("enter")
	if command != nil || model.Screen() != ScreenOffers || !strings.Contains(model.ViewText(), "Already attached") {
		t.Fatalf("wallet screen=%v command=%v view=%q", model.Screen(), command, model.ViewText())
	}
}

func TestWalletOfferSelectionAttachesOrExplainsUnavailable(t *testing.T) {
	backend := config.Backend{Name: "ollama-qwen", Kind: "ollama", Model: "qwen", Enabled: true}
	qwen := account.EditableOffer{OfferID: "local-qwen", Model: backend.Model, BackendKind: backend.Kind, Capabilities: []string{"stream", "text"}, MeteringMode: "tokens_and_compute", Version: 1}
	gpt := account.EditableOffer{OfferID: "gpt", Model: "gpt", BackendKind: "openai", Capabilities: []string{"stream", "text"}, MeteringMode: "tokens_and_compute", Version: 1}
	attached := ""
	model := NewModel(Dependencies{Backends: []config.Backend{backend}, Attach: func(_ context.Context, backendName, offerID string) error {
		attached = backendName + ":" + offerID
		return nil
	}}, nil)
	model.screen = ScreenOffers
	model.applyAccount(accountMsg{account: account.ProviderAccount{Offers: []account.EditableOffer{qwen, gpt}}})
	model, _ = model.HandleKey("down")
	if model.cursor != 1 {
		t.Fatalf("cursor=%d want wallet row 1", model.cursor)
	}
	model, command := model.HandleKey("enter")
	if command == nil {
		t.Fatal("attachable wallet offer did not return an attachment command")
	}
	message := command().(providerOperationMsg)
	if message.err != nil || attached != "ollama-qwen:local-qwen" {
		t.Fatalf("attached=%q err=%v", attached, message.err)
	}
	model.busy = false
	model, _ = model.HandleKey("down")
	model, command = model.HandleKey("enter")
	if command != nil || !strings.Contains(model.ViewText(), "Configure a matching provider first") {
		t.Fatalf("command=%v view=%q", command, model.ViewText())
	}
}

func TestExplicitRepricingOpensPricing(t *testing.T) {
	backend := config.Backend{Name: "ollama-qwen", OfferID: "local-qwen", Kind: "ollama", Model: "qwen", PriceVersion: 1, Enabled: true}
	model := NewModel(Dependencies{Backends: []config.Backend{backend}}, nil)
	model.screen = ScreenOffers
	model, command := model.HandleKey("e")
	if command != nil || model.Screen() != ScreenPricing {
		t.Fatalf("screen=%v command=%v", model.Screen(), command)
	}
}

func TestOffersFetchAndAttachCompatibleWalletOffer(t *testing.T) {
	backend := config.Backend{Name: "ollama-qwen", Kind: "ollama", Model: "qwen2.5:0.5b", Enabled: true}
	offer := account.EditableOffer{OfferID: "local-qwen", Model: backend.Model, BackendKind: backend.Kind, Capabilities: []string{"stream", "text"}, MeteringMode: "tokens_and_compute", Version: 1}
	attached := ""
	model := NewModel(Dependencies{
		Backends: []config.Backend{backend},
		Account: func(context.Context) (account.ProviderAccount, error) {
			return account.ProviderAccount{Offers: []account.EditableOffer{offer}}, nil
		},
		Attach: func(_ context.Context, backendName, offerID string) error {
			attached = backendName + ":" + offerID
			return nil
		},
	}, nil)
	model, _ = model.HandleKey("down")
	updated, teaCommand := model.HandleKey("enter")
	model = updated
	if teaCommand == nil {
		t.Fatal("offers did not fetch provider account")
	}
	message := teaCommand().(accountMsg)
	model.applyAccount(message)
	if view := model.ViewText(); !strings.Contains(view, "Attach local-qwen v1") {
		t.Fatalf("view=%q", view)
	}
	model, teaCommand = model.HandleKey("enter")
	if teaCommand == nil {
		t.Fatal("compatible offer did not create attachment command")
	}
	result := teaCommand().(providerOperationMsg)
	if result.err != nil || result.status != "Offer attached." || attached != "ollama-qwen:local-qwen" {
		t.Fatalf("attached=%q status=%q err=%v", attached, result.status, result.err)
	}
}

func TestOffersRequireSelectionWhenSeveralWalletOffersAreCompatible(t *testing.T) {
	backend := config.Backend{Name: "ollama-qwen", Kind: "ollama", Model: "qwen", Enabled: true}
	offer := func(id string) account.EditableOffer {
		return account.EditableOffer{OfferID: id, Model: backend.Model, BackendKind: backend.Kind, Capabilities: []string{"stream", "text"}, MeteringMode: "tokens_and_compute", Version: 1}
	}
	attached := ""
	model := NewModel(Dependencies{Backends: []config.Backend{backend}, Attach: func(_ context.Context, _, offerID string) error { attached = offerID; return nil }}, nil)
	model.screen = ScreenOffers
	model.applyAccount(accountMsg{account: account.ProviderAccount{Offers: []account.EditableOffer{offer("first"), offer("second")}}})
	model, command := model.HandleKey("enter")
	if command != nil || model.Screen() != ScreenOfferAttach || !strings.Contains(model.ViewText(), "first") || !strings.Contains(model.ViewText(), "second") {
		t.Fatalf("screen=%v view=%q", model.Screen(), model.ViewText())
	}
	model, _ = model.HandleKey("down")
	_, command = model.HandleKey("enter")
	if command == nil {
		t.Fatal("selected offer did not create attachment command")
	}
	result := command().(providerOperationMsg)
	if result.err != nil || attached != "second" {
		t.Fatalf("attached=%q err=%v", attached, result.err)
	}
}

func TestFailedProviderOperationKeepsErrorWithoutRefreshingAccount(t *testing.T) {
	model := NewModel(Dependencies{Account: func(context.Context) (account.ProviderAccount, error) {
		return account.ProviderAccount{}, nil
	}}, nil)
	updated, command := model.Update(providerOperationMsg{err: errors.New("attachment failed")})
	result := updated.(Model)
	if command != nil || result.err == nil || !strings.Contains(result.err.Error(), "attachment failed") {
		t.Fatalf("command=%v err=%v", command, result.err)
	}
}

func TestCollateralRendersAccountAndRunsAvailableAction(t *testing.T) {
	requested := false
	model := NewModel(Dependencies{Account: func(context.Context) (account.ProviderAccount, error) {
		return account.ProviderAccount{ProviderBondWei: "5000000000000000000", ClaimableWei: "2"}, nil
	}, RequestExit: func(context.Context) error { requested = true; return nil }}, nil)
	model.screen = ScreenCollateral
	model.applyAccount(accountMsg{account: account.ProviderAccount{ProviderBondWei: "5000000000000000000", ClaimableWei: "2"}})
	if view := model.ViewText(); !strings.Contains(view, "5 MON") || !strings.Contains(view, "Request exit") {
		t.Fatalf("view=%q", view)
	}
	_, command := model.HandleKey("x")
	if command == nil {
		t.Fatal("request exit did not create command")
	}
	message := command().(providerOperationMsg)
	if message.err != nil || !requested {
		t.Fatalf("requested=%v err=%v", requested, message.err)
	}
}

func TestPricingPublishDelegatesToProviderOperations(t *testing.T) {
	called := false
	backend := config.Backend{Name: "qwen", Kind: "ollama", Model: "qwen"}
	model := NewModel(Dependencies{Backends: []config.Backend{backend}, Publish: func(_ context.Context, got config.Backend, rates providerops.Rates) error {
		called = got.Name == "qwen" && rates.InputPerMillionMON == "1"
		return nil
	}}, nil)
	model.openPricing(backend)
	model.priceInput.SetValue("1")
	model.priceOutput.SetValue("2")
	model.priceCompute.SetValue("3")
	model.priceStep = priceStepReview
	_, command := model.HandleKey("enter")
	if command == nil {
		t.Fatal("publish did not create command")
	}
	message := command().(providerOperationMsg)
	if message.err != nil || !called {
		t.Fatalf("called=%v err=%v", called, message.err)
	}
}

func TestHomeProvidersReviewNavigationAndMultiSelection(t *testing.T) {
	candidates := []host.Candidate{
		{ID: "ollama||llama", Kind: "ollama", Name: "Ollama", Model: "llama", State: host.StateReady},
		{ID: "codex||gpt", Kind: "codex", Name: "Codex", Model: "gpt", State: host.StateReady},
	}
	model := NewModel(Dependencies{}, candidates)
	model, _ = model.HandleKey("enter")
	if model.screen != ScreenProviders {
		t.Fatalf("screen=%v", model.screen)
	}
	model, _ = model.HandleKey("space")
	model, _ = model.HandleKey("down")
	model, _ = model.HandleKey("space")
	if !model.Selected(candidates[0].ID) || !model.Selected(candidates[1].ID) {
		t.Fatalf("selected=%v", model.selected)
	}
	model, _ = model.HandleKey("r")
	if model.screen != ScreenReview || !strings.Contains(model.ViewText(), "2 providers selected") {
		t.Fatalf("screen=%v view=%q", model.screen, model.ViewText())
	}
}

func TestAPISecretIsMaskedAndNeverRendered(t *testing.T) {
	model := NewModel(Dependencies{}, nil)
	model.openAPIForm(false)
	model.apiKey.SetValue("api-secret-value")
	view := model.ViewText()
	if strings.Contains(view, "api-secret-value") || !strings.Contains(view, "********") {
		t.Fatalf("view=%q", view)
	}
	model.clearAPIForm()
	if model.apiKey.Value() != "" {
		t.Fatal("secret remains after clearing form")
	}
}

func TestCatalogFailureOffersManualModelEntry(t *testing.T) {
	model := NewModel(Dependencies{}, nil)
	model.openAPIForm(true)
	model.applyCatalog(catalogMsg{baseURL: "https://provider.example", providerName: "Example", secret: "secret", err: host.ErrModelCatalogUnavailable})
	if model.screen != ScreenAPI || model.apiStep != apiStepModel || !strings.Contains(model.ViewText(), "Enter the model name manually") {
		t.Fatalf("screen=%v step=%v view=%q", model.screen, model.apiStep, model.ViewText())
	}
}

func TestRuntimeFailureMovesToStatusWithoutLosingSelections(t *testing.T) {
	candidate := host.Candidate{ID: "ollama||qwen", Kind: "ollama", Name: "Ollama", Model: "qwen", State: host.StateReady}
	model := NewModel(Dependencies{}, []host.Candidate{candidate})
	model.selected[candidate.ID] = true
	model.applyStarted(startedMsg{err: errors.New("relay unavailable")})
	if model.screen != ScreenStatus || model.running || !strings.Contains(model.ViewText(), "relay unavailable") || !model.Selected(candidate.ID) {
		t.Fatalf("model=%+v view=%q", model, model.ViewText())
	}
}

func TestQuitRequiresConfirmationWhileRunning(t *testing.T) {
	model := NewModel(Dependencies{Stop: func() {}}, nil)
	model.running = true
	model.screen = ScreenStatus
	model, command := model.HandleKey("q")
	if command != nil || !model.confirmQuit {
		t.Fatalf("confirm=%v command=%v", model.confirmQuit, command)
	}
	model, command = model.HandleKey("y")
	if command == nil {
		t.Fatal("confirmed quit did not return a command")
	}
}

func TestWindowSizeIsRemembered(t *testing.T) {
	model := NewModel(Dependencies{}, nil)
	model = model.Resize(120, 40)
	if model.width != 120 || model.height != 40 {
		t.Fatalf("size=%dx%d", model.width, model.height)
	}
}

func TestStartPassesAllSelectionsAndSecrets(t *testing.T) {
	candidate := host.Candidate{ID: "openai|https://api.openai.com|gpt", Kind: "openai", Name: "OpenAI", URL: "https://api.openai.com", Model: "gpt", State: host.StateReady}
	var got []host.Selection
	model := NewModel(Dependencies{Configure: func(_ context.Context, selection []host.Selection) error {
		got = selection
		return nil
	}, Start: func(context.Context) error { return nil }}, []host.Candidate{candidate})
	model.selected[candidate.ID] = true
	model.secrets[candidate.ID] = "secret"
	message := model.startCommand()()
	started, ok := message.(startedMsg)
	if !ok || started.err != nil || len(got) != 1 || got[0].Secret != "secret" {
		t.Fatalf("message=%T %+v selection=%+v", message, message, got)
	}
}

func TestStartConfiguresThenConnects(t *testing.T) {
	candidate := host.Candidate{ID: "ollama||qwen", Kind: "ollama", Name: "Ollama", Model: "qwen", State: host.StateReady}
	var calls []string
	model := NewModel(Dependencies{
		Configure: func(context.Context, []host.Selection) error { calls = append(calls, "configure"); return nil },
		Start:     func(context.Context) error { calls = append(calls, "start"); return nil },
	}, []host.Candidate{candidate})
	model.selected[candidate.ID] = true
	message := model.startCommand()().(startedMsg)
	if message.err != nil || strings.Join(calls, ",") != "configure,start" {
		t.Fatalf("calls=%v err=%v", calls, message.err)
	}
}

func TestLiveStatusRendersOfferHealthAndRequestCount(t *testing.T) {
	model := NewModel(Dependencies{}, nil)
	model.screen, model.running = ScreenStatus, true
	model.applySnapshot(snapshotMsg{snapshot: provider.StatusSnapshot{Connected: true, Requests: 12, Offers: []provider.OfferStatus{{OfferID: "qwen", Model: "qwen2.5", Healthy: false, Error: "Ollama stopped"}}}})
	view := model.ViewText()
	for _, expected := range []string{"Connected", "12 completed requests", "qwen2.5", "Ollama stopped"} {
		if !strings.Contains(view, expected) {
			t.Fatalf("view=%q missing=%q", view, expected)
		}
	}
}
