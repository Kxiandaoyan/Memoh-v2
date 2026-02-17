-- 0014_privileged_bot (rollback)
-- Remove is_privileged flag from bots table
ALTER TABLE bots DROP COLUMN IF EXISTS is_privileged;
