package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

const coinGeckoMONPriceURL = "https://api.coingecko.com/api/v3/simple/price?ids=monad&vs_currencies=usd&include_last_updated_at=true"
const defiLlamaMONPriceURL = "https://coins.llama.fi/prices/current/coingecko:monad"

type ReferencePriceConfig struct {
	Endpoint         string
	FallbackEndpoint string
	HTTPClient       *http.Client
	CacheTTL         time.Duration
	MaxAge           time.Duration
	Now              func() time.Time
}

type referencePrice struct {
	Symbol    string    `json:"symbol"`
	USDPerMON string    `json:"usd_per_mon"`
	Source    string    `json:"source"`
	UpdatedAt time.Time `json:"updated_at"`
}

type referencePriceHandler struct {
	config     ReferencePriceConfig
	mu         sync.Mutex
	cached     referencePrice
	fetchedAt  time.Time
	refreshing bool
}

func NewReferencePrice(config ReferencePriceConfig) http.Handler {
	if config.Endpoint == "" {
		config.Endpoint = coinGeckoMONPriceURL
		config.FallbackEndpoint = defiLlamaMONPriceURL
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	}
	if config.CacheTTL <= 0 {
		config.CacheTTL = time.Minute
	}
	if config.MaxAge <= 0 {
		config.MaxAge = 15 * time.Minute
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	return &referencePriceHandler{config: config}
}

func (h *referencePriceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	now := h.config.Now().UTC()
	quote := h.cached
	shouldFetch := (h.fetchedAt.IsZero() || now.Sub(h.fetchedAt) >= h.config.CacheTTL) && !h.refreshing
	if shouldFetch {
		h.refreshing = true
	}
	h.mu.Unlock()
	if shouldFetch {
		if fetched, err := h.fetch(r, now); err == nil {
			quote = fetched
			h.mu.Lock()
			h.cached = fetched
			h.mu.Unlock()
		} else {
			slog.Error("reference price refresh failed", "error", err)
		}
		h.mu.Lock()
		h.fetchedAt, h.refreshing = now, false
		h.mu.Unlock()
	}
	if quote.USDPerMON == "" || now.Sub(quote.UpdatedAt) > h.config.MaxAge {
		http.Error(w, "MON reference price unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	_ = json.NewEncoder(w).Encode(quote)
}

func (h *referencePriceHandler) fetch(r *http.Request, now time.Time) (referencePrice, error) {
	quote, primaryErr := h.fetchCoinGecko(r, now)
	if primaryErr == nil || h.config.FallbackEndpoint == "" {
		return quote, primaryErr
	}
	quote, fallbackErr := h.fetchDefiLlama(r, now)
	if fallbackErr != nil {
		return referencePrice{}, fmt.Errorf("primary: %v; fallback: %v", primaryErr, fallbackErr)
	}
	return quote, nil
}

func (h *referencePriceHandler) request(r *http.Request, endpoint string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Myference/0.1 (+https://github.com/kunalshah017/myference)")
	response, err := h.config.HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, fmt.Errorf("price provider status %d", response.StatusCode)
	}
	return response, nil
}

func (h *referencePriceHandler) fetchCoinGecko(r *http.Request, now time.Time) (referencePrice, error) {
	response, err := h.request(r, h.config.Endpoint)
	if err != nil {
		return referencePrice{}, err
	}
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	decoder.UseNumber()
	var payload struct {
		Monad struct {
			USD           json.Number `json:"usd"`
			LastUpdatedAt int64       `json:"last_updated_at"`
		} `json:"monad"`
	}
	if err := decoder.Decode(&payload); err != nil {
		return referencePrice{}, err
	}
	return validateReferencePrice(strings.TrimSpace(payload.Monad.USD.String()), payload.Monad.LastUpdatedAt, "CoinGecko", now, h.config.MaxAge)
}

func (h *referencePriceHandler) fetchDefiLlama(r *http.Request, now time.Time) (referencePrice, error) {
	response, err := h.request(r, h.config.FallbackEndpoint)
	if err != nil {
		return referencePrice{}, err
	}
	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	decoder.UseNumber()
	var payload struct {
		Coins map[string]struct {
			Price      json.Number `json:"price"`
			Symbol     string      `json:"symbol"`
			Timestamp  int64       `json:"timestamp"`
			Confidence json.Number `json:"confidence"`
		} `json:"coins"`
	}
	if err := decoder.Decode(&payload); err != nil {
		return referencePrice{}, err
	}
	coin := payload.Coins["coingecko:monad"]
	confidence, _ := new(big.Rat).SetString(coin.Confidence.String())
	if coin.Symbol != "MON" || confidence == nil || confidence.Cmp(big.NewRat(9, 10)) < 0 {
		return referencePrice{}, errors.New("invalid price confidence")
	}
	return validateReferencePrice(strings.TrimSpace(coin.Price.String()), coin.Timestamp, "DefiLlama", now, h.config.MaxAge)
}

func validateReferencePrice(value string, timestamp int64, source string, now time.Time, maxAge time.Duration) (referencePrice, error) {
	price, ok := new(big.Rat).SetString(value)
	updated := time.Unix(timestamp, 0).UTC()
	if !ok || price.Sign() <= 0 || timestamp <= 0 || updated.After(now.Add(time.Minute)) || now.Sub(updated) > maxAge {
		return referencePrice{}, errors.New("invalid or stale price")
	}
	return referencePrice{Symbol: "MON", USDPerMON: value, Source: source, UpdatedAt: updated}, nil
}
