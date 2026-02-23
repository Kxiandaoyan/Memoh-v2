DROP INDEX IF EXISTS subagents_name_unique_active;

ALTER TABLE subagents ADD CONSTRAINT subagents_name_unique UNIQUE (bot_id, name);
