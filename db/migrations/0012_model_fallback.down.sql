-- 0012_model_fallback
-- Remove fallback_model_id column from models table
ALTER TABLE models DROP COLUMN IF EXISTS fallback_model_id;
