-- 0006_model_context_window (rollback)
-- Remove context_window column from models table.
ALTER TABLE models DROP COLUMN IF EXISTS context_window;
