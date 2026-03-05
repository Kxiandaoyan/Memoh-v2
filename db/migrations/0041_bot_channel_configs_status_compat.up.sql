-- 0041_bot_channel_configs_status_compat
-- Add status column for legacy bot_channel_configs rows that still use disabled.

DO $$
BEGIN
  -- Add status column if it doesn't exist
  IF NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'bot_channel_configs' AND column_name = 'status'
  ) THEN
    ALTER TABLE bot_channel_configs ADD COLUMN status TEXT;
  END IF;

  -- Migrate from disabled column only if it exists
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_name = 'bot_channel_configs' AND column_name = 'disabled'
  ) THEN
    UPDATE bot_channel_configs
    SET status = CASE
      WHEN status IS NOT NULL THEN status
      WHEN COALESCE(disabled, false) THEN 'disabled'
      WHEN verified_at IS NOT NULL THEN 'verified'
      ELSE 'pending'
    END
    WHERE status IS NULL;
  END IF;

  -- Set default for status column
  ALTER TABLE bot_channel_configs ALTER COLUMN status SET DEFAULT 'pending';

  -- Fill any remaining NULL values
  UPDATE bot_channel_configs SET status = 'pending' WHERE status IS NULL;

  -- Make status NOT NULL
  ALTER TABLE bot_channel_configs ALTER COLUMN status SET NOT NULL;

  -- Add constraint
  ALTER TABLE bot_channel_configs DROP CONSTRAINT IF EXISTS bot_channel_status_check;
  ALTER TABLE bot_channel_configs ADD CONSTRAINT bot_channel_status_check
    CHECK (status IN ('pending', 'verified', 'disabled'));
END $$;
