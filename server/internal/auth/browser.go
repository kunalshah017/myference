package auth

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var ErrOriginDenied = errors.New("origin denied")

type WalletChallenge struct {
	ID             string    `json:"id"`
	Address        string    `json:"address"`
	Nonce          string    `json:"nonce"`
	Message        string    `json:"message"`
	ChainID        uint64    `json:"chain_id"`
	IssuedAt       time.Time `json:"issued_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	Origin, Domain string
}

type BrowserSession struct {
	ID, Token, AccountID, WalletAddress string
	ExpiresAt                           time.Time
}

type PendingDevice struct {
	MachineName string    `json:"machine_name"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type APIKeyRecord struct {
	ID        string      `json:"id"`
	Scope     APIKeyScope `json:"scope"`
	CreatedAt time.Time   `json:"created_at"`
}

func (s *Service) CreateWalletChallenge(ctx context.Context, domain, origin, address string, chainID uint64, lifetime time.Duration) (WalletChallenge, error) {
	if domain == "" || origin == "" || !common.IsHexAddress(address) || chainID == 0 || lifetime <= 0 {
		return WalletChallenge{}, errors.New("domain, origin, wallet, chain, and positive lifetime are required")
	}
	id, err := randomID("challenge")
	if err != nil {
		return WalletChallenge{}, err
	}
	nonce, err := randomToken(18)
	if err != nil {
		return WalletChallenge{}, err
	}
	issued := s.now().UTC().Truncate(time.Microsecond)
	challenge := WalletChallenge{
		ID: id, Address: common.HexToAddress(address).Hex(), Nonce: nonce, ChainID: chainID,
		IssuedAt: issued, ExpiresAt: issued.Add(lifetime), Origin: origin, Domain: domain,
	}
	challenge.Message = fmt.Sprintf("Myference account login\nDomain: %s\nOrigin: %s\nAddress: %s\nChain ID: %d\nNonce: %s\nIssued At: %s\nExpiration Time: %s", challenge.Domain, challenge.Origin, challenge.Address, challenge.ChainID, challenge.Nonce, challenge.IssuedAt.Format(time.RFC3339Nano), challenge.ExpiresAt.Format(time.RFC3339Nano))
	_, err = s.db.ExecContext(ctx, `INSERT INTO wallet_challenges (id, wallet_address, domain, origin, chain_id, nonce, message, issued_at, expires_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, challenge.ID, challenge.Address, challenge.Domain, challenge.Origin, challenge.ChainID, challenge.Nonce, challenge.Message, challenge.IssuedAt, challenge.ExpiresAt)
	return challenge, err
}

func (s *Service) VerifyWalletChallenge(ctx context.Context, challengeID, origin, signature string, sessionLifetime time.Duration) (BrowserSession, error) {
	if sessionLifetime <= 0 {
		return BrowserSession{}, errors.New("positive session lifetime is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return BrowserSession{}, err
	}
	defer tx.Rollback()
	var address, storedOrigin, message string
	var expiresAt time.Time
	var consumedAt sql.NullTime
	if err = tx.QueryRowContext(ctx, `SELECT wallet_address, origin, message, expires_at, consumed_at FROM wallet_challenges WHERE id=$1 FOR UPDATE`, challengeID).Scan(&address, &storedOrigin, &message, &expiresAt, &consumedAt); errors.Is(err, sql.ErrNoRows) {
		return BrowserSession{}, ErrInvalidCredential
	} else if err != nil {
		return BrowserSession{}, err
	}
	if storedOrigin != origin {
		return BrowserSession{}, ErrOriginDenied
	}
	if consumedAt.Valid {
		return BrowserSession{}, ErrAuthorizationConsumed
	}
	if !s.now().Before(expiresAt) {
		return BrowserSession{}, ErrAuthorizationExpired
	}
	if !validPersonalSignature(address, message, signature) {
		return BrowserSession{}, ErrInvalidCredential
	}
	var accountID string
	if err = tx.QueryRowContext(ctx, `SELECT id FROM accounts WHERE lower(wallet_address)=lower($1)`, address).Scan(&accountID); errors.Is(err, sql.ErrNoRows) {
		accountID, err = randomID("acct")
		if err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO accounts (id, wallet_address) VALUES ($1,$2)`, accountID, address)
		}
	}
	if err != nil {
		return BrowserSession{}, err
	}
	sessionID, err := randomID("session")
	if err != nil {
		return BrowserSession{}, err
	}
	secret, err := randomToken(32)
	if err != nil {
		return BrowserSession{}, err
	}
	now := s.now().UTC().Truncate(time.Microsecond)
	session := BrowserSession{ID: sessionID, Token: sessionID + "." + secret, AccountID: accountID, WalletAddress: address, ExpiresAt: now.Add(sessionLifetime)}
	if _, err = tx.ExecContext(ctx, `INSERT INTO browser_sessions (id, account_id, token_hash, expires_at) VALUES ($1,$2,$3,$4)`, session.ID, session.AccountID, digest(secret), session.ExpiresAt); err != nil {
		return BrowserSession{}, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE wallet_challenges SET consumed_at=$2 WHERE id=$1`, challengeID, now); err != nil {
		return BrowserSession{}, err
	}
	return session, tx.Commit()
}

func (s *Service) AuthenticateBrowserSession(ctx context.Context, token string) (BrowserSession, error) {
	id, secret, ok := splitCredential(token)
	if !ok {
		return BrowserSession{}, ErrInvalidCredential
	}
	var session BrowserSession
	var stored []byte
	err := s.db.QueryRowContext(ctx, `SELECT bs.id, bs.account_id, a.wallet_address, bs.token_hash, bs.expires_at FROM browser_sessions bs JOIN accounts a ON a.id=bs.account_id WHERE bs.id=$1 AND bs.revoked_at IS NULL AND bs.expires_at>$2`, id, s.now()).Scan(&session.ID, &session.AccountID, &session.WalletAddress, &stored, &session.ExpiresAt)
	if err != nil || subtle.ConstantTimeCompare(stored, digest(secret)) != 1 {
		return BrowserSession{}, ErrInvalidCredential
	}
	return session, nil
}

func (s *Service) PendingDevice(ctx context.Context, userCode string) (PendingDevice, error) {
	var pending PendingDevice
	var approved, exchanged sql.NullTime
	err := s.db.QueryRowContext(ctx, `SELECT machine_name, expires_at, approved_at, exchanged_at FROM device_authorizations WHERE user_code_hash=$1`, digest(strings.ToUpper(userCode))).Scan(&pending.MachineName, &pending.ExpiresAt, &approved, &exchanged)
	if errors.Is(err, sql.ErrNoRows) {
		return PendingDevice{}, ErrInvalidCredential
	}
	if err != nil {
		return PendingDevice{}, err
	}
	if !s.now().Before(pending.ExpiresAt) {
		return PendingDevice{}, ErrAuthorizationExpired
	}
	if approved.Valid || exchanged.Valid {
		return PendingDevice{}, ErrAuthorizationConsumed
	}
	return pending, nil
}

func (s *Service) ListAPIKeys(ctx context.Context, accountID string) ([]APIKeyRecord, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, scope_json, created_at FROM api_keys WHERE account_id=$1 AND revoked_at IS NULL ORDER BY created_at DESC`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := []APIKeyRecord{}
	for rows.Next() {
		var item APIKeyRecord
		var scope []byte
		if err := rows.Scan(&item.ID, &scope, &item.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(scope, &item.Scope); err != nil {
			return nil, err
		}
		keys = append(keys, item)
	}
	return keys, rows.Err()
}

func validPersonalSignature(address, message, encoded string) bool {
	raw, err := hex.DecodeString(strings.TrimPrefix(encoded, "0x"))
	if err != nil || len(raw) != crypto.SignatureLength {
		return false
	}
	if raw[64] >= 27 {
		raw[64] -= 27
	}
	publicKey, err := crypto.SigToPub(accounts.TextHash([]byte(message)), raw)
	return err == nil && strings.EqualFold(crypto.PubkeyToAddress(*publicKey).Hex(), address)
}
