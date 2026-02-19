-- 0030_image_model
-- Add image_model_id for dedicated image generation model (e.g. Gemini Flash Image).
-- Falls back to resolving credentials from chat_model_id when NULL.
ALTER TABLE bots
  ADD COLUMN IF NOT EXISTS image_model_id UUID REFERENCES models(id) ON DELETE SET NULL;
