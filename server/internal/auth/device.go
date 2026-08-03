package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	ErrAuthorizationPending  = errors.New("device authorization pending")
	ErrAuthorizationExpired  = errors.New("device authorization expired")
	ErrAuthorizationConsumed = errors.New("device authorization already consumed")
	ErrInvalidCredential     = errors.New("invalid credential")
	ErrScopeDenied           = errors.New("API key scope denied")
)

type Service struct {
	db  *sql.DB
	now func() time.Time
}

type DeviceAuthorization struct {
	DeviceCode, UserCode, AccountID, SignerAddress string
	ExpiresAt                                      time.Time
}

type Machine struct {
	ID            string `json:"id"`
	AccountID     string `json:"account_id"`
	Name          string `json:"name"`
	SignerAddress string `json:"signer_address"`
}
type MachinePrincipal struct{ MachineID, AccountID string }

type APIKeyScope struct {
	Models      []string `json:"models"`
	Endpoints   []string `json:"endpoints"`
	MaxSpendWei uint64   `json:"max_spend_wei,string"`
}

type APIKey struct{ ID, Token string }
type APIKeyPrincipal struct {
	KeyID, AccountID string
	Scope            APIKeyScope
}

func Open(ctx context.Context, databaseURL string) (*Service, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return &Service{db: db, now: time.Now}, nil
}

func (s *Service) Close() error { return s.db.Close() }

func (s *Service) CreateDeviceAuthorization(ctx context.Context, machineName, signerAddress string, lifetime time.Duration) (DeviceAuthorization, error) {
	if strings.TrimSpace(machineName) == "" || !common.IsHexAddress(signerAddress) || common.HexToAddress(signerAddress) == (common.Address{}) || lifetime <= 0 {
		return DeviceAuthorization{}, errors.New("machine name, signer address, and positive lifetime are required")
	}
	signerAddress = common.HexToAddress(signerAddress).Hex()
	deviceCode, err := randomToken(32)
	if err != nil {
		return DeviceAuthorization{}, err
	}
	userBytes := make([]byte, 4)
	if _, err := rand.Read(userBytes); err != nil {
		return DeviceAuthorization{}, err
	}
	userCode := strings.ToUpper(hex.EncodeToString(userBytes))
	expiresAt := s.now().UTC().Truncate(time.Microsecond).Add(lifetime)
	_, err = s.db.ExecContext(ctx, `INSERT INTO device_authorizations (device_code_hash, user_code_hash, machine_name, signer_address, expires_at) VALUES ($1, $2, $3, $4, $5)`, digest(deviceCode), digest(userCode), machineName, signerAddress, expiresAt)
	return DeviceAuthorization{DeviceCode: deviceCode, UserCode: userCode, SignerAddress: signerAddress, ExpiresAt: expiresAt}, err
}

func (s *Service) PollDeviceAuthorization(ctx context.Context, deviceCode string) (DeviceAuthorization, error) {
	var result DeviceAuthorization
	var accountID sql.NullString
	var approvedAt, exchangedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `SELECT account_id, expires_at, approved_at, exchanged_at FROM device_authorizations WHERE device_code_hash = $1`, digest(deviceCode)).Scan(&accountID, &result.ExpiresAt, &approvedAt, &exchangedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return DeviceAuthorization{}, ErrInvalidCredential
	}
	if err != nil {
		return DeviceAuthorization{}, err
	}
	if !s.now().Before(result.ExpiresAt) {
		return DeviceAuthorization{}, ErrAuthorizationExpired
	}
	if exchangedAt.Valid {
		return DeviceAuthorization{}, ErrAuthorizationConsumed
	}
	if !approvedAt.Valid || !accountID.Valid {
		return DeviceAuthorization{}, ErrAuthorizationPending
	}
	result.AccountID = accountID.String
	return result, nil
}

