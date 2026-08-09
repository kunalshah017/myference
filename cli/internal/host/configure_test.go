package host

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/kunalshah017/myference/cli/internal/config"
)

func TestApplyIsIdempotentAndPreservesOfferIdentity(t *testing.T) {
	current := config.Config{ServerURL: "https://api.example", AccountID: "account", MachineID: "machine", Backends: []config.Backend{
		{Name: "machine-qwen", OfferID: "my-qwen-offer", Kind: "ollama", URL: "http://127.0.0.1:11434", Model: "qwen", PriceVersion: 7, Enabled: true},
		{Name: "old", Kind: "ollama", URL: "http://127.0.0.1:11434", Model: "old", PriceVersion: 2, Enabled: true},
	}}
	selection := []Selection{{Candidate: Candidate{Kind: "ollama", Name: "Ollama", URL: "http://127.0.0.1:11434", Model: "qwen"}}}
	var saved config.Config
	updated, err := Apply(context.Background(), current, selection, CredentialStore{}, func(value config.Config) error {
		saved = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Backends) != 2 || updated.Backends[0].Name != "machine-qwen" || updated.Backends[0].OfferID != "my-qwen-offer" || updated.Backends[0].PriceVersion != 7 || !updated.Backends[0].Enabled || updated.Backends[1].Enabled {
		t.Fatalf("backends=%+v", updated.Backends)
	}
	if !reflect.DeepEqual(saved, updated) {
		t.Fatalf("saved=%+v updated=%+v", saved, updated)
	}
}

func TestApplyStoresAPISecretOutsideConfig(t *testing.T) {
	current := config.Config{ServerURL: "https://api.example", AccountID: "account", MachineID: "machine"}
	var service, account, secret string
	selection := []Selection{{Candidate: Candidate{Kind: "openai", Name: "OpenAI", URL: "https://api.openai.com", Model: "gpt-test"}, Secret: "top-secret"}}
	updated, err := Apply(context.Background(), current, selection, CredentialStore{Service: "myference.backend", Save: func(gotService, gotAccount, gotSecret string) error {
		service, account, secret = gotService, gotAccount, gotSecret
		return nil
	}}, func(config.Config) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.Backends) != 1 || updated.Backends[0].URL != "https://api.openai.com" || updated.Backends[0].Name == "" || updated.Backends[0].PriceVersion != 0 {
		t.Fatalf("backends=%+v", updated.Backends)
	}
	if service != "myference.backend" || account != "machine/"+updated.Backends[0].Name || secret != "top-secret" {
		t.Fatalf("service=%q account=%q secret=%q", service, account, secret)
	}
}

func TestApplyRollsBackNewCredentialWhenConfigSaveFails(t *testing.T) {
	current := config.Config{ServerURL: "https://api.example", AccountID: "account", MachineID: "machine"}
	deleted := ""
	selection := []Selection{{Candidate: Candidate{Kind: "openai", Name: "OpenAI", URL: "https://api.openai.com", Model: "gpt-test"}, Secret: "top-secret"}}
	updated, err := Apply(context.Background(), current, selection, CredentialStore{
		Service: "myference.backend",
		Load:    func(string, string) (string, error) { return "", errors.New("not found") },
		Save:    func(string, string, string) error { return nil },
		Delete:  func(_ string, account string) error { deleted = account; return nil },
	}, func(config.Config) error { return errors.New("disk full") })
	if err == nil || !reflect.DeepEqual(updated, current) || deleted == "" {
		t.Fatalf("updated=%+v deleted=%q err=%v", updated, deleted, err)
	}
}

func TestBackendNameIsStableAndSafe(t *testing.T) {
	candidate := Candidate{Kind: "openai", Name: "Example", URL: "https://provider.example/v1", Model: "Acme/Model:Latest"}
	first := BackendName(candidate)
	if first != BackendName(candidate) || first != "example-acme-model-latest" {
		t.Fatalf("name=%q", first)
	}
}
