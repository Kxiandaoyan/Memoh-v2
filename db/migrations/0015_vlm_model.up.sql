-- 0015_vlm_model
-- Add vlm_model_id column to bots table for independent VLM model selection in ov.conf.

ALTER TABLE bots ADD COLUMN IF NOT EXISTS vlm_model_id UUID REFERENCES models(id) ON DELETE SET NULL;
