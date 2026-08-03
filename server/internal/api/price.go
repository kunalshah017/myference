package api

import (
	"encoding/json"
	"errors"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

const coinGeckoMONPriceURL = "https://api.coingecko.com/api/v3/simple/price?ids=monad&vs_currencies=usd&include_last_updated_at=true"

type ReferencePriceConfig struct {
	Endpoint   string
	HTTPClient *http.Client
	CacheTTL   time.Duration
	MaxAge     time.Duration
	Now        func() time.Time
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
			log.Printf("reference price refresh failed: %v", err)
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
	request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, h.config.Endpoint, nil)
	if err != nil {
		return referencePrice{}, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Myference/0.1 (+https://github.com/kunalshah017/myference)")
	response, err := h.config.HTTPClient.Do(request)
	if err != nil {
		return referencePrice{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return referencePrice{}, errors.New("price provider unavailable")
	}
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
	value := strings.TrimSpace(payload.Monad.USD.String())
	price, ok := new(big.Rat).SetString(value)
	updated := time.Unix(payload.Monad.LastUpdatedAt, 0).UTC()
	if !ok || price.Sign() <= 0 || payload.Monad.LastUpdatedAt <= 0 || updated.After(now.Add(time.Minute)) || now.Sub(updated) > h.config.MaxAge {
		return referencePrice{}, errors.New("invalid or stale price")
	}
	return referencePrice{Symbol: "MON", USDPerMON: value, Source: "CoinGecko", UpdatedAt: updated}, nil
}
