-- 0026_heartbeat_active_hours (down)
ALTER TABLE heartbeat_configs
  DROP COLUMN IF EXISTS active_hours_start,
  DROP COLUMN IF EXISTS active_hours_end,
  DROP COLUMN IF EXISTS active_days;
