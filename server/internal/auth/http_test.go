package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSessionCookieModeSupportsHostedCrossOriginWebClient(t *testing.T) {
	if got := sessionSameSite(true); got != http.SameSiteNoneMode {
		t.Fatalf("secure hosted cookie SameSite=%v", got)
	}
	if got := sessionSameSite(false); got != http.SameSiteLaxMode {
		t.Fatalf("local cookie SameSite=%v", got)
	}
}

func TestSessionProbeIsQuietWhenSignedOut(t *testing.T) {
	handler := NewHandler(nil, HTTPConfig{AllowedOrigins: []string{testWebOrigin}})

	probe := httptest.NewRequest(http.MethodGet, "/auth/session", nil)
	probe.Header.Set("Origin", testWebOrigin)
	probeResponse := httptest.NewRecorder()
	handler.ServeHTTP(probeResponse, probe)
	if probeResponse.Code != http.StatusNoContent {
		t.Fatalf("signed-out session probe status=%d, want %d", probeResponse.Code, http.StatusNoContent)
	}

	protected := httptest.NewRequest(http.MethodPost, "/auth/device/inspect", nil)
	protected.Header.Set("Origin", testWebOrigin)
	protectedResponse := httptest.NewRecorder()
	handler.ServeHTTP(protectedResponse, protected)
	if protectedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("signed-out protected request status=%d, want %d", protectedResponse.Code, http.StatusUnauthorized)
	}
}
