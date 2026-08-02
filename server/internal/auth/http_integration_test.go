package auth

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/kunalshah017/myference/server/internal/store"
)

const testWebOrigin = "https://app.myference.test"

func TestWalletDeviceAndAPIKeyHTTPFlow(t *testing.T) {
	ctx, service := newHTTPIntegrationService(t)
	handler := NewHandler(service, HTTPConfig{
		Domain:          "api.myference.test",
		AllowedOrigins:  []string{testWebOrigin},
		ChainID:         10143,
		SessionLifetime: time.Hour,
		SecureCookies:   false,
		VerificationURL: testWebOrigin + "/devices",
	})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar}

	walletKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	address := crypto.PubkeyToAddress(walletKey.PublicKey).Hex()
	challenge := postJSON[WalletChallenge](t, client, server.URL+"/auth/wallet/challenge", map[string]string{"address": address}, nil, http.StatusCreated)
	for _, expected := range []string{"api.myference.test", testWebOrigin, address, "Chain ID: 10143", challenge.Nonce} {
		if !strings.Contains(challenge.Message, expected) {
			t.Fatalf("challenge message %q does not contain %q", challenge.Message, expected)
		}
	}
	signature := signPersonalMessage(t, walletKey, challenge.Message)
	postJSON[SessionView](t, client, server.URL+"/auth/wallet/verify", map[string]string{
		"challenge_id": challenge.ID,
		"signature":    signature,
	}, nil, http.StatusOK)
	postJSON[map[string]any](t, client, server.URL+"/auth/wallet/verify", map[string]string{
		"challenge_id": challenge.ID,
		"signature":    signature,
	}, nil, http.StatusConflict)
	session := getJSON[SessionView](t, client, server.URL+"/auth/session", nil, http.StatusOK)
	if !strings.EqualFold(session.WalletAddress, address) || session.AccountID == "" {
		t.Fatalf("unexpected session: %+v", session)
	}

	device := postJSON[DeviceHTTPAuthorization](t, http.DefaultClient, server.URL+"/auth/device", map[string]string{"machine_name": "studio-mac"}, nil, http.StatusCreated)
	if device.DeviceCode == "" || device.UserCode == "" || device.VerificationURI != testWebOrigin+"/devices" {
		t.Fatalf("unexpected device authorization: %+v", device)
	}
	pending := postJSON[PendingDevice](t, client, server.URL+"/auth/device/inspect", map[string]string{"user_code": device.UserCode}, nil, http.StatusOK)
	if pending.MachineName != "studio-mac" || !pending.ExpiresAt.Equal(device.ExpiresAt) {
		t.Fatalf("unexpected pending machine: %+v", pending)
	}
	postJSON[map[string]any](t, client, server.URL+"/auth/device/approve", map[string]string{"user_code": device.UserCode}, nil, http.StatusNoContent)
	exchanged := postJSON[DeviceToken](t, http.DefaultClient, server.URL+"/auth/device/token", map[string]string{"device_code": device.DeviceCode}, nil, http.StatusOK)
	if exchanged.Machine.ID == "" || exchanged.Machine.Name != "studio-mac" || exchanged.Token == "" {
		t.Fatalf("unexpected device exchange: %+v", exchanged)
	}
	postJSON[map[string]any](t, http.DefaultClient, server.URL+"/auth/device/token", map[string]string{"device_code": device.DeviceCode}, nil, http.StatusConflict)

	created := postJSON[APIKeyView](t, client, server.URL+"/auth/api-keys", APIKeyScope{
		Models: []string{"qwen2.5:0.5b"}, Endpoints: []string{"/v1/chat/completions"}, MaxSpendWei: 1000,
	}, nil, http.StatusCreated)
	if created.Token == "" || created.ID == "" {
		t.Fatalf("API key must be revealed on creation: %+v", created)
	}
	keys := getJSON[[]APIKeyView](t, client, server.URL+"/auth/api-keys", nil, http.StatusOK)
	if len(keys) != 1 || keys[0].ID != created.ID || keys[0].Token != "" || keys[0].Scope.MaxSpendWei != 1000 {
		t.Fatalf("API key list leaked or lost metadata: %+v", keys)
	}
	requestJSON[map[string]any](t, client, http.MethodDelete, server.URL+"/auth/api-keys/"+created.ID, nil, nil, http.StatusNoContent)
	if _, err := service.AuthorizeAPIKey(ctx, created.Token, "qwen2.5:0.5b", "/v1/chat/completions", 1); err != ErrInvalidCredential {
		t.Fatalf("expected revoked key rejection, got %v", err)
	}
}

func TestWalletHTTPRejectsWrongOriginChainAndExpiredDevice(t *testing.T) {
	_, service := newHTTPIntegrationService(t)
	handler := NewHandler(service, HTTPConfig{Domain: "api.myference.test", AllowedOrigins: []string{testWebOrigin}, ChainID: 10143, SessionLifetime: time.Hour})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	postJSON[map[string]any](t, http.DefaultClient, server.URL+"/auth/wallet/challenge", map[string]string{"address": "0x0000000000000000000000000000000000000001"}, map[string]string{"Origin": "https://evil.test"}, http.StatusForbidden)

	device, err := service.CreateDeviceAuthorization(context.Background(), "expired-machine", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return device.ExpiresAt.Add(time.Second) }
	postJSON[map[string]any](t, http.DefaultClient, server.URL+"/auth/device/token", map[string]string{"device_code": device.DeviceCode}, nil, http.StatusGone)
}

func newHTTPIntegrationService(t *testing.T) (context.Context, *Service) {
	t.Helper()
	databaseURL := os.Getenv("MYFERENCE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("MYFERENCE_TEST_DATABASE_URL is required for PostgreSQL integration")
	}
	ctx := context.Background()
	control, err := store.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { control.Close() })
	for _, name := range []string{"000001_control_plane.sql", "000004_browser_auth.sql"} {
		if err := control.ApplyMigration(ctx, filepath.Join("..", "..", "..", "migrations", name)); err != nil {
			t.Fatal(err)
		}
	}
	service, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { service.Close() })
	return ctx, service
}

func signPersonalMessage(t *testing.T, key *ecdsa.PrivateKey, message string) string {
	t.Helper()
	signature, err := crypto.Sign(accounts.TextHash([]byte(message)), key)
	if err != nil {
		t.Fatal(err)
	}
	signature[64] += 27
	return "0x" + hex.EncodeToString(signature)
}

func postJSON[T any](t *testing.T, client *http.Client, url string, body any, headers map[string]string, status int) T {
	t.Helper()
	return requestJSON[T](t, client, http.MethodPost, url, body, headers, status)
}

func getJSON[T any](t *testing.T, client *http.Client, url string, headers map[string]string, status int) T {
	t.Helper()
	return requestJSON[T](t, client, http.MethodGet, url, nil, headers, status)
}

func requestJSON[T any](t *testing.T, client *http.Client, method, url string, body any, headers map[string]string, status int) T {
	t.Helper()
	var encoded bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&encoded).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	request, err := http.NewRequest(method, url, &encoded)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Origin", testWebOrigin)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != status {
		var errorBody bytes.Buffer
		errorBody.ReadFrom(response.Body)
		t.Fatalf("%s %s status=%d want=%d body=%s", method, url, response.StatusCode, status, errorBody.String())
	}
	var result T
	if status != http.StatusNoContent && status < http.StatusBadRequest {
		if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
			t.Fatal(err)
		}
	}
	return result
}
