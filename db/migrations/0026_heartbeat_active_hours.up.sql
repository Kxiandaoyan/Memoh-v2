-- 0026_heartbeat_active_hours
-- Add configurable active hours and active days to per-bot heartbeat configs,
-- enabling bots to only fire during business hours or in specific time windows.
-- All values use the global timezone setting stored in global_settings.

ALTER TABLE heartbeat_configs
  ADD COLUMN IF NOT EXISTS active_hours_start SMALLINT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS active_hours_end   SMALLINT NOT NULL DEFAULT 23,
  ADD COLUMN IF NOT EXISTS active_days        SMALLINT[] NOT NULL DEFAULT '{0,1,2,3,4,5,6}';

COMMENT ON COLUMN heartbeat_configs.active_hours_start IS '0-23: hour of day (inclusive) at which heartbeat becomes active';
COMMENT ON COLUMN heartbeat_configs.active_hours_end   IS '0-23: hour of day (inclusive) at which heartbeat stops firing';
COMMENT ON COLUMN heartbeat_configs.active_days        IS 'ISO weekday numbers (0=Sunday … 6=Saturday) on which the heartbeat is active';
