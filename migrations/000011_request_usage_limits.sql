ALTER TABLE requests ADD COLUMN IF NOT EXISTS maximum_input_tokens bigint CHECK (maximum_input_tokens > 0);
ALTER TABLE requests ADD COLUMN IF NOT EXISTS maximum_output_tokens bigint CHECK (maximum_output_tokens > 0);
ALTER TABLE requests ADD COLUMN IF NOT EXISTS maximum_compute_milliseconds bigint CHECK (maximum_compute_milliseconds > 0);
