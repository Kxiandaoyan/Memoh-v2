-- 0009_bot_enable_openviking
-- Per-bot toggle to enable OpenViking context database.

ALTER TABLE bots ADD COLUMN IF NOT EXISTS enable_openviking BOOLEAN NOT NULL DEFAULT false;
