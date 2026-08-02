package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRootHandlerMountsRelayInferenceAndAccountAPIs(t *testing.T) {
	marker := func(name string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(name)) })
	}
	handler := newRootHandler(marker("relay"), marker("openai"), marker("auth"), marker("market"), marker("events"))
	for path, expected := range map[string]string{
		"/relay":               "relay",
		"/v1/chat/completions": "openai",
		"/auth/session":        "auth",
		"/api/models":          "market",
		"/events":              "events",
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK || response.Body.String() != expected {
			t.Fatalf("%s returned %d %q", path, response.Code, response.Body.String())
		}
	}
}
