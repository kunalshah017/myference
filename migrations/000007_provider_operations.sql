ALTER TABLE chain_accounts ADD COLUMN IF NOT EXISTS bond_exit_available_at bigint NOT NULL DEFAULT 0;
ALTER TABLE chain_sessions ADD COLUMN IF NOT EXISTS close_available_at bigint NOT NULL DEFAULT 0;
ALTER TABLE chain_sessions ADD COLUMN IF NOT EXISTS finalized boolean NOT NULL DEFAULT false;
