CREATE TABLE IF NOT EXISTS chain_blocks (
    chain_id bigint NOT NULL,
    contract_address text NOT NULL,
    block_number bigint NOT NULL,
    block_hash text NOT NULL,
    parent_hash text NOT NULL,
    PRIMARY KEY (chain_id, contract_address, block_number)
);

CREATE TABLE IF NOT EXISTS chain_cursors (
    chain_id bigint NOT NULL,
    contract_address text NOT NULL,
    next_block bigint NOT NULL,
    last_block bigint,
    last_block_hash text,
    PRIMARY KEY (chain_id, contract_address)
);

CREATE TABLE IF NOT EXISTS chain_accounts (
    chain_id bigint NOT NULL,
    contract_address text NOT NULL,
    address text NOT NULL,
    customer_balance numeric(78,0) NOT NULL DEFAULT 0,
    provider_bond numeric(78,0) NOT NULL DEFAULT 0,
    claimable numeric(78,0) NOT NULL DEFAULT 0,
    PRIMARY KEY (chain_id, contract_address, address)
);

CREATE TABLE IF NOT EXISTS chain_offers (
    chain_id bigint NOT NULL,
    contract_address text NOT NULL,
    provider text NOT NULL,
    offer_id text NOT NULL,
    version bigint NOT NULL,
    model_hash text NOT NULL,
    capability_hash text NOT NULL,
    input_per_million numeric(78,0) NOT NULL,
    output_per_million numeric(78,0) NOT NULL,
    compute_per_second numeric(78,0) NOT NULL,
    PRIMARY KEY (chain_id, contract_address, provider, offer_id, version)
);

CREATE TABLE IF NOT EXISTS chain_sessions (
    chain_id bigint NOT NULL,
    contract_address text NOT NULL,
    session_id text NOT NULL,
    customer text NOT NULL,
    allowance numeric(78,0) NOT NULL,
    spent numeric(78,0) NOT NULL DEFAULT 0,
    expires_at bigint NOT NULL,
    PRIMARY KEY (chain_id, contract_address, session_id)
);

CREATE TABLE IF NOT EXISTS chain_settlements (
    chain_id bigint NOT NULL,
    contract_address text NOT NULL,
    request_id text NOT NULL,
    session_id text NOT NULL,
    provider text NOT NULL,
    provider_amount numeric(78,0) NOT NULL,
    fee_amount numeric(78,0) NOT NULL,
    transaction_hash text NOT NULL,
    PRIMARY KEY (chain_id, contract_address, request_id)
);

CREATE TABLE IF NOT EXISTS settlement_queue (
    request_id text PRIMARY KEY,
    receipt_json jsonb NOT NULL,
    provider_signature bytea NOT NULL,
    settlement_signature bytea NOT NULL,
    state text NOT NULL CHECK (state IN ('cosigned', 'broadcasting', 'settled', 'failed')),
    transaction_hash text,
    raw_transaction bytea,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE settlement_queue ADD COLUMN IF NOT EXISTS raw_transaction bytea;
