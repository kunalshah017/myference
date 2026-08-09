package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/kunalshah017/myference/server/internal/store"
)

func TestDeviceAuthorizationIsExpiringOneTimeAndRevocable(t *testing.T) {
	ctx, service, accountID := newIntegrationService(t)
	signer := "0x0000000000000000000000000000000000001234"
	authz, err := service.CreateDeviceAuthorization(ctx, "windows-worker", signer, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.PollDeviceAuthorization(ctx, authz.DeviceCode); !errors.Is(err, ErrAuthorizationPending) {
		t.Fatalf("expected pending, got %v", err)
	}
	if err := service.ApproveDeviceAuthorization(ctx, authz.UserCode, accountID); err != nil {
		t.Fatal(err)
	}
	approved, err := service.PollDeviceAuthorization(ctx, authz.DeviceCode)
	if err != nil || approved.AccountID != accountID {
		t.Fatalf("approved=%+v err=%v", approved, err)
	}

	machine, token, err := service.ExchangeDeviceAuthorization(ctx, authz.DeviceCode)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || machine.AccountID != accountID || machine.Name != "windows-worker" || machine.SignerAddress != signer {
		t.Fatalf("machine=%+v tokenEmpty=%v", machine, token == "")
	}
	if _, _, err := service.ExchangeDeviceAuthorization(ctx, authz.DeviceCode); !errors.Is(err, ErrAuthorizationConsumed) {
		t.Fatalf("expected consumed, got %v", err)
	}
	principal, err := service.AuthenticateMachine(ctx, token)
	if err != nil || principal.MachineID != machine.ID {
		t.Fatalf("principal=%+v err=%v", principal, err)
	}
	if err := service.RevokeMachine(ctx, machine.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AuthenticateMachine(ctx, token); !errors.Is(err, ErrInvalidCredential) {
		t.Fatalf("expected revoked token rejection, got %v", err)
	}

	expired, err := service.CreateDeviceAuthorization(ctx, "expired-worker", "0x000000000000000000000000000000000000bEEF", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return time.Now().Add(2 * time.Minute) }
	if _, err := service.PollDeviceAuthorization(ctx, expired.DeviceCode); !errors.Is(err, ErrAuthorizationExpired) {
		t.Fatalf("expected expiry, got %v", err)
	}
}

func TestDeviceAuthorizationReclaimsAccountMachineName(t *testing.T) {
	ctx, service, accountID := newIntegrationService(t)
	signerBase := time.Now().UnixNano()
	firstSigner := fmt.Sprintf("0x%040x", signerBase)
	secondSigner := fmt.Sprintf("0x%040x", signerBase+1)
	exchange := func(signer string) (Machine, string, error) {
		authorization, err := service.CreateDeviceAuthorization(ctx, "MacBook-Air.local", signer, 5*time.Minute)
		if err != nil {
			return Machine{}, "", err
		}
		if err := service.ApproveDeviceAuthorization(ctx, authorization.UserCode, accountID); err != nil {
			return Machine{}, "", err
		}
		return service.ExchangeDeviceAuthorization(ctx, authorization.DeviceCode)
	}

	first, oldToken, err := exchange(firstSigner)
	if err != nil {
		t.Fatal(err)
	}
	second, newToken, err := exchange(secondSigner)
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID || second.AccountID != accountID || second.Name != first.Name || second.SignerAddress != common.HexToAddress(secondSigner).Hex() || newToken == oldToken {
		t.Fatalf("first=%+v second=%+v tokenRotated=%v", first, second, newToken != oldToken)
	}
	if principal, err := service.AuthenticateMachine(ctx, newToken); err != nil || principal.MachineID != first.ID {
		t.Fatalf("new principal=%+v err=%v", principal, err)
	}
	if _, err := service.AuthenticateMachine(ctx, oldToken); !errors.Is(err, ErrInvalidCredential) {
		t.Fatalf("old token error=%v", err)
	}
	var count int
	if err := service.db.QueryRowContext(ctx, `SELECT count(*) FROM machines WHERE account_id=$1 AND name=$2`, accountID, first.Name).Scan(&count); err != nil || count != 1 {
		t.Fatalf("machine count=%d err=%v", count, err)
	}
}

func TestAPIKeysEnforceModelEndpointAndSpendScopes(t *testing.T) {
	ctx, service, accountID := newIntegrationService(t)
	key, err := service.CreateAPIKey(ctx, accountID, APIKeyScope{
		Models:      []string{"qwen3:8b"},
		Endpoints:   []string{"/v1/chat/completions"},
		MaxSpendWei: 1_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	principal, err := service.AuthorizeAPIKey(ctx, key.Token, "qwen3:8b", "/v1/chat/completions", 1_000)
	if err != nil || principal.AccountID != accountID {
		t.Fatalf("principal=%+v err=%v", principal, err)
	}
	for _, tc := range []struct {
		model, endpoint string
		spend           uint64
	}{
		{"other-model", "/v1/chat/completions", 1_000},
		{"qwen3:8b", "/v1/responses", 1_000},
		{"qwen3:8b", "/v1/chat/completions", 1_001},
	} {
		if _, err := service.AuthorizeAPIKey(ctx, key.Token, tc.model, tc.endpoint, tc.spend); !errors.Is(err, ErrScopeDenied) {
			t.Fatalf("expected scope denial for %+v, got %v", tc, err)
		}
	}
	if err := service.RevokeAPIKey(ctx, key.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AuthorizeAPIKey(ctx, key.Token, "qwen3:8b", "/v1/chat/completions", 1); !errors.Is(err, ErrInvalidCredential) {
		t.Fatalf("expected revoked key rejection, got %v", err)
	}
}

func TestEmptyModelScopeAllowsEveryModel(t *testing.T) {
	for _, model := range []string{"qwen3:8b", "claude-sonnet", "a-model-published-later"} {
		if !allowsModel(nil, model) {
			t.Fatalf("empty model scope must allow %q", model)
		}
	}
	if allowsModel([]string{"qwen3:8b"}, "claude-sonnet") {
		t.Fatal("non-empty model scope must remain an exact allowlist")
	}
}

func TestAPIKeysMayBeCreatedWithoutAModelRestriction(t *testing.T) {
	ctx, service, accountID := newIntegrationService(t)
	key, err := service.CreateAPIKey(ctx, accountID, APIKeyScope{
		Models:      []string{},
		Endpoints:   []string{"/v1/chat/completions", "/v1/messages"},
		MaxSpendWei: 1_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, model := range []string{"qwen3:8b", "claude-sonnet"} {
		if _, err := service.AuthorizeAPIKey(ctx, key.Token, model, "/v1/chat/completions", 1_000); err != nil {
			t.Fatalf("unrestricted key denied %q: %v", model, err)
		}
	}
}

func newIntegrationService(t *testing.T) (context.Context, *Service, string) {
	t.Helper()
	databaseURL := os.Getenv("MYFERENCE_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("MYFERENCE_TEST_DATABASE_URL is required for PostgreSQL integration")
	}
	ctx := context.Background()
	control, err := store.Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { control.Close() })
	if err := control.ApplyMigration(ctx, filepath.Join("..", "..", "..", "migrations", "000001_control_plane.sql")); err != nil {
		t.Fatal(err)
	}
	if err := control.ApplyMigration(ctx, filepath.Join("..", "..", "..", "migrations", "000008_machine_signers.sql")); err != nil {
		t.Fatal(err)
	}
	accountID := "acct-" + time.Now().Format("150405.000000000")
	if err := control.CreateAccount(ctx, store.Account{ID: accountID, WalletAddress: "0x" + time.Now().Format("20060102150405.000000000")}); err != nil {
		t.Fatal(err)
	}
	service, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { service.Close() })
	return ctx, service, accountID
}
