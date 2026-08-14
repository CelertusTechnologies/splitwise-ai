CREATE UNIQUE INDEX users_phone_number_active_unique ON users (phone_number) WHERE deleted_at IS NULL AND phone_number IS NOT NULL;

CREATE TABLE phone_otp_codes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  phone_number TEXT NOT NULL,
  code_hash TEXT NOT NULL,
  attempt_count INTEGER NOT NULL DEFAULT 0,
  expires_at TIMESTAMPTZ NOT NULL,
  consumed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX phone_otp_codes_phone_created_idx ON phone_otp_codes (phone_number, created_at DESC);
