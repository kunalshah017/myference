ALTER TABLE inference_reservations DROP CONSTRAINT IF EXISTS inference_reservations_amount_check;
ALTER TABLE inference_reservations ADD CONSTRAINT inference_reservations_amount_check CHECK (amount >= 0);
