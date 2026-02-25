-- 0040_models_is_multimodal
-- Add missing is_multimodal column for upgraded databases.

ALTER TABLE models
  ADD COLUMN IF NOT EXISTS is_multimodal BOOLEAN NOT NULL DEFAULT false;
