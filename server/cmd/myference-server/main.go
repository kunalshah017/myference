package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	v1 "github.com/kunalshah017/myference/protocol/v1"
	"github.com/kunalshah017/myference/server/internal/api"
	"github.com/kunalshah017/myference/server/internal/auth"
	"github.com/kunalshah017/myference/server/internal/ratelimit"
	"github.com/kunalshah017/myference/server/internal/realtime"
	"github.com/kunalshah017/myference/server/internal/relay"
	"github.com/kunalshah017/myference/server/internal/router"
	"github.com/kunalshah017/myference/server/internal/store"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	if err := run(); err != nil {
		slog.Error("server exited", "error", err)
		os.Exit(1)
	}
}

func run() error {
	databaseURL := os.Getenv("MYFERENCE_DATABASE_URL")
	if databaseURL == "" {
		return errors.New("MYFERENCE_DATABASE_URL is required")
	}
	chainConfiguration, err := loadChainConfig(os.Getenv)
	if err != nil {
		return err
	}
	ctx := context.Background()
	authService, err := auth.Open(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer authService.Close()
	repository, err := store.Open(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer repository.Close()

	hub := relay.NewHub(func(ctx context.Context, token string) (string, error) {
		principal, err := authService.AuthenticateMachine(ctx, token)
		return principal.MachineID, err
	}, relay.Options{CapacityHandler: func(machineID string, capacity v1.Capacity) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return repository.ReconcileProviderCapacity(ctx, machineID, capacity, 10143, chainConfiguration.ContractAddress)
	}})
	chainState, err := openChainRuntime(ctx, chainConfiguration, databaseURL, repository, hub)
	if err != nil {
		return err
	}
	defer chainState.Close()
	runtimeContext, stopRuntime := context.WithCancel(ctx)
	defer stopRuntime()
	chainState.Run(runtimeContext)
	openAI := api.NewOpenAI(api.Dependencies{
		Hub: hub,
		Authorize: func(ctx context.Context, token, model, endpoint string, maximum uint64) (api.Principal, error) {
			principal, err := authService.AuthorizeAPIKey(ctx, token, model, endpoint, maximum)
			if err != nil {
				return api.Principal{}, err
			}
			sessionID, balance, err := repository.OpenSession(ctx, principal.AccountID)
			return api.Principal{AccountID: principal.AccountID, KeyID: principal.KeyID, SessionID: sessionID, SessionBalance: balance}, err
		},
		Candidates: func(ctx context.Context, model string) ([]router.Candidate, error) {
			return repository.RoutingCandidates(ctx, model)
		},
		Reserve: func(ctx context.Context, reservation api.Reservation) error {
			return repository.ReserveInference(ctx, store.InferenceReservation{RequestID: reservation.RequestID, SessionID: reservation.SessionID, AccountID: reservation.AccountID, MachineID: reservation.MachineID, OfferID: reservation.OfferID, PriceVersion: reservation.PriceVersion, MaximumSpend: reservation.MaximumSpend, MaximumInputTokens: reservation.MaximumInputTokens, MaximumOutputTokens: reservation.MaximumOutputTokens, MaximumComputeMilliseconds: reservation.MaximumComputeMilliseconds})
		},
		Transition: repository.TransitionRequest,
		Abort:      repository.AbortInference,
		Persist: func(ctx context.Context, proposal api.Proposal) error {
			return chainState.coordinator.Complete(ctx, store.ReceiptProposal{RequestID: proposal.RequestID, SessionID: proposal.SessionID, MachineID: proposal.MachineID, OfferID: proposal.OfferID, Model: proposal.Model, PriceVersion: proposal.PriceVersion, InputTokens: proposal.InputTokens, OutputTokens: proposal.OutputTokens, ComputeMilliseconds: proposal.ComputeMilliseconds, InputHash: proposal.InputHash, OutputHash: proposal.OutputHash, CompletedAt: proposal.CompletedAt})
		},
		RateLimiter: ratelimit.New(3, 10),
		Concurrency: ratelimit.NewConcurrency(4),
	})
	anthropic := api.NewAnthropic(openAI)
	webOrigin := envOr("MYFERENCE_WEB_ORIGIN", "http://127.0.0.1:5173")
	authHTTP := auth.NewHandler(authService, auth.HTTPConfig{
		Domain:          envOr("MYFERENCE_AUTH_DOMAIN", "localhost"),
		AllowedOrigins:  []string{webOrigin},
		ChainID:         10143,
		SessionLifetime: 12 * time.Hour,
		SecureCookies:   strings.HasPrefix(webOrigin, "https://"),
		VerificationURL: strings.TrimSuffix(webOrigin, "/") + "/devices",
		ContractAddress: chainConfiguration.ContractAddress,
	})
	authHTTP = rateLimitByIP(authHTTP, ratelimit.New(1, 60))
	marketplace := api.NewMarketplace(repository, func(request *http.Request) (string, error) {
		session, err := authService.AuthenticateBrowserRequest(request)
		return session.AccountID, err
	}, 30*time.Second)
	browserAccount := func(request *http.Request) (string, error) {
		session, err := authService.AuthenticateBrowserRequest(request)
		return session.AccountID, err
	}
	machineAccount := func(request *http.Request) (string, string, error) {
		token := strings.TrimSpace(strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer "))
		principal, err := authService.AuthenticateMachine(request.Context(), token)
		return principal.MachineID, principal.AccountID, err
	}
	providerConfig := store.ProviderAccountConfig{
		ChainID:         10143,
		ContractAddress: chainConfiguration.ContractAddress,
		ExplorerURL:     envOr("MYFERENCE_EXPLORER_URL", "https://testnet.monadexplorer.com"),
		Confirmations:   2,
		MinimumBondWei:  chainState.terms.MinimumBond.String(),
	}
	providerAccount := api.NewProviderAccount(repository, machineAccount, browserAccount, providerConfig)
	providerActions := api.NewProviderActions(api.NewProviderActionStore(time.Now), api.ProviderActionDependencies{
		MachineAuth: machineAccount,
		AccountAuth: browserAccount,
		Prepare: func(ctx context.Context, source, machineID, accountID string, input api.ProviderActionInput) (string, api.ProviderActionBaseline, error) {
			return prepareProviderAction(ctx, repository, providerConfig, source, machineID, accountID, input)
		},
		Verify: func(ctx context.Context, action api.ProviderAction) (map[string]uint64, bool, error) {
			return verifyProviderAction(ctx, repository, providerConfig, action)
		},
	})
	operations := api.NewOperations(repository, func(request *http.Request) (string, error) {
		session, err := authService.AuthenticateBrowserRequest(request)
		return session.AccountID, err
	}, api.OperationsConfig{ChainID: 10143, ContractAddress: chainConfiguration.ContractAddress, ExplorerURL: envOr("MYFERENCE_EXPLORER_URL", "https://testnet.monadexplorer.com"), Confirmations: 2})
	analytics := api.NewAnalytics(repository, func(request *http.Request) (string, error) {
		session, err := authService.AuthenticateBrowserRequest(request)
		return session.AccountID, err
	}, api.AnalyticsConfig{ChainID: 10143, ContractAddress: chainConfiguration.ContractAddress})
	referencePrice := api.NewReferencePrice(api.ReferencePriceConfig{})
	events, err := realtime.Open(ctx, databaseURL, func(ctx context.Context, ticket string) (string, error) {
		return authService.ConsumeStreamTicket(ctx, ticket)
	})
	if err != nil {
		return err
	}
	defer events.Close()
	handler := allowWebOrigin(rateLimitByIP(newRootHandler(hub, openAI, anthropic, authHTTP, marketplace, operations, analytics, referencePrice, providerAccount, providerActions, events), ratelimit.New(20, 200)), webOrigin)
	server := &http.Server{Addr: listenAddress(os.Getenv), Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}

	certificate, key := os.Getenv("MYFERENCE_TLS_CERT"), os.Getenv("MYFERENCE_TLS_KEY")
	if (certificate == "") != (key == "") {
		return errors.New("MYFERENCE_TLS_CERT and MYFERENCE_TLS_KEY must be configured together")
	}
	serveErrors := make(chan error, 1)
	go func() {
		if certificate != "" && key != "" {
			serveErrors <- server.ListenAndServeTLS(certificate, key)
			return
		}
		serveErrors <- server.ListenAndServe()
	}()
	stop, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()
	select {
	case err := <-serveErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-stop.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}

func newRootHandler(relayHandler, openAIHandler, anthropicHandler, authHandler, marketplaceHandler, operationsHandler, analyticsHandler, referencePriceHandler, providerAccountHandler, providerActionsHandler, eventsHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "ok\n") })
	mux.Handle("/relay", relayHandler)
	mux.Handle("/v1/chat/completions", openAIHandler)
	mux.Handle("/v1/messages", anthropicHandler)
	mux.Handle("/auth/", authHandler)
	mux.Handle("/api/account/operations", operationsHandler)
	mux.Handle("/api/account/analytics", analyticsHandler)
	mux.Handle("/api/reference-price", referencePriceHandler)
	mux.Handle("/api/provider/account", providerAccountHandler)
	mux.Handle("/api/provider/machines/", providerAccountHandler)
	mux.Handle("/api/provider/actions", providerActionsHandler)
	mux.Handle("/api/provider/actions/", providerActionsHandler)
	mux.Handle("/api/", marketplaceHandler)
	mux.Handle("/events", eventsHandler)
	return mux
}

func rateLimitByIP(next http.Handler, limiter *ratelimit.Limiter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !limiter.Allow(clientIP(r)) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		return forwarded
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func allowWebOrigin(next http.Handler, origin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") == origin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Myference-Max-Spend")
			w.Header().Add("Vary", "Origin")
			w.Header().Add("Vary", "Access-Control-Request-Method")
			w.Header().Add("Vary", "Access-Control-Request-Headers")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func listenAddress(getenv func(string) string) string {
	if address := strings.TrimSpace(getenv("MYFERENCE_LISTEN_ADDR")); address != "" {
		return address
	}
	if port := strings.TrimSpace(getenv("PORT")); port != "" {
		return "0.0.0.0:" + port
	}
	return "127.0.0.1:8080"
}
