ALTER TABLE provider_routing_state ADD COLUMN IF NOT EXISTS input_per_million numeric(78,0) NOT NULL DEFAULT 0 CHECK (input_per_million >= 0);
ALTER TABLE provider_routing_state ADD COLUMN IF NOT EXISTS output_per_million numeric(78,0) NOT NULL DEFAULT 0 CHECK (output_per_million >= 0);
ALTER TABLE provider_routing_state ADD COLUMN IF NOT EXISTS compute_per_second numeric(78,0) NOT NULL DEFAULT 0 CHECK (compute_per_second >= 0);
ALTER TABLE provider_routing_state ADD COLUMN IF NOT EXISTS backend_kind text;

UPDATE provider_routing_state prs
SET input_per_million=o.input_per_million,
    output_per_million=o.output_per_million,
    compute_per_second=o.compute_per_second,
    backend_kind=b.kind,
    healthy=false,
    capacity=0
FROM offers o JOIN backends b ON b.id=o.backend_id
WHERE o.id=prs.offer_id AND o.version=prs.price_version
  AND (prs.backend_kind IS NULL OR (prs.input_per_million=0 AND prs.output_per_million=0 AND prs.compute_per_second=0));
