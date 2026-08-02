package main

import (
	"context"
	"errors"
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
		return repository.UpdateProviderCapacity(ctx, machineID, capacity)
	}})
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
			return repository.ReserveInference(ctx, store.InferenceReservation{RequestID: reservation.RequestID, SessionID: reservation.SessionID, AccountID: reservation.AccountID, MachineID: reservation.MachineID, OfferID: reservation.OfferID, PriceVersion: reservation.PriceVersion, MaximumSpend: reservation.MaximumSpend})
		},
		Transition: repository.TransitionRequest,
		Abort:      repository.AbortInference,
		Persist: func(ctx context.Context, proposal api.Proposal) error {
			return repository.CompleteInference(ctx, store.ReceiptProposal{RequestID: proposal.RequestID, SessionID: proposal.SessionID, MachineID: proposal.MachineID, OfferID: proposal.OfferID, Model: proposal.Model, PriceVersion: proposal.PriceVersion, InputTokens: proposal.InputTokens, OutputTokens: proposal.OutputTokens, ComputeMilliseconds: proposal.ComputeMilliseconds, InputHash: proposal.InputHash, OutputHash: proposal.OutputHash, CompletedAt: proposal.CompletedAt})
		},
	})
	webOrigin := envOr("MYFERENCE_WEB_ORIGIN", "http://127.0.0.1:5173")
	authHTTP := auth.NewHandler(authService, auth.HTTPConfig{
		Domain:          envOr("MYFERENCE_AUTH_DOMAIN", "localhost"),
		AllowedOrigins:  []string{webOrigin},
		ChainID:         10143,
		SessionLifetime: 12 * time.Hour,
		SecureCookies:   strings.HasPrefix(webOrigin, "https://"),
		VerificationURL: strings.TrimSuffix(webOrigin, "/") + "/devices",
	})
	server := &http.Server{Addr: envOr("MYFERENCE_LISTEN_ADDR", "127.0.0.1:8080"), Handler: newRootHandler(hub, openAI, authHTTP), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}

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

func newRootHandler(relayHandler, openAIHandler, authHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/relay", relayHandler)
	mux.Handle("/v1/chat/completions", openAIHandler)
	mux.Handle("/auth/", authHandler)
	return mux
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
