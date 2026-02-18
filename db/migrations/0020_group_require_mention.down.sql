-- 0020_group_require_mention (rollback)
ALTER TABLE bots DROP COLUMN IF EXISTS group_require_mention;
