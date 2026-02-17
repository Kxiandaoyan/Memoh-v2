-- 0015_vlm_model (rollback)
-- Remove vlm_model_id column from bots table.

ALTER TABLE bots DROP COLUMN IF EXISTS vlm_model_id;
