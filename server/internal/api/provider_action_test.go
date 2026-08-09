package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProviderActionRequiresWalletSubmissionAndIndexedEvidence(t *testing.T) {
	now := time.Unix(100, 0)
	store := NewProviderActionStore(func() time.Time { return now })
	indexed := false
	handler := NewProviderActions(store, ProviderActionDependencies{
		MachineAuth: func(r *http.Request) (string, string, error) {
			if r.Header.Get("Authorization") != "Bearer machine-token" {
				return "", "", errors.New("unauthorized")
			}
			return "machine-1", "account-1", nil
		},
		AccountAuth: func(*http.Request) (string, error) { return "account-1", nil },
		Prepare: func(_ context.Context, source, machineID, accountID string, input ProviderActionInput) (string, ProviderActionBaseline, error) {
			if source != ActionSourceMachine || machineID != "machine-1" || accountID != "account-1" || input.Kind != ActionPublishOffer {
				t.Fatalf("prepare source=%s machine=%s account=%s input=%+v", source, machineID, accountID, input)
			}
			return "0x1111111111111111111111111111111111111111", ProviderActionBaseline{Versions: map[string]uint64{"local-qwen": 2}}, nil
		},
		Verify: func(_ context.Context, action ProviderAction) (map[string]uint64, bool, error) {
			return map[string]uint64{"local-qwen": 3}, indexed, nil
		},
	})

	create := httptest.NewRequest(http.MethodPost, "/api/provider/actions", bytes.NewBufferString(`{"kind":"publish_offer","offers":[{"offer_id":"local-qwen","model":"qwen","kind":"ollama","capabilities":["stream","text"],"metering_mode":"tokens_and_compute","input_per_million_wei":"10","output_per_million_wei":"20","compute_per_second_wei":"30"}]}`))
	create.Header.Set("Authorization", "Bearer machine-token")
	created := httptest.NewRecorder()
	handler.ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var action ProviderAction
	if err := json.NewDecoder(created.Body).Decode(&action); err != nil || action.ID == "" || action.Status != ActionPendingWallet || action.WalletAddress == "" {
		t.Fatalf("action=%+v err=%v", action, err)
	}

	submit := httptest.NewRequest(http.MethodPost, "/api/provider/actions/"+action.ID+"/submitted", bytes.NewBufferString(`{"transaction_hashes":["0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"]}`))
	submitted := httptest.NewRecorder()
	handler.ServeHTTP(submitted, submit)
	if submitted.Code != http.StatusOK {
		t.Fatalf("submit status=%d body=%s", submitted.Code, submitted.Body.String())
	}
	if err := json.NewDecoder(submitted.Body).Decode(&action); err != nil || action.Status != ActionPendingChain {
		t.Fatalf("pending action=%+v err=%v", action, err)
	}

	indexed = true
	poll := httptest.NewRecorder()
	handler.ServeHTTP(poll, httptest.NewRequest(http.MethodGet, "/api/provider/actions/"+action.ID, nil))
	if err := json.NewDecoder(poll.Body).Decode(&action); err != nil || action.Status != ActionConfirmed || action.Versions["local-qwen"] != 3 {
		t.Fatalf("confirmed action=%+v err=%v body=%s", action, err, poll.Body.String())
	}
}

func TestProviderActionValidatesKindsAndPreservesDraftAcrossCrossAccountReads(t *testing.T) {
	store := NewProviderActionStore(time.Now)
	prepared := func(_ context.Context, _, machineID, accountID string, input ProviderActionInput) (string, ProviderActionBaseline, error) {
		return "0x1111111111111111111111111111111111111111", ProviderActionBaseline{BondWei: "100", ExitAvailableAt: 0}, nil
	}
	owner := NewProviderActions(store, ProviderActionDependencies{
		MachineAuth: func(*http.Request) (string, string, error) { return "machine", "owner", nil },
		AccountAuth: func(*http.Request) (string, error) { return "owner", nil }, Prepare: prepared,
	})
	create := httptest.NewRecorder()
	owner.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/provider/actions", bytes.NewBufferString(`{"kind":"deposit_collateral","amount_wei":"25"}`)))
	if create.Code != http.StatusCreated {
		t.Fatalf("create=%d %s", create.Code, create.Body.String())
	}
	var action ProviderAction
	_ = json.NewDecoder(create.Body).Decode(&action)

	other := NewProviderActions(store, ProviderActionDependencies{
		MachineAuth: func(*http.Request) (string, string, error) { return "other-machine", "other", nil },
		AccountAuth: func(*http.Request) (string, error) { return "other", nil }, Prepare: prepared,
	})
	read := httptest.NewRecorder()
	other.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/api/provider/actions/"+action.ID, nil))
	if read.Code != http.StatusNotFound {
		t.Fatalf("cross-account status=%d", read.Code)
	}
	read = httptest.NewRecorder()
	owner.ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/api/provider/actions/"+action.ID, nil))
	if read.Code != http.StatusOK {
		t.Fatalf("owner draft removed: status=%d", read.Code)
	}

	invalid := httptest.NewRecorder()
	owner.ServeHTTP(invalid, httptest.NewRequest(http.MethodPost, "/api/provider/actions", bytes.NewBufferString(`{"kind":"deposit_collateral","amount_wei":"0"}`)))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid deposit status=%d", invalid.Code)
	}
}
