-- 0023_ov_session_log_steps
-- Add OpenViking session completed/failed process log step enum values.
DO $$
BEGIN
  ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'openviking_session_completed';
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
  ALTER TYPE process_log_step ADD VALUE IF NOT EXISTS 'openviking_session_failed';
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