func (s *Service) ApproveDeviceAuthorization(ctx context.Context, userCode, accountID string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE device_authorizations SET account_id = $2, approved_at = $3 WHERE user_code_hash = $1 AND approved_at IS NULL AND exchanged_at IS NULL AND expires_at > $3`, digest(strings.ToUpper(userCode)), accountID, s.now())
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return ErrInvalidCredential
	}
	return nil
}

func (s *Service) ExchangeDeviceAuthorization(ctx context.Context, deviceCode string) (Machine, string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Machine{}, "", err
	}
	defer tx.Rollback()
	var machine Machine
	var accountID sql.NullString
	var expiresAt time.Time
	var exchangedAt sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT account_id, machine_name, signer_address, expires_at, exchanged_at FROM device_authorizations WHERE device_code_hash = $1 FOR UPDATE`, digest(deviceCode)).Scan(&accountID, &machine.Name, &machine.SignerAddress, &expiresAt, &exchangedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Machine{}, "", ErrInvalidCredential
	}
	if err != nil {
		return Machine{}, "", err
	}
	if exchangedAt.Valid {
		return Machine{}, "", ErrAuthorizationConsumed
	}
	if !s.now().Before(expiresAt) {
		return Machine{}, "", ErrAuthorizationExpired
	}
	if !accountID.Valid {
		return Machine{}, "", ErrAuthorizationPending
	}
	machine.AccountID = accountID.String
	machine.ID, err = randomID("mach")
	if err != nil {
		return Machine{}, "", err
	}
	secret, err := randomToken(32)
	if err != nil {
		return Machine{}, "", err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO machines (id, account_id, name, signer_address) VALUES ($1, $2, $3, $4)`, machine.ID, machine.AccountID, machine.Name, machine.SignerAddress); err != nil {
		return Machine{}, "", err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO machine_tokens (machine_id, token_hash) VALUES ($1, $2)`, machine.ID, digest(secret)); err != nil {
		return Machine{}, "", err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE device_authorizations SET exchanged_at = $2 WHERE device_code_hash = $1`, digest(deviceCode), s.now()); err != nil {
		return Machine{}, "", err
	}
	if err = tx.Commit(); err != nil {
		return Machine{}, "", err
	}
	return machine, machine.ID + "." + secret, nil
}

func (s *Service) AuthenticateMachine(ctx context.Context, token string) (MachinePrincipal, error) {
	id, secret, ok := splitCredential(token)
	if !ok {
		return MachinePrincipal{}, ErrInvalidCredential
	}
	var principal MachinePrincipal
	var stored []byte
	err := s.db.QueryRowContext(ctx, `SELECT m.id, m.account_id, mt.token_hash FROM machine_tokens mt JOIN machines m ON m.id = mt.machine_id WHERE mt.machine_id = $1 AND mt.revoked_at IS NULL AND m.revoked_at IS NULL`, id).Scan(&principal.MachineID, &principal.AccountID, &stored)
	if err != nil || subtle.ConstantTimeCompare(stored, digest(secret)) != 1 {
		return MachinePrincipal{}, ErrInvalidCredential
	}
	return principal, nil
}

func (s *Service) RevokeMachine(ctx context.Context, machineID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE machines SET revoked_at = $2 WHERE id = $1`, machineID, s.now())
	return err
}

func (s *Service) CreateAPIKey(ctx context.Context, accountID string, scope APIKeyScope) (APIKey, error) {
	if len(scope.Endpoints) == 0 || scope.MaxSpendWei == 0 {
		return APIKey{}, errors.New("endpoint and spend scopes are required")
	}
	id, err := randomID("key")
	if err != nil {
		return APIKey{}, err
	}
	secret, err := randomToken(32)
	if err != nil {
		return APIKey{}, err
	}
	scopeJSON, _ := json.Marshal(scope)
	_, err = s.db.ExecContext(ctx, `INSERT INTO api_keys (id, account_id, key_hash, scope_json) VALUES ($1, $2, $3, $4)`, id, accountID, digest(secret), scopeJSON)
	return APIKey{ID: id, Token: id + "." + secret}, err
}

func (s *Service) AuthorizeAPIKey(ctx context.Context, token, model, endpoint string, spend uint64) (APIKeyPrincipal, error) {
	id, secret, ok := splitCredential(token)
	if !ok {
		return APIKeyPrincipal{}, ErrInvalidCredential
	}
	var principal APIKeyPrincipal
	var stored, scopeJSON []byte
	err := s.db.QueryRowContext(ctx, `SELECT id, account_id, key_hash, scope_json FROM api_keys WHERE id = $1 AND revoked_at IS NULL`, id).Scan(&principal.KeyID, &principal.AccountID, &stored, &scopeJSON)
	if err != nil || subtle.ConstantTimeCompare(stored, digest(secret)) != 1 {
		return APIKeyPrincipal{}, ErrInvalidCredential
	}
	if err := json.Unmarshal(scopeJSON, &principal.Scope); err != nil {
		return APIKeyPrincipal{}, fmt.Errorf("decode API key scope: %w", err)
	}
	if !allowsModel(principal.Scope.Models, model) || !contains(principal.Scope.Endpoints, endpoint) || spend > principal.Scope.MaxSpendWei {
		return APIKeyPrincipal{}, ErrScopeDenied
	}
	return principal, nil
}

func allowsModel(models []string, model string) bool {
	return len(models) == 0 || contains(models, model)
}

func (s *Service) RevokeAPIKey(ctx context.Context, keyID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE api_keys SET revoked_at = $2 WHERE id = $1`, keyID, s.now())
	return err
}

func randomID(prefix string) (string, error) {
	token, err := randomToken(12)
	return prefix + "_" + token, err
}

func randomToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func digest(value string) []byte {
	sum := sha256.Sum256([]byte(value))
	return sum[:]
}

func splitCredential(token string) (string, string, bool) {
	id, secret, ok := strings.Cut(token, ".")
	return id, secret, ok && id != "" && secret != ""
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
