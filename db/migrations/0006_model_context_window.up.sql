-- 0006_model_context_window
-- Add context_window column to models table for token budget calculation.
ALTER TABLE models ADD COLUMN IF NOT EXISTS context_window INTEGER NOT NULL DEFAULT 128000;
