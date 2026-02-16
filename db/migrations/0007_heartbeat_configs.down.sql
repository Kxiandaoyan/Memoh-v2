-- 0007_heartbeat_configs (rollback)
-- Drop the heartbeat_configs table.

DROP INDEX IF EXISTS idx_heartbeat_configs_enabled;
DROP INDEX IF EXISTS idx_heartbeat_configs_bot_id;
DROP TABLE IF EXISTS heartbeat_configs;
