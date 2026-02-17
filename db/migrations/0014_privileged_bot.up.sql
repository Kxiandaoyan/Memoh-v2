-- 0014_privileged_bot
-- Add is_privileged flag to bots table for privileged bot management capability
ALTER TABLE bots ADD COLUMN IF NOT EXISTS is_privileged BOOLEAN NOT NULL DEFAULT false;
