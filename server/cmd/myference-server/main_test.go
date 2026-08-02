package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChainRuntimeConfigurationRejectsMissingAndLocalMainnetMixups(t *testing.T) {
	if _, err := loadChainConfig(func(string) string { return "" }); err == nil {
		t.Fatal("accepted missing chain runtime")
	}
	values := map[string]string{"MYFERENCE_RPC_URL": "http://127.0.0.1:8546", "MYFERENCE_CONTRACT_ADDRESS": "0x4444444444444444444444444444444444444444", "MYFERENCE_SETTLEMENT_PRIVATE_KEY": strings.Repeat("1", 64), "MYFERENCE_CHAIN_START_BLOCK": "10"}
	config, err := loadChainConfig(func(name string) string { return values[name] })
	if err != nil || config.StartBlock != 10 {
		t.Fatalf("config=%+v err=%v", config, err)
	}
}

func TestRootHandlerMountsRelayInferenceAndAccountAPIs(t *testing.T) {
	marker := func(name string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(name)) })
	}
	handler := newRootHandler(marker("relay"), marker("openai"), marker("anthropic"), marker("auth"), marker("market"), marker("operations"), marker("analytics"), marker("events"))
	for path, expected := range map[string]string{
		"/healthz":                "ok\n",
		"/relay":                  "relay",
		"/v1/chat/completions":    "openai",
		"/v1/messages":            "anthropic",
		"/auth/session":           "auth",
		"/api/models":             "market",
		"/api/account/operations": "operations",
		"/api/account/analytics":  "analytics",
		"/events":                 "events",
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK || response.Body.String() != expected {
			t.Fatalf("%s returned %d %q", path, response.Code, response.Body.String())
		}
	}
}

func TestListenAddressUsesRenderPortWhenConfigured(t *testing.T) {
	values := map[string]string{"PORT": "10000"}
	if got := listenAddress(func(name string) string { return values[name] }); got != "0.0.0.0:10000" {
		t.Fatalf("listenAddress() = %q", got)
	}
	values["MYFERENCE_LISTEN_ADDR"] = "127.0.0.1:9090"
	if got := listenAddress(func(name string) string { return values[name] }); got != "127.0.0.1:9090" {
		t.Fatalf("explicit listen address = %q", got)
	}
}
