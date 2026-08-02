CREATE TABLE IF NOT EXISTS accounts (
    id text PRIMARY KEY,
    wallet_address text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS api_keys (
    id text PRIMARY KEY,
    account_id text NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    key_hash bytea NOT NULL UNIQUE,
    scopes text[] NOT NULL DEFAULT '{}',
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS machines (
    id text PRIMARY KEY,
    account_id text NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (account_id, name)
);

CREATE TABLE IF NOT EXISTS backends (
    id text PRIMARY KEY,
    machine_id text NOT NULL REFERENCES machines(id) ON DELETE CASCADE,
    kind text NOT NULL,
    model text NOT NULL,
    enabled boolean NOT NULL DEFAULT true,
    UNIQUE (machine_id, kind, model)
);

CREATE TABLE IF NOT EXISTS offers (
    id text NOT NULL,
    backend_id text NOT NULL REFERENCES backends(id) ON DELETE CASCADE,
    version bigint NOT NULL CHECK (version > 0),
    input_per_million numeric(78,0) NOT NULL CHECK (input_per_million >= 0),
    output_per_million numeric(78,0) NOT NULL CHECK (output_per_million >= 0),
    compute_per_second numeric(78,0) NOT NULL CHECK (compute_per_second >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (id, version),
    UNIQUE (backend_id, id, version)
);

CREATE TABLE IF NOT EXISTS sessions (
    id text PRIMARY KEY,
    account_id text NOT NULL REFERENCES accounts(id),
    state text NOT NULL CHECK (state IN ('open', 'closing', 'closed')),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS requests (
    id text PRIMARY KEY,
    session_id text NOT NULL REFERENCES sessions(id),
    state text NOT NULL CHECK (state IN (
        'created', 'reserved', 'offered', 'accepted', 'streaming', 'completed',
        'signed', 'settled', 'rejected', 'expired', 'cancelled', 'failed'
    )),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS receipt_nonces (
    provider_address text NOT NULL,
    nonce bigint NOT NULL CHECK (nonce > 0),
    request_id text NOT NULL REFERENCES requests(id),
    PRIMARY KEY (provider_address, nonce)
);

CREATE TABLE IF NOT EXISTS chain_logs (
    chain_id bigint NOT NULL,
    contract_address text NOT NULL,
    block_number bigint NOT NULL,
    block_hash text NOT NULL,
    transaction_hash text NOT NULL,
    log_index integer NOT NULL,
    payload jsonb NOT NULL,
    PRIMARY KEY (chain_id, contract_address, transaction_hash, log_index)
);

CREATE TABLE IF NOT EXISTS outbox (
    id bigserial PRIMARY KEY,
    aggregate_type text NOT NULL,
    aggregate_id text NOT NULL,
    event_type text NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);

CREATE INDEX IF NOT EXISTS outbox_unpublished_idx ON outbox (id) WHERE published_at IS NULL;
