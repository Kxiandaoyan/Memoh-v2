-- 0041_bot_channel_configs_status_compat (rollback)
-- Remove bot_channel_configs.status compatibility column.

ALTER TABLE bot_channel_configs
  DROP CONSTRAINT IF EXISTS bot_channel_status_check;

ALTER TABLE bot_channel_configs
  DROP COLUMN IF EXISTS status;
