package account

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type providerRoundTrip func(*http.Request) (*http.Response, error)

func (fn providerRoundTrip) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestProviderClientUsesMachineCredentialForActionsAndVersions(t *testing.T) {
	calls := 0
	httpClient := &http.Client{Transport: providerRoundTrip(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.Header.Get("Authorization") != "Bearer machine-token" {
			t.Fatalf("authorization=%q", request.Header.Get("Authorization"))
		}
		switch calls {
		case 1:
			body, _ := io.ReadAll(request.Body)
			if request.Method != http.MethodPost || request.URL.Path != "/api/provider/actions" || !strings.Contains(string(body), `"kind":"publish_offer"`) {
				t.Fatalf("create=%s %s body=%s", request.Method, request.URL.Path, body)
			}
			return providerResponse(http.StatusCreated, `{"id":"action-1","status":"pending_wallet","expires_at":"2026-08-09T12:00:00Z"}`), nil
		case 2:
			if request.URL.Path != "/api/provider/actions/action-1" {
				t.Fatalf("poll path=%s", request.URL.Path)
			}
			return providerResponse(http.StatusOK, `{"id":"action-1","status":"confirmed","versions":{"qwen":4},"expires_at":"2026-08-09T12:00:00Z"}`), nil
		default:
			if request.URL.Path != "/api/provider/machines/machine-1/offer-versions" {
				t.Fatalf("versions path=%s", request.URL.Path)
			}
			return providerResponse(http.StatusOK, `{"qwen":4}`), nil
		}
	})}
	client, err := NewClient("https://api.myference.test", httpClient)
	if err != nil {
		t.Fatal(err)
	}
	draft, err := client.CreateProviderAction(context.Background(), "machine-token", ProviderActionInput{Kind: ActionPublishOffer, Offers: []ProviderOffer{{OfferID: "qwen"}}})
	if err != nil || draft.ID != "action-1" {
		t.Fatalf("draft=%+v err=%v", draft, err)
	}
	confirmed, err := client.ProviderAction(context.Background(), "machine-token", draft.ID)
	if err != nil || confirmed.Status != ActionConfirmed || confirmed.Versions["qwen"] != 4 {
		t.Fatalf("confirmed=%+v err=%v", confirmed, err)
	}
	versions, err := client.MachineOfferVersions(context.Background(), "machine-token", "machine-1")
	if err != nil || versions["qwen"] != 4 {
		t.Fatalf("versions=%v err=%v", versions, err)
	}
}

func providerResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Status: http.StatusText(status), Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}
