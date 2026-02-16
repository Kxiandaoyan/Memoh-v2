-- 0005_bot_prompts
-- Add identity, soul, task prompt fields and allow_self_evolution flag to bots table

ALTER TABLE bots ADD COLUMN IF NOT EXISTS identity TEXT DEFAULT NULL;
ALTER TABLE bots ADD COLUMN IF NOT EXISTS soul TEXT DEFAULT NULL;
ALTER TABLE bots ADD COLUMN IF NOT EXISTS task TEXT DEFAULT NULL;
ALTER TABLE bots ADD COLUMN IF NOT EXISTS allow_self_evolution BOOLEAN NOT NULL DEFAULT true;
