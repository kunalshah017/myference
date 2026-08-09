package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kunalshah017/myference/cli/internal/host"
)

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
