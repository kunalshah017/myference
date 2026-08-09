package account

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type activationRoundTrip func(*http.Request) (*http.Response, error)

func (fn activationRoundTrip) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestProviderActivationClientCreatesAndPollsWithMachineCredential(t *testing.T) {
	calls := 0
	httpClient := &http.Client{Transport: activationRoundTrip(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.Header.Get("Authorization") != "Bearer machine-token" {
			t.Fatalf("authorization=%q", request.Header.Get("Authorization"))
		}
		if calls == 1 {
			if request.Method != http.MethodPost || request.URL.Path != "/api/provider/activations" {
				t.Fatalf("create request=%s %s", request.Method, request.URL.Path)
			}
			body, _ := io.ReadAll(request.Body)
			if !strings.Contains(string(body), `"offer_id":"qwen"`) {
				t.Fatalf("body=%s", body)
			}
			return activationResponse(http.StatusCreated, `{"id":"draft-1","status":"pending","offers":[{"offer_id":"qwen","model":"qwen2.5","kind":"ollama"}],"expires_at":"2026-08-09T12:00:00Z"}`), nil
		}
		if request.Method != http.MethodGet || request.URL.Path != "/api/provider/activations/draft-1" {
			t.Fatalf("poll request=%s %s", request.Method, request.URL.Path)
		}
		return activationResponse(http.StatusOK, `{"id":"draft-1","status":"confirmed","offers":[{"offer_id":"qwen","model":"qwen2.5","kind":"ollama"}],"versions":{"qwen":4},"expires_at":"2026-08-09T12:00:00Z"}`), nil
	})}
	client, err := NewClient("https://api.myference.test", httpClient)
	if err != nil {
		t.Fatal(err)
	}
	draft, err := client.CreateProviderActivation(context.Background(), "machine-token", []ActivationOffer{{OfferID: "qwen", Model: "qwen2.5", Kind: "ollama"}})
	if err != nil || draft.ID != "draft-1" || draft.Status != ActivationPending {
		t.Fatalf("draft=%+v err=%v", draft, err)
	}
	confirmed, err := client.ProviderActivation(context.Background(), "machine-token", draft.ID)
	if err != nil || confirmed.Status != ActivationConfirmed || confirmed.Versions["qwen"] != 4 {
		t.Fatalf("confirmed=%+v err=%v", confirmed, err)
	}
}

func activationResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Status: http.StatusText(status), Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}
