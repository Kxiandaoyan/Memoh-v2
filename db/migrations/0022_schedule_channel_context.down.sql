-- 0022_schedule_channel_context (rollback)

ALTER TABLE schedule DROP COLUMN IF EXISTS reply_target;
ALTER TABLE schedule DROP COLUMN IF EXISTS platform;
