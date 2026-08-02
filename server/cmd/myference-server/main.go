package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	v1 "github.com/kunalshah017/myference/protocol/v1"
	"github.com/kunalshah017/myference/server/internal/api"
	"github.com/kunalshah017/myference/server/internal/auth"
	"github.com/kunalshah017/myference/server/internal/realtime"
	"github.com/kunalshah017/myference/server/internal/relay"
	"github.com/kunalshah017/myference/server/internal/router"
	"github.com/kunalshah017/myference/server/internal/store"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
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
			return api.Principal{AccountID: principal.AccountID, SessionID: sessionID, SessionBalance: balance}, err
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
	marketplace := api.NewMarketplace(repository, func(request *http.Request) (string, error) {
		session, err := authService.AuthenticateBrowserRequest(request)
		return session.AccountID, err
	}, 30*time.Second)
	operations := api.NewOperations(repository, func(request *http.Request) (string, error) {
		session, err := authService.AuthenticateBrowserRequest(request)
		return session.AccountID, err
	}, api.OperationsConfig{ChainID: 10143, ContractAddress: chainConfiguration.ContractAddress, ExplorerURL: envOr("MYFERENCE_EXPLORER_URL", "https://testnet.monadexplorer.com"), Confirmations: 2})
	analytics := api.NewAnalytics(repository, func(request *http.Request) (string, error) {
		session, err := authService.AuthenticateBrowserRequest(request)
		return session.AccountID, err
	}, api.AnalyticsConfig{ChainID: 10143, ContractAddress: chainConfiguration.ContractAddress})
	events, err := realtime.Open(ctx, databaseURL, func(ctx context.Context, ticket string) (string, error) {
		return authService.ConsumeStreamTicket(ctx, ticket)
	})
	if err != nil {
		return err
	}
	defer events.Close()
	handler := allowWebOrigin(newRootHandler(hub, openAI, anthropic, authHTTP, marketplace, operations, analytics, events), webOrigin)
	server := &http.Server{Addr: envOr("MYFERENCE_LISTEN_ADDR", "127.0.0.1:8080"), Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}

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

func newRootHandler(relayHandler, openAIHandler, anthropicHandler, authHandler, marketplaceHandler, operationsHandler, analyticsHandler, eventsHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "ok\n") })
	mux.Handle("/relay", relayHandler)
	mux.Handle("/v1/chat/completions", openAIHandler)
	mux.Handle("/v1/messages", anthropicHandler)
	mux.Handle("/auth/", authHandler)
	mux.Handle("/api/account/operations", operationsHandler)
	mux.Handle("/api/account/analytics", analyticsHandler)
	mux.Handle("/api/", marketplaceHandler)
	mux.Handle("/events", eventsHandler)
	return mux
}

func allowWebOrigin(next http.Handler, origin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") == origin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
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
