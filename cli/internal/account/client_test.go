package account

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientCreatesAndExchangesDeviceAuthorization(t *testing.T) {
	polls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth/device":
			var input map[string]string
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input["machine_name"] != "render-node" || input["signer_address"] != "0x0000000000000000000000000000000000001234" {
				t.Fatalf("unexpected device request: %+v err=%v", input, err)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"device_code":"device-secret","user_code":"ABCD1234","verification_uri":"https://app.myference.test/devices","expires_at":"2026-08-02T18:00:00Z","chain_id":10143,"contract_address":"0x4444444444444444444444444444444444444444"}`))
		case "/auth/device/token":
			polls++
			if polls == 1 {
				http.Error(w, "device authorization pending", http.StatusTooEarly)
				return
			}
			_, _ = w.Write([]byte(`{"machine":{"ID":"mach-1","AccountID":"acct-1","Name":"render-node"},"token":"mach-1.secret"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	authz, err := client.CreateDeviceAuthorization(t.Context(), "render-node", "0x0000000000000000000000000000000000001234")
	if err != nil || authz.UserCode != "ABCD1234" || authz.ChainID != 10143 || authz.ContractAddress != "0x4444444444444444444444444444444444444444" {
		t.Fatalf("authz=%+v err=%v", authz, err)
	}
	if _, err := client.ExchangeDeviceAuthorization(t.Context(), authz.DeviceCode); err != ErrPending {
		t.Fatalf("expected pending, got %v", err)
	}
	token, err := client.ExchangeDeviceAuthorization(t.Context(), authz.DeviceCode)
	if err != nil || token.Machine.ID != "mach-1" || token.Token != "mach-1.secret" {
		t.Fatalf("token=%+v err=%v", token, err)
	}
	if authz.ExpiresAt.Before(time.Date(2026, 8, 2, 17, 59, 0, 0, time.UTC)) {
		t.Fatalf("unexpected expiry: %s", authz.ExpiresAt)
	}
}
