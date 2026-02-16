-- 0009_bot_enable_openviking (rollback)

ALTER TABLE bots DROP COLUMN IF EXISTS enable_openviking;
