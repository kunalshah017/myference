ALTER TABLE chain_settlements ADD COLUMN IF NOT EXISTS block_number bigint NOT NULL DEFAULT 0;
ALTER TABLE chain_settlements ADD COLUMN IF NOT EXISTS indexed_at timestamptz NOT NULL DEFAULT now();

CREATE INDEX IF NOT EXISTS chain_settlements_provider_time_idx
    ON chain_settlements (chain_id, contract_address, lower(provider), indexed_at DESC);

CREATE TABLE IF NOT EXISTS chain_slashes (
    chain_id bigint NOT NULL,
    contract_address text NOT NULL,
    request_id text NOT NULL,
    provider text NOT NULL,
    amount numeric(78,0) NOT NULL CHECK (amount > 0),
    block_number bigint NOT NULL,
    transaction_hash text NOT NULL,
    indexed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (chain_id, contract_address, request_id, provider, transaction_hash)
);

CREATE INDEX IF NOT EXISTS chain_slashes_provider_time_idx
    ON chain_slashes (chain_id, contract_address, lower(provider), indexed_at DESC);
