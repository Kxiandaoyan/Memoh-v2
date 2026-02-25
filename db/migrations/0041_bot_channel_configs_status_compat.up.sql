-- 0041_bot_channel_configs_status_compat
-- Add status column for legacy bot_channel_configs rows that still use disabled.

ALTER TABLE bot_channel_configs
  ADD COLUMN IF NOT EXISTS status TEXT;

UPDATE bot_channel_configs
SET status = CASE
  WHEN status IS NOT NULL THEN status
  WHEN COALESCE(disabled, false) THEN 'disabled'
  WHEN verified_at IS NOT NULL THEN 'verified'
  ELSE 'pending'
END
WHERE status IS NULL;

ALTER TABLE bot_channel_configs
  ALTER COLUMN status SET DEFAULT 'pending';

UPDATE bot_channel_configs
SET status = 'pending'
WHERE status IS NULL;

ALTER TABLE bot_channel_configs
  ALTER COLUMN status SET NOT NULL;

ALTER TABLE bot_channel_configs
  DROP CONSTRAINT IF EXISTS bot_channel_status_check;

ALTER TABLE bot_channel_configs
  ADD CONSTRAINT bot_channel_status_check
  CHECK (status IN ('pending', 'verified', 'disabled'));
