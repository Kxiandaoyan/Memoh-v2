-- 0029_background_model
-- Add background_model_id for cheap model routing on background tasks
-- (heartbeats, scheduled tasks, subagents). Falls back to chat_model_id when NULL.
ALTER TABLE bots
  ADD COLUMN IF NOT EXISTS background_model_id UUID REFERENCES models(id) ON DELETE SET NULL;
