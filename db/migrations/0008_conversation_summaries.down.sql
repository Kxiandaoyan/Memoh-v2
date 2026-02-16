-- 0008_conversation_summaries (rollback)
-- Drop the conversation_summaries table.

DROP INDEX IF EXISTS idx_conversation_summaries_bot_chat;
DROP TABLE IF EXISTS conversation_summaries;
