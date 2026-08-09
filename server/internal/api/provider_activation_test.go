package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProviderActivationMachineCreatesAndPollsBrowserCompletion(t *testing.T) {
	store := NewActivationStore(func() time.Time { return time.Unix(100, 0) })
	handler := NewProviderActivation(store,
		func(request *http.Request) (string, string, error) {
			if request.Header.Get("Authorization") != "Bearer machine-token" {
				return "", "", errors.New("unauthorized")
			}
			return "machine-1", "account-1", nil
		},
		func(*http.Request) (string, error) { return "account-1", nil },
	)

	create := httptest.NewRequest(http.MethodPost, "/api/provider/activations", bytes.NewBufferString(`{"offers":[{"offer_id":"local-qwen","model":"qwen","kind":"ollama"}]}`))
	create.Header.Set("Authorization", "Bearer machine-token")
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", created.Code, created.Body.String())
	}
	var draft ProviderActivation
	if err := json.NewDecoder(created.Body).Decode(&draft); err != nil || draft.ID == "" || draft.Status != ActivationPending || len(draft.Offers) != 1 {
		t.Fatalf("draft=%+v err=%v", draft, err)
	}

	read := httptest.NewRequest(http.MethodGet, "/api/provider/activations/"+draft.ID, nil)
	readResult := httptest.NewRecorder()
	handler.ServeHTTP(readResult, read)
	if readResult.Code != http.StatusOK || !bytes.Contains(readResult.Body.Bytes(), []byte("local-qwen")) {
		t.Fatalf("status=%d body=%s", readResult.Code, readResult.Body.String())
	}

	complete := httptest.NewRequest(http.MethodPost, "/api/provider/activations/"+draft.ID+"/complete", bytes.NewBufferString(`{"versions":{"local-qwen":3}}`))
	completed := httptest.NewRecorder()
	handler.ServeHTTP(completed, complete)
	if completed.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", completed.Code, completed.Body.String())
	}

	poll := httptest.NewRequest(http.MethodGet, "/api/provider/activations/"+draft.ID, nil)
	poll.Header.Set("Authorization", "Bearer machine-token")
	polled := httptest.NewRecorder()
	handler.ServeHTTP(polled, poll)
	if err := json.NewDecoder(polled.Body).Decode(&draft); err != nil || draft.Status != ActivationConfirmed || draft.Versions["local-qwen"] != 3 {
		t.Fatalf("draft=%+v err=%v", draft, err)
	}
}

func TestProviderActivationRejectsCrossAccountAndIncompleteVersions(t *testing.T) {
	store := NewActivationStore(time.Now)
	draft, err := store.Create("machine-1", "account-1", []ActivationOffer{{OfferID: "one", Model: "m1", Kind: "ollama"}, {OfferID: "two", Model: "m2", Kind: "codex"}}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewProviderActivation(store,
		func(*http.Request) (string, string, error) { return "machine-1", "account-1", nil },
		func(*http.Request) (string, error) { return "account-2", nil },
	)
	read := httptest.NewRecorder()
	handler.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/api/provider/activations/"+draft.ID, nil))
	if read.Code != http.StatusNotFound {
		t.Fatalf("cross-account status=%d", read.Code)
	}
	if _, err := store.Get(draft.ID); err != nil {
		t.Fatalf("cross-account read removed activation: %v", err)
	}

	handler = NewProviderActivation(store,
		func(*http.Request) (string, string, error) { return "machine-1", "account-1", nil },
		func(*http.Request) (string, error) { return "account-1", nil },
	)
	complete := httptest.NewRecorder()
	handler.ServeHTTP(complete, httptest.NewRequest(http.MethodPost, "/api/provider/activations/"+draft.ID+"/complete", bytes.NewBufferString(`{"versions":{"one":1}}`)))
	if complete.Code != http.StatusBadRequest {
		t.Fatalf("incomplete status=%d body=%s", complete.Code, complete.Body.String())
	}
}

func TestProviderActivationExpires(t *testing.T) {
	now := time.Unix(100, 0)
	store := NewActivationStore(func() time.Time { return now })
	draft, err := store.Create("machine", "account", []ActivationOffer{{OfferID: "one", Model: "m", Kind: "ollama"}}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Minute)
	if _, err := store.Get(draft.ID); !errors.Is(err, ErrActivationNotFound) {
		t.Fatalf("error=%v", err)
	}
}
