ALTER TABLE provider_routing_state
    ADD COLUMN IF NOT EXISTS evidence_kind text NOT NULL DEFAULT 'provider_claimed',
    ADD COLUMN IF NOT EXISTS evidence_digest text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS metering_mode text NOT NULL DEFAULT 'tokens_and_compute';

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'provider_routing_state_evidence_kind_check' AND conrelid = 'provider_routing_state'::regclass) THEN
        ALTER TABLE provider_routing_state ADD CONSTRAINT provider_routing_state_evidence_kind_check
            CHECK (evidence_kind IN ('provider_claimed','ollama_digest','upstream_model','runtime_image'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'provider_routing_state_metering_mode_check' AND conrelid = 'provider_routing_state'::regclass) THEN
        ALTER TABLE provider_routing_state ADD CONSTRAINT provider_routing_state_metering_mode_check
            CHECK (metering_mode IN ('tokens_and_compute','compute_only'));
    END IF;
END $$;
