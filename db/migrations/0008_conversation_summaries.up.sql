-- 0008_conversation_summaries
-- Per-chat conversation summaries for context compression.

CREATE TABLE IF NOT EXISTS conversation_summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id UUID NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    chat_id TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    message_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT conversation_summaries_unique UNIQUE (bot_id, chat_id)
);

CREATE INDEX IF NOT EXISTS idx_conversation_summaries_bot_chat ON conversation_summaries(bot_id, chat_id);
