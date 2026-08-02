package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

const browserSessionCookie = "myference_session"

type HTTPConfig struct {
	Domain, VerificationURL, ContractAddress string
	AllowedOrigins                           []string
	ChainID                                  uint64
	SessionLifetime                          time.Duration
	SecureCookies                            bool
}

type SessionView struct {
	AccountID     string    `json:"account_id"`
	WalletAddress string    `json:"wallet_address"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type DeviceHTTPAuthorization struct {
	DeviceCode      string    `json:"device_code"`
	UserCode        string    `json:"user_code"`
	VerificationURI string    `json:"verification_uri"`
	ExpiresAt       time.Time `json:"expires_at"`
	SignerAddress   string    `json:"signer_address"`
	ChainID         uint64    `json:"chain_id"`
	ContractAddress string    `json:"contract_address"`
}

type DeviceToken struct {
	Machine Machine `json:"machine"`
	Token   string  `json:"token"`
}

type APIKeyView struct {
	ID        string      `json:"id"`
	Token     string      `json:"token,omitempty"`
	Scope     APIKeyScope `json:"scope"`
	CreatedAt time.Time   `json:"created_at,omitempty"`
}

type httpHandler struct {
	service *Service
	config  HTTPConfig
}

func NewHandler(service *Service, config HTTPConfig) http.Handler {
	if config.ChainID == 0 {
		config.ChainID = 10143
	}
	if config.SessionLifetime <= 0 {
		config.SessionLifetime = 12 * time.Hour
	}
	h := &httpHandler{service: service, config: config}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/wallet/challenge", h.walletChallenge)
	mux.HandleFunc("POST /auth/wallet/verify", h.walletVerify)
	mux.HandleFunc("GET /auth/session", h.session)
	mux.HandleFunc("POST /auth/device", h.deviceCreate)
	mux.HandleFunc("POST /auth/device/token", h.deviceToken)
	mux.HandleFunc("POST /auth/device/inspect", h.withSession(h.deviceInspect))
	mux.HandleFunc("POST /auth/device/approve", h.withSession(h.deviceApprove))
	mux.HandleFunc("GET /auth/api-keys", h.withSession(h.apiKeysList))
	mux.HandleFunc("POST /auth/api-keys", h.withSession(h.apiKeysCreate))
	mux.HandleFunc("DELETE /auth/api-keys/{id}", h.withSession(h.apiKeysDelete))
	mux.HandleFunc("POST /auth/stream-ticket", h.withSession(h.streamTicket))
	return h.cors(mux)
}

func (h *httpHandler) streamTicket(w http.ResponseWriter, r *http.Request, session BrowserSession) {
	ticket, err := h.service.CreateStreamTicket(r.Context(), session.AccountID, time.Minute)
	h.write(w, ticket, err, http.StatusCreated)
}

func (h *httpHandler) walletChallenge(w http.ResponseWriter, r *http.Request) {
	origin, ok := h.browserOrigin(w, r)
	if !ok {
		return
	}
	var input struct {
		Address string `json:"address"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	challenge, err := h.service.CreateWalletChallenge(r.Context(), h.config.Domain, origin, input.Address, h.config.ChainID, 5*time.Minute)
	h.write(w, challenge, err, http.StatusCreated)
}

