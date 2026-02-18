-- 0022_schedule_channel_context
-- Store the originating platform and reply target so schedule-fired
-- commands (e.g. send) know where to deliver messages.

ALTER TABLE schedule ADD COLUMN IF NOT EXISTS platform TEXT NOT NULL DEFAULT '';
ALTER TABLE schedule ADD COLUMN IF NOT EXISTS reply_target TEXT NOT NULL DEFAULT '';
