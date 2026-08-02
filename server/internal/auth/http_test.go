package auth

import (
	"net/http"
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
