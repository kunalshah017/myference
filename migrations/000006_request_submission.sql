ALTER TABLE requests DROP CONSTRAINT IF EXISTS requests_state_check;
ALTER TABLE requests ADD CONSTRAINT requests_state_check CHECK (state IN (
    'created', 'reserved', 'offered', 'accepted', 'streaming', 'completed',
    'signed', 'submitted', 'settled', 'rejected', 'expired', 'cancelled', 'failed'
));
