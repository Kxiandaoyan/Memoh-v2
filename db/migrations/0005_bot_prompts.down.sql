-- 0005_bot_prompts (rollback)
-- Remove identity, soul, task prompt fields and allow_self_evolution flag from bots table

ALTER TABLE bots DROP COLUMN IF EXISTS allow_self_evolution;
ALTER TABLE bots DROP COLUMN IF EXISTS task;
ALTER TABLE bots DROP COLUMN IF EXISTS soul;
ALTER TABLE bots DROP COLUMN IF EXISTS identity;
