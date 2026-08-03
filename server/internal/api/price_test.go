package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestReferencePriceCachesLastFreshQuote(t *testing.T) {
	now := time.Unix(1_785_717_600, 0).UTC()
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls > 1 {
			http.Error(w, "offline", http.StatusBadGateway)
			return
		}
		fmt.Fprintf(w, `{"monad":{"usd":0.02090926,"last_updated_at":%d}}`, now.Unix())
	}))
	t.Cleanup(upstream.Close)
	handler := NewReferencePrice(ReferencePriceConfig{Endpoint: upstream.URL, HTTPClient: upstream.Client(), CacheTTL: time.Minute, MaxAge: 15 * time.Minute, Now: func() time.Time { return now }})

	for attempt := 0; attempt < 2; attempt++ {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/reference-price", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("attempt %d status=%d body=%q", attempt, response.Code, response.Body.String())
		}
		if got := response.Body.String(); got != `{"symbol":"MON","usd_per_mon":"0.02090926","source":"CoinGecko","updated_at":"2026-08-03T00:40:00Z"}`+"\n" {
			t.Fatalf("unexpected quote: %s", got)
		}
		now = now.Add(30 * time.Second)
	}
	if calls != 1 {
		t.Fatalf("upstream calls=%d, want 1", calls)
	}
}

func TestReferencePriceDoesNotSerializeCallersBehindSlowUpstream(t *testing.T) {
	now := time.Unix(1_785_717_600, 0).UTC()
	started := make(chan struct{})
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		fmt.Fprintf(w, `{"monad":{"usd":0.02,"last_updated_at":%d}}`, now.Unix())
	}))
	t.Cleanup(upstream.Close)
	handler := NewReferencePrice(ReferencePriceConfig{Endpoint: upstream.URL, HTTPClient: upstream.Client(), CacheTTL: time.Minute, MaxAge: 15 * time.Minute, Now: func() time.Time { return now }})
	var group sync.WaitGroup
	group.Add(1)
	go func() {
		defer group.Done()
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/reference-price", nil))
	}()
	<-started
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/reference-price", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("concurrent status=%d, want immediate 503", response.Code)
	}
	close(release)
	group.Wait()
}

func TestReferencePriceRejectsStaleOrUnavailableQuote(t *testing.T) {
	now := time.Unix(1_785_717_600, 0).UTC()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"monad":{"usd":0.02,"last_updated_at":%d}}`, now.Add(-time.Hour).Unix())
	}))
	t.Cleanup(upstream.Close)
	handler := NewReferencePrice(ReferencePriceConfig{Endpoint: upstream.URL, HTTPClient: upstream.Client(), CacheTTL: time.Minute, MaxAge: 15 * time.Minute, Now: func() time.Time { return now }})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/reference-price", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
}

func TestReferencePriceFallsBackToFreshDefiLlamaQuote(t *testing.T) {
	now := time.Unix(1_785_717_600, 0).UTC()
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	t.Cleanup(primary.Close)
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `{"coins":{"coingecko:monad":{"price":0.0208,"symbol":"MON","timestamp":%d,"confidence":0.99}}}`, now.Unix())
	}))
	t.Cleanup(fallback.Close)
	handler := NewReferencePrice(ReferencePriceConfig{Endpoint: primary.URL, FallbackEndpoint: fallback.URL, HTTPClient: fallback.Client(), Now: func() time.Time { return now }})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/reference-price", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"source":"DefiLlama"`) {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
}
