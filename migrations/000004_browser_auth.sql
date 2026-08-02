CREATE TABLE IF NOT EXISTS wallet_challenges (
    id text PRIMARY KEY,
    wallet_address text NOT NULL,
    domain text NOT NULL,
    origin text NOT NULL,
    chain_id bigint NOT NULL CHECK (chain_id > 0),
    nonce text NOT NULL UNIQUE,
    message text NOT NULL,
    issued_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    CHECK (expires_at > issued_at)
);

CREATE TABLE IF NOT EXISTS browser_sessions (
    id text PRIMARY KEY,
    account_id text NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS browser_sessions_account_idx
    ON browser_sessions (account_id, expires_at)
    WHERE revoked_at IS NULL;
