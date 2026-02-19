-- 0028_evolution_snapshot
-- Add files_snapshot JSONB to evolution_logs to capture persona file contents
-- before each evolution run, enabling one-click rollback if evolution degrades behavior.

ALTER TABLE evolution_logs
  ADD COLUMN IF NOT EXISTS files_snapshot JSONB;
