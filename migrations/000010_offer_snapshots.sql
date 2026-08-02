ALTER TABLE provider_routing_state ADD COLUMN IF NOT EXISTS offer_hash text;
ALTER TABLE provider_routing_state ADD COLUMN IF NOT EXISTS model_hash text;
ALTER TABLE provider_routing_state ADD COLUMN IF NOT EXISTS capability_hash text;

ALTER TABLE requests ADD COLUMN IF NOT EXISTS offer_hash text;
ALTER TABLE requests ADD COLUMN IF NOT EXISTS model_hash text;
ALTER TABLE requests ADD COLUMN IF NOT EXISTS capability_hash text;
