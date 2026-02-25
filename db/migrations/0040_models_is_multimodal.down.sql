-- 0040_models_is_multimodal (rollback)
-- Remove is_multimodal column from models table.

ALTER TABLE models
  DROP COLUMN IF EXISTS is_multimodal;
