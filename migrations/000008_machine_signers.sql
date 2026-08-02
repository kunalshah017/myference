ALTER TABLE device_authorizations ADD COLUMN IF NOT EXISTS signer_address text;
ALTER TABLE machines ADD COLUMN IF NOT EXISTS signer_address text;

CREATE UNIQUE INDEX IF NOT EXISTS machines_signer_address_unique
    ON machines (lower(signer_address))
    WHERE signer_address IS NOT NULL AND revoked_at IS NULL;

CREATE TABLE IF NOT EXISTS chain_provider_signers (
    chain_id bigint NOT NULL,
    contract_address text NOT NULL,
    provider text NOT NULL,
    signer text NOT NULL,
    allowed boolean NOT NULL,
    PRIMARY KEY (chain_id, contract_address, provider, signer)
);
