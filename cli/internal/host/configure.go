package host

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/kunalshah017/myference/cli/internal/config"
)

type Selection struct {
	Candidate Candidate
	Secret    string
}

type CredentialStore struct {
	Service string
	Load    func(string, string) (string, error)
	Save    func(string, string, string) error
	Delete  func(string, string) error
}

type credentialChange struct {
	account, previous string
	existed           bool
}

func Apply(ctx context.Context, current config.Config, selections []Selection, store CredentialStore, saveConfig func(config.Config) error) (config.Config, error) {
	if saveConfig == nil {
		return current, errors.New("configuration saver is required")
	}
	if err := ctx.Err(); err != nil {
		return current, err
	}
	updated := current
	updated.Backends = append([]config.Backend(nil), current.Backends...)
	for index := range updated.Backends {
		updated.Backends[index].Enabled = false
	}
	changes := make([]credentialChange, 0)
	rollback := func() {
		for index := len(changes) - 1; index >= 0; index-- {
			change := changes[index]
			if change.existed && store.Save != nil {
				_ = store.Save(store.Service, change.account, change.previous)
			} else if !change.existed && store.Delete != nil {
				_ = store.Delete(store.Service, change.account)
			}
		}
	}
	for _, selection := range selections {
		candidate := selection.Candidate
		identity := StableID(candidate)
		index := slices.IndexFunc(updated.Backends, func(item config.Backend) bool {
			return StableID(Candidate{Kind: item.Kind, URL: item.URL, Model: item.Model, Image: item.Image}) == identity
		})
		item := config.Backend{Name: BackendName(candidate), Kind: candidate.Kind, URL: candidate.URL, Model: candidate.Model, Image: candidate.Image, Enabled: true}
		if candidate.Kind != "ollama" && candidate.Kind != "openai" {
			item.URL = ""
		}
		if index >= 0 {
			item.Name = updated.Backends[index].Name
			item.PriceVersion = updated.Backends[index].PriceVersion
			updated.Backends[index] = item
		} else {
			item.Name = uniqueBackendName(item.Name, updated.Backends)
			updated.Backends = append(updated.Backends, item)
		}
		if strings.TrimSpace(selection.Secret) == "" {
			continue
		}
		if store.Service == "" || store.Save == nil {
			rollback()
			return current, errors.New("credential store is unavailable")
		}
		account := current.MachineID + "/" + item.Name
		change := credentialChange{account: account}
		if store.Load != nil {
			if previous, err := store.Load(store.Service, account); err == nil {
				change.previous, change.existed = previous, true
			}
		}
		if err := store.Save(store.Service, account, selection.Secret); err != nil {
			rollback()
			return current, fmt.Errorf("store backend credential: %w", err)
		}
		changes = append(changes, change)
	}
	if err := saveConfig(updated); err != nil {
		rollback()
		return current, err
	}
	return updated, nil
}

func BackendName(candidate Candidate) string {
	prefix := candidate.Name
	if strings.TrimSpace(prefix) == "" {
		prefix = candidate.Kind
	}
	return safeName(prefix + "-" + candidate.Model)
}

func safeName(value string) string {
	var result strings.Builder
	dash := false
	for _, character := range strings.ToLower(value) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			if dash && result.Len() > 0 {
				result.WriteByte('-')
			}
			result.WriteRune(character)
			dash = false
		} else {
			dash = result.Len() > 0
		}
	}
	return strings.Trim(result.String(), "-")
}

func uniqueBackendName(base string, backends []config.Backend) string {
	if !slices.ContainsFunc(backends, func(item config.Backend) bool { return item.Name == base }) {
		return base
	}
	for suffix := 2; ; suffix++ {
		candidate := fmt.Sprintf("%s-%d", base, suffix)
		if !slices.ContainsFunc(backends, func(item config.Backend) bool { return item.Name == candidate }) {
			return candidate
		}
	}
}