func (h *httpHandler) walletVerify(w http.ResponseWriter, r *http.Request) {
	origin, ok := h.browserOrigin(w, r)
	if !ok {
		return
	}
	var input struct {
		ChallengeID string `json:"challenge_id"`
		Signature   string `json:"signature"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	session, err := h.service.VerifyWalletChallenge(r.Context(), input.ChallengeID, origin, input.Signature, h.config.SessionLifetime)
	if err != nil {
		h.write(w, nil, err, 0)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: browserSessionCookie, Value: session.Token, Path: "/", HttpOnly: true, Secure: h.config.SecureCookies, SameSite: http.SameSiteLaxMode, Expires: session.ExpiresAt})
	writeJSON(w, SessionView{AccountID: session.AccountID, WalletAddress: session.WalletAddress, ExpiresAt: session.ExpiresAt}, http.StatusOK)
}

func (h *httpHandler) session(w http.ResponseWriter, r *http.Request) {
	session, ok := h.authenticate(w, r)
	if ok {
		writeJSON(w, SessionView{AccountID: session.AccountID, WalletAddress: session.WalletAddress, ExpiresAt: session.ExpiresAt}, http.StatusOK)
	}
}

func (h *httpHandler) deviceCreate(w http.ResponseWriter, r *http.Request) {
	var input struct {
		MachineName   string `json:"machine_name"`
		SignerAddress string `json:"signer_address"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	authz, err := h.service.CreateDeviceAuthorization(r.Context(), input.MachineName, input.SignerAddress, 10*time.Minute)
	h.write(w, DeviceHTTPAuthorization{DeviceCode: authz.DeviceCode, UserCode: authz.UserCode, VerificationURI: h.config.VerificationURL, ExpiresAt: authz.ExpiresAt, SignerAddress: authz.SignerAddress, ChainID: h.config.ChainID, ContractAddress: h.config.ContractAddress}, err, http.StatusCreated)
}

func (h *httpHandler) deviceToken(w http.ResponseWriter, r *http.Request) {
	var input struct {
		DeviceCode string `json:"device_code"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	machine, token, err := h.service.ExchangeDeviceAuthorization(r.Context(), input.DeviceCode)
	h.write(w, DeviceToken{Machine: machine, Token: token}, err, http.StatusOK)
}

func (h *httpHandler) deviceInspect(w http.ResponseWriter, r *http.Request, _ BrowserSession) {
	var input struct {
		UserCode string `json:"user_code"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	pending, err := h.service.PendingDevice(r.Context(), input.UserCode)
	h.write(w, pending, err, http.StatusOK)
}

func (h *httpHandler) deviceApprove(w http.ResponseWriter, r *http.Request, session BrowserSession) {
	var input struct {
		UserCode string `json:"user_code"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	err := h.service.ApproveDeviceAuthorization(r.Context(), input.UserCode, session.AccountID)
	h.write(w, map[string]any{}, err, http.StatusNoContent)
}

func (h *httpHandler) apiKeysList(w http.ResponseWriter, r *http.Request, session BrowserSession) {
	keys, err := h.service.ListAPIKeys(r.Context(), session.AccountID)
	views := make([]APIKeyView, len(keys))
	for i, key := range keys {
		views[i] = APIKeyView{ID: key.ID, Scope: key.Scope, CreatedAt: key.CreatedAt}
	}
	h.write(w, views, err, http.StatusOK)
}

func (h *httpHandler) apiKeysCreate(w http.ResponseWriter, r *http.Request, session BrowserSession) {
	var scope APIKeyScope
	if !decodeJSON(w, r, &scope) {
		return
	}
	key, err := h.service.CreateAPIKey(r.Context(), session.AccountID, scope)
	h.write(w, APIKeyView{ID: key.ID, Token: key.Token, Scope: scope}, err, http.StatusCreated)
}

func (h *httpHandler) apiKeysDelete(w http.ResponseWriter, r *http.Request, session BrowserSession) {
	id := r.PathValue("id")
	if id == "" {
		http.Error(w, "missing API key", http.StatusBadRequest)
		return
	}
	keys, err := h.service.ListAPIKeys(r.Context(), session.AccountID)
	if err == nil {
		found := false
		for _, key := range keys {
			found = found || key.ID == id
		}
		if !found {
			err = ErrInvalidCredential
		}
	}
	if err == nil {
		err = h.service.RevokeAPIKey(r.Context(), id)
	}
	h.write(w, map[string]any{}, err, http.StatusNoContent)
}

func (h *httpHandler) withSession(next func(http.ResponseWriter, *http.Request, BrowserSession)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := h.browserOrigin(w, r); !ok {
			return
		}
		session, ok := h.authenticate(w, r)
		if ok {
			next(w, r, session)
		}
	}
}

func (h *httpHandler) authenticate(w http.ResponseWriter, r *http.Request) (BrowserSession, bool) {
	cookie, err := r.Cookie(browserSessionCookie)
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return BrowserSession{}, false
	}
	session, err := h.service.AuthenticateBrowserSession(r.Context(), cookie.Value)
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return BrowserSession{}, false
	}
	return session, true
}

func (s *Service) AuthenticateBrowserRequest(r *http.Request) (BrowserSession, error) {
	cookie, err := r.Cookie(browserSessionCookie)
	if err != nil {
		return BrowserSession{}, ErrInvalidCredential
	}
	return s.AuthenticateBrowserSession(r.Context(), cookie.Value)
}

func (h *httpHandler) browserOrigin(w http.ResponseWriter, r *http.Request) (string, bool) {
	origin := r.Header.Get("Origin")
	for _, allowed := range h.config.AllowedOrigins {
		if origin == allowed {
			return origin, true
		}
	}
	http.Error(w, "origin denied", http.StatusForbidden)
	return "", false
}

func (h *httpHandler) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		for _, allowed := range h.config.AllowedOrigins {
			if origin == allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", "Origin")
				break
			}
		}
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *httpHandler) write(w http.ResponseWriter, value any, err error, success int) {
	if err == nil {
		if success == http.StatusNoContent {
			w.WriteHeader(success)
			return
		}
		writeJSON(w, value, success)
		return
	}
	status := http.StatusBadRequest
	switch {
	case errors.Is(err, ErrAuthorizationPending):
		status = http.StatusTooEarly
	case errors.Is(err, ErrAuthorizationExpired):
		status = http.StatusGone
	case errors.Is(err, ErrAuthorizationConsumed):
		status = http.StatusConflict
	case errors.Is(err, ErrInvalidCredential):
		status = http.StatusUnauthorized
	case errors.Is(err, ErrOriginDenied):
		status = http.StatusForbidden
	}
	http.Error(w, err.Error(), status)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "application/json required", http.StatusUnsupportedMediaType)
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, value any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
