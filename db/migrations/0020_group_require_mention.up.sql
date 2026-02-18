-- 0020_group_require_mention
-- Add group_require_mention column to bots table for controlling whether bot requires @mention in group chats
ALTER TABLE bots ADD COLUMN IF NOT EXISTS group_require_mention BOOLEAN NOT NULL DEFAULT true;
