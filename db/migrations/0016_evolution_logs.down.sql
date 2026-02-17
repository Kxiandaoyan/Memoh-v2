-- 0016_evolution_logs (rollback)
-- Drop evolution_logs table

DROP INDEX IF EXISTS idx_evolution_logs_bot_id_created_at;
DROP INDEX IF EXISTS idx_evolution_logs_bot_id;
DROP TABLE IF EXISTS evolution_logs;
