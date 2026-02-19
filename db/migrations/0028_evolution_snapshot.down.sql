-- 0028_evolution_snapshot (down)
ALTER TABLE evolution_logs
  DROP COLUMN IF EXISTS files_snapshot;
