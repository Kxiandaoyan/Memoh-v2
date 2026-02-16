-- 0012_model_fallback
-- Add fallback_model_id column to models table for automatic model failover
ALTER TABLE models ADD COLUMN IF NOT EXISTS fallback_model_id UUID REFERENCES models(id) ON DELETE SET NULL;
