-- Allow re-creating subagents with the same name after soft-delete.
-- The old UNIQUE constraint covers all rows including deleted ones.
ALTER TABLE subagents DROP CONSTRAINT IF EXISTS subagents_name_unique;

CREATE UNIQUE INDEX IF NOT EXISTS subagents_name_unique_active
  ON subagents (bot_id, name) WHERE deleted = false;
