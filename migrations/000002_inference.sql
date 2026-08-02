ALTER TABLE sessions ADD COLUMN IF NOT EXISTS confirmed_balance_wei numeric(78,0) NOT NULL DEFAULT 0 CHECK (confirmed_balance_wei >= 0);

ALTER TABLE requests ADD COLUMN IF NOT EXISTS machine_id text REFERENCES machines(id);
ALTER TABLE requests ADD COLUMN IF NOT EXISTS offer_id text;
ALTER TABLE requests ADD COLUMN IF NOT EXISTS price_version bigint CHECK (price_version > 0);
ALTER TABLE requests ADD COLUMN IF NOT EXISTS maximum_spend numeric(78,0) CHECK (maximum_spend > 0);

CREATE TABLE IF NOT EXISTS provider_routing_state (
    machine_id text NOT NULL REFERENCES machines(id) ON DELETE CASCADE,
    offer_id text NOT NULL,
    model text NOT NULL,
    capabilities text[] NOT NULL,
    price_version bigint NOT NULL CHECK (price_version > 0),
    confirmed_bond boolean NOT NULL DEFAULT false,
    healthy boolean NOT NULL DEFAULT false,
    capacity integer NOT NULL DEFAULT 0 CHECK (capacity >= 0),
    maximum_cost numeric(78,0) NOT NULL CHECK (maximum_cost >= 0),
    latency_milliseconds bigint NOT NULL DEFAULT 0 CHECK (latency_milliseconds >= 0),
    success_basis_points integer NOT NULL DEFAULT 0 CHECK (success_basis_points BETWEEN 0 AND 10000),
    reputation bigint NOT NULL DEFAULT 0 CHECK (reputation >= 0),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (machine_id, offer_id),
    FOREIGN KEY (offer_id, price_version) REFERENCES offers(id, version)
);

CREATE INDEX IF NOT EXISTS provider_routing_model_idx ON provider_routing_state (model) WHERE confirmed_bond AND healthy AND capacity > 0;

CREATE TABLE IF NOT EXISTS inference_reservations (
    request_id text PRIMARY KEY REFERENCES requests(id) ON DELETE CASCADE,
    session_id text NOT NULL REFERENCES sessions(id),
    amount numeric(78,0) NOT NULL CHECK (amount > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    released_at timestamptz
);

CREATE INDEX IF NOT EXISTS inference_reservations_active_idx ON inference_reservations (session_id) WHERE released_at IS NULL;

CREATE TABLE IF NOT EXISTS receipt_proposals (
    request_id text PRIMARY KEY REFERENCES requests(id) ON DELETE CASCADE,
    session_id text NOT NULL REFERENCES sessions(id),
    machine_id text NOT NULL REFERENCES machines(id),
    offer_id text NOT NULL,
    model text NOT NULL,
    price_version bigint NOT NULL CHECK (price_version > 0),
    input_tokens bigint NOT NULL CHECK (input_tokens >= 0),
    output_tokens bigint NOT NULL CHECK (output_tokens >= 0),
    compute_milliseconds bigint NOT NULL CHECK (compute_milliseconds >= 0),
    input_hash bytea NOT NULL CHECK (octet_length(input_hash) = 32),
    output_hash bytea NOT NULL CHECK (octet_length(output_hash) = 32),
    completed_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (offer_id, price_version) REFERENCES offers(id, version)
);
